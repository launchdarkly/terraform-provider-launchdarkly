package launchdarkly

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                = &ExperimentationSettingsResource{}
	_ resource.ResourceWithImportState = &ExperimentationSettingsResource{}
)

type ExperimentationSettingsResource struct {
	client *Client
}

type ExperimentationSettingsResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ProjectKey         types.String `tfsdk:"project_key"`
	RandomizationUnits types.List   `tfsdk:"randomization_units"`
}

// randomizationUnitAttrTypes describes one entry in the randomization_units list.
var randomizationUnitAttrTypes = map[string]attr.Type{
	RANDOMIZATION_UNIT: types.StringType,
	DEFAULT:            types.BoolType,
}

func NewExperimentationSettingsResource() resource.Resource {
	return &ExperimentationSettingsResource{}
}

func (r *ExperimentationSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_experimentation_settings"
}

func (r *ExperimentationSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly experimentation settings resource.\n\nThis resource lets you configure the randomization units used for experiments in a project. There is exactly one experimentation settings object per project, so this resource behaves as a singleton: its `id` is the project key, and destroying it only removes it from Terraform state (it does not reset the project's randomization units).\n\nTo learn more about experiment allocation, read [Allocating experiment audiences](https://launchdarkly.com/docs/home/experimentation/allocation).",
		Attributes: map[string]schema.Attribute{
			ID: schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of this resource. Equal to the project key.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The project key. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			RANDOMIZATION_UNITS: schema.ListNestedAttribute{
				Required:    true,
				Description: "An ordered list of the randomization units allowed for experiments in this project. Each entry must reference the key of an existing context kind in the project.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						RANDOMIZATION_UNIT: schema.StringAttribute{
							Required:    true,
							Description: "The unit of randomization. Must match the key of an existing context kind in this project.",
						},
						DEFAULT: schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Whether new experiment iterations in this project default to using this randomization unit. A project can only have one default randomization unit. Defaults to `false`.",
						},
					},
				},
			},
		},
	}
}

func (r *ExperimentationSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

type randomizationUnitModel struct {
	RandomizationUnit types.String `tfsdk:"randomization_unit"`
	Default           types.Bool   `tfsdk:"default"`
}

func (r *ExperimentationSettingsResource) put(ctx context.Context, plan *ExperimentationSettingsResourceModel, diags *diag.Diagnostics) {
	var units []randomizationUnitModel
	diags.Append(plan.RandomizationUnits.ElementsAs(ctx, &units, false)...)
	if diags.HasError() {
		return
	}

	inputs := make([]ldapi.RandomizationUnitInput, 0, len(units))
	for _, u := range units {
		input := ldapi.RandomizationUnitInput{RandomizationUnit: u.RandomizationUnit.ValueString()}
		if !u.Default.IsNull() && !u.Default.IsUnknown() {
			d := u.Default.ValueBool()
			input.Default = &d
		}
		inputs = append(inputs, input)
	}

	body := ldapi.RandomizationSettingsPut{RandomizationUnits: inputs}
	projectKey := plan.ProjectKey.ValueString()
	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.ExperimentsApi.PutExperimentationSettings(r.client.ctx, projectKey).RandomizationSettingsPut(body).Execute()
		return e
	})
	if err != nil {
		addLdapiError(diags, fmt.Sprintf("Failed to update experimentation settings for project %q", projectKey), err)
	}
}

func (r *ExperimentationSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ExperimentationSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.put(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(plan.ProjectKey.ValueString())
	r.readIntoModel(ctx, plan.ProjectKey.ValueString(), &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ExperimentationSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ExperimentationSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readIntoModel(ctx, data.ProjectKey.ValueString(), &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ExperimentationSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ExperimentationSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.put(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(plan.ProjectKey.ValueString())
	r.readIntoModel(ctx, plan.ProjectKey.ValueString(), &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete is a no-op beyond removing the resource from state. Experimentation
// settings always exist for a project; there is no API to delete them.
func (r *ExperimentationSettingsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *ExperimentationSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(ID), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), req.ID)...)
}

func (r *ExperimentationSettingsResource) readIntoModel(
	ctx context.Context,
	projectKey string,
	data *ExperimentationSettingsResourceModel,
	diags *diag.Diagnostics,
) {
	var settings *ldapi.RandomizationSettingsRep
	var res *http.Response
	err := r.client.withConcurrency(r.client.ctx, func() error {
		var e error
		settings, res, e = r.client.ld.ExperimentsApi.GetExperimentationSettings(r.client.ctx, projectKey).Execute()
		return e
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError(fmt.Sprintf("Failed to get experimentation settings for project %q", projectKey), handleLdapiErr(err).Error())
		return
	}

	data.ID = types.StringValue(projectKey)
	data.ProjectKey = types.StringValue(projectKey)

	objType := types.ObjectType{AttrTypes: randomizationUnitAttrTypes}
	elems := make([]attr.Value, 0, len(settings.RandomizationUnits))
	for _, u := range settings.RandomizationUnits {
		// Hidden units are system-managed defaults the user does not configure.
		if u.Hidden != nil && *u.Hidden {
			continue
		}
		obj, d := types.ObjectValue(randomizationUnitAttrTypes, map[string]attr.Value{
			RANDOMIZATION_UNIT: stringValueFromPointer(u.RandomizationUnit),
			DEFAULT:            types.BoolValue(u.Default != nil && *u.Default),
		})
		diags.Append(d...)
		elems = append(elems, obj)
	}
	list, d := types.ListValue(objType, elems)
	diags.Append(d...)
	data.RandomizationUnits = list
}
