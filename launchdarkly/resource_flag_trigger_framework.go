package launchdarkly

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var (
	_ resource.Resource                = &FlagTriggerResource{}
	_ resource.ResourceWithImportState = &FlagTriggerResource{}
)

type FlagTriggerResource struct {
	client *Client
}

type FlagTriggerResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectKey     types.String `tfsdk:"project_key"`
	EnvKey         types.String `tfsdk:"env_key"`
	FlagKey        types.String `tfsdk:"flag_key"`
	IntegrationKey types.String `tfsdk:"integration_key"`
	Instructions   types.List   `tfsdk:"instructions"`
	TriggerURL     types.String `tfsdk:"trigger_url"`
	MaintainerID   types.String `tfsdk:"maintainer_id"`
	Enabled        types.Bool   `tfsdk:"enabled"`
}

func NewFlagTriggerResource() resource.Resource {
	return &FlagTriggerResource{}
}

func (r *FlagTriggerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flag_trigger"
}

func (r *FlagTriggerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly flag trigger resource (Enterprise plan only).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The project key.",
				Validators:  []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			ENV_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The environment key.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			FLAG_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The flag key.",
				Validators:  []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			INTEGRATION_KEY: schema.StringAttribute{
				Required:    true,
				Description: fmt.Sprintf("The integration key. Supported: %s.", oxfordCommaJoin(VALID_TRIGGER_INTEGRATIONS)),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			TRIGGER_URL: schema.StringAttribute{
				Computed:      true,
				Sensitive:     true,
				Description:   "The unique URL used to invoke the trigger. Exposed once.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			MAINTAINER_ID: schema.StringAttribute{
				Computed:      true,
				Description:   "Member ID maintaining the trigger.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			ENABLED: schema.BoolAttribute{
				Required:    true,
				Description: "Whether the trigger is enabled.",
			},
		},
		Blocks: map[string]schema.Block{
			INSTRUCTIONS: schema.ListNestedBlock{
				Description: "Instructions specifying the action to perform.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						KIND: schema.StringAttribute{
							Required:    true,
							Description: "Currently `turnFlagOn` or `turnFlagOff`.",
						},
					},
				},
			},
		},
	}
}

func (r *FlagTriggerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

type flagTriggerInstructionModel struct {
	Kind string `tfsdk:"kind"`
}

func (r *FlagTriggerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FlagTriggerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	envKey := plan.EnvKey.ValueString()
	flagKey := plan.FlagKey.ValueString()
	integrationKey := plan.IntegrationKey.ValueString()

	var ins []flagTriggerInstructionModel
	resp.Diagnostics.Append(plan.Instructions.ElementsAs(ctx, &ins, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	postInstructions := make([]map[string]interface{}, 0, len(ins))
	for _, in := range ins {
		postInstructions = append(postInstructions, map[string]interface{}{KIND: in.Kind})
	}

	triggerBody := ldapi.NewTriggerPost(integrationKey)
	triggerBody.Instructions = postInstructions

	var createdTrigger *ldapi.TriggerWorkflowRep
	err := r.client.withConcurrency(r.client.ctx, func() error {
		var e error
		createdTrigger, _, e = r.client.ld.FlagTriggersApi.CreateTriggerWorkflow(r.client.ctx, projectKey, envKey, flagKey).TriggerPost(*triggerBody).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to create flag trigger", err)
		return
	}
	if createdTrigger.Id == nil {
		resp.Diagnostics.AddError("Missing trigger ID", "API returned a trigger without an ID.")
		return
	}
	plan.ID = types.StringValue(*createdTrigger.Id)
	plan.TriggerURL = stringValueFromPointer(createdTrigger.TriggerURL)

	// SDKv2: if enabled=false at create, follow up with a PATCH because
	// the create endpoint does not accept multiple instructions.
	if !plan.Enabled.ValueBool() {
		input := ldapi.FlagTriggerInput{
			Instructions: []map[string]interface{}{{KIND: "disableTrigger"}},
		}
		if e := r.client.withConcurrency(r.client.ctx, func() error {
			_, _, ee := r.client.ld.FlagTriggersApi.PatchTriggerWorkflow(r.client.ctx, projectKey, envKey, flagKey, *createdTrigger.Id).FlagTriggerInput(input).Execute()
			return ee
		}); e != nil {
			addLdapiError(&resp.Diagnostics, "Failed to disable trigger after creation", e)
			return
		}
	}

	r.readIntoModel(ctx, projectKey, envKey, flagKey, *createdTrigger.Id, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FlagTriggerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FlagTriggerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readIntoModel(ctx, data.ProjectKey.ValueString(), data.EnvKey.ValueString(), data.FlagKey.ValueString(), data.ID.ValueString(), &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FlagTriggerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state FlagTriggerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var ins []flagTriggerInstructionModel
	resp.Diagnostics.Append(plan.Instructions.ElementsAs(ctx, &ins, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var patchInstructions []map[string]interface{}
	if !plan.Instructions.Equal(state.Instructions) {
		for _, in := range ins {
			patchInstructions = append(patchInstructions, map[string]interface{}{
				KIND: "replaceTriggerActionInstructions",
				VALUE: []map[string]interface{}{{
					KIND: in.Kind,
				}},
			})
		}
	}

	if !plan.Enabled.Equal(state.Enabled) {
		if plan.Enabled.ValueBool() {
			patchInstructions = append(patchInstructions, map[string]interface{}{KIND: "enableTrigger"})
		} else {
			patchInstructions = append(patchInstructions, map[string]interface{}{KIND: "disableTrigger"})
		}
	}

	if len(patchInstructions) > 0 {
		input := ldapi.FlagTriggerInput{Instructions: patchInstructions}
		err := r.client.withConcurrency(r.client.ctx, func() error {
			_, _, e := r.client.ld.FlagTriggersApi.PatchTriggerWorkflow(r.client.ctx, plan.ProjectKey.ValueString(), plan.EnvKey.ValueString(), plan.FlagKey.ValueString(), plan.ID.ValueString()).FlagTriggerInput(input).Execute()
			return e
		})
		if err != nil {
			addLdapiError(&resp.Diagnostics, "Failed to update flag trigger", err)
			return
		}
	}

	r.readIntoModel(ctx, plan.ProjectKey.ValueString(), plan.EnvKey.ValueString(), plan.FlagKey.ValueString(), plan.ID.ValueString(), &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FlagTriggerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FlagTriggerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, e := r.client.ld.FlagTriggersApi.DeleteTriggerWorkflow(r.client.ctx, data.ProjectKey.ValueString(), data.EnvKey.ValueString(), data.FlagKey.ValueString(), data.ID.ValueString()).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to delete flag trigger", err)
	}
}

func (r *FlagTriggerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if strings.Count(req.ID, "/") != 3 {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("expected project_key/env_key/flag_key/trigger_id, got %q", req.ID))
		return
	}
	parts := strings.SplitN(req.ID, "/", 4)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(ENV_KEY), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(FLAG_KEY), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[3])...)
}

func (r *FlagTriggerResource) readIntoModel(
	ctx context.Context,
	projectKey, envKey, flagKey, triggerID string,
	data *FlagTriggerResourceModel,
	diags interface{ AddError(string, string) },
) {
	var trigger *ldapi.TriggerWorkflowRep
	var res *http.Response
	var err error
	err = r.client.withConcurrency(r.client.ctx, func() error {
		trigger, res, err = r.client.ld.FlagTriggersApi.GetTriggerWorkflowById(r.client.ctx, projectKey, flagKey, envKey, triggerID).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError("Failed to get flag trigger", handleLdapiErr(err).Error())
		return
	}
	if trigger.Id == nil {
		data.ID = types.StringNull()
		return
	}
	data.ID = types.StringValue(*trigger.Id)
	data.ProjectKey = types.StringValue(projectKey)
	data.EnvKey = types.StringValue(envKey)
	data.FlagKey = types.StringValue(flagKey)
	if trigger.IntegrationKey != nil {
		data.IntegrationKey = types.StringValue(*trigger.IntegrationKey)
	}
	if trigger.MaintainerId != nil {
		data.MaintainerID = types.StringValue(*trigger.MaintainerId)
	} else {
		data.MaintainerID = types.StringValue("")
	}
	if trigger.Enabled != nil {
		data.Enabled = types.BoolValue(*trigger.Enabled)
	}
	// Don't refresh TRIGGER_URL — it's only exposed at create.

	// instructions
	instObjType := types.ObjectType{AttrTypes: flagTriggerInstructionAttrTypes}
	elems := make([]attr.Value, 0, len(trigger.Instructions))
	for _, instr := range trigger.Instructions {
		kindVal := types.StringNull()
		if k, ok := instr[KIND].(string); ok {
			kindVal = types.StringValue(k)
		}
		obj, _ := types.ObjectValue(flagTriggerInstructionAttrTypes, map[string]attr.Value{KIND: kindVal})
		elems = append(elems, obj)
	}
	list, _ := types.ListValue(instObjType, elems)
	data.Instructions = list
}
