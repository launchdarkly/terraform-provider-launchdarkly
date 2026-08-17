package launchdarkly

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v23"
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
	RandomizationUnits types.Map    `tfsdk:"randomization_units"`
}

// experimentationSettingsUnitAttrTypes describes the object stored in each
// randomization_units map value. The map key is the randomization unit itself,
// so it is not duplicated inside the object.
var experimentationSettingsUnitAttrTypes = map[string]attr.Type{
	DEFAULT:      types.BoolType,
	DISPLAY_NAME: types.StringType,
	HIDDEN:       types.BoolType,
}

func NewExperimentationSettingsResource() resource.Resource {
	return &ExperimentationSettingsResource{}
}

func (r *ExperimentationSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_experimentation_settings"
}

func (r *ExperimentationSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly experimentation settings resource.\n\nThis resource lets you configure which randomization units are available for experiments within a LaunchDarkly project. Each randomization unit must correspond to an existing context kind in the project. Exactly one unit must be marked as the default.\n\nThere is only one set of experimentation settings per project, so this resource behaves as a singleton keyed by `project_key`. Destroying this resource only removes it from Terraform state; the project's experimentation settings remain unchanged in LaunchDarkly.\n\nTo learn more, read [Experimentation Documentation](https://launchdarkly.com/docs/home/experimentation).",
		Attributes: map[string]schema.Attribute{
			ID: schema.StringAttribute{
				Computed:      true,
				Description:   "The unique resource ID, which is equal to the `project_key`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:      true,
				Description:   addForceNewDescription("The project key. Experimentation settings are configured per project.", true),
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			RANDOMIZATION_UNITS: schema.MapNestedAttribute{
				Required:    true,
				Description: "The randomization units available for experiments in this project, keyed by the randomization unit. Each key must match the key of an existing context kind in the project.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						DEFAULT: schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Whether this randomization unit is the default for experiments in the project. Exactly one randomization unit must be the default.",
						},
						DISPLAY_NAME: schema.StringAttribute{
							Computed:    true,
							Description: "The display name for the randomization unit, shown in the LaunchDarkly user interface.",
						},
						HIDDEN: schema.BoolAttribute{
							Computed:    true,
							Description: "Whether this randomization unit is hidden in the LaunchDarkly user interface.",
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

// unitInputModel is the decoded shape of a single randomization_units map value.
type experimentationSettingsUnitModel struct {
	Default     types.Bool   `tfsdk:"default"`
	DisplayName types.String `tfsdk:"display_name"`
	Hidden      types.Bool   `tfsdk:"hidden"`
}

func (r *ExperimentationSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ExperimentationSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	if exists, err := projectExists(projectKey, r.client); !exists {
		if err != nil {
			resp.Diagnostics.AddError("Failed to check project", err.Error())
			return
		}
		resp.Diagnostics.AddError("Project not found", fmt.Sprintf("cannot find project with key %q", projectKey))
		return
	}

	if err := r.putSettings(ctx, &plan, &resp.Diagnostics); err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error creating experimentation settings for project %q", projectKey), err)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(projectKey)
	r.readIntoModel(ctx, projectKey, &plan, &resp.Diagnostics)
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

	projectKey := plan.ProjectKey.ValueString()
	if err := r.putSettings(ctx, &plan, &resp.Diagnostics); err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error updating experimentation settings for project %q", projectKey), err)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(projectKey)
	r.readIntoModel(ctx, projectKey, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete is a no-op against the API. LaunchDarkly does not expose an endpoint to
// remove a project's experimentation settings, so destroying this resource only
// removes it from Terraform state.
func (r *ExperimentationSettingsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *ExperimentationSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(ID), req.ID)...)
}

// putSettings converts the planned randomization_units map into the API input
// and issues the PUT. The units are sorted by key so the request payload is
// deterministic.
func (r *ExperimentationSettingsResource) putSettings(ctx context.Context, plan *ExperimentationSettingsResourceModel, diags *diag.Diagnostics) error {
	units := make(map[string]experimentationSettingsUnitModel, len(plan.RandomizationUnits.Elements()))
	diags.Append(plan.RandomizationUnits.ElementsAs(ctx, &units, false)...)
	if diags.HasError() {
		return nil
	}

	keys := make([]string, 0, len(units))
	for k := range units {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	inputs := make([]ldapi.RandomizationUnitInput, 0, len(keys))
	for _, k := range keys {
		u := units[k]
		input := ldapi.NewRandomizationUnitInput(k)
		if !u.Default.IsNull() && !u.Default.IsUnknown() {
			d := u.Default.ValueBool()
			input.Default = &d
		}
		inputs = append(inputs, *input)
	}

	body := ldapi.RandomizationSettingsPut{RandomizationUnits: inputs}
	return r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.ExperimentsApi.PutExperimentationSettings(r.client.ctx, plan.ProjectKey.ValueString()).RandomizationSettingsPut(body).Execute()
		return e
	})
}

func (r *ExperimentationSettingsResource) readIntoModel(
	ctx context.Context,
	projectKey string,
	data *ExperimentationSettingsResourceModel,
	diags *diag.Diagnostics,
) {
	var settings *ldapi.RandomizationSettingsRep
	var res *http.Response
	var err error
	err = r.client.withConcurrency(r.client.ctx, func() error {
		settings, res, err = r.client.ld.ExperimentsApi.GetExperimentationSettings(r.client.ctx, projectKey).Execute()
		return err
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

	elems := make(map[string]attr.Value, len(settings.RandomizationUnits))
	for _, u := range settings.RandomizationUnits {
		unitKey := ""
		if u.RandomizationUnit != nil {
			unitKey = *u.RandomizationUnit
		}
		if unitKey == "" {
			continue
		}
		def := false
		if u.Default != nil {
			def = *u.Default
		}
		hidden := false
		if u.Hidden != nil {
			hidden = *u.Hidden
		}
		displayName := ""
		if u.DisplayName != nil {
			displayName = *u.DisplayName
		}
		obj, d := types.ObjectValue(experimentationSettingsUnitAttrTypes, map[string]attr.Value{
			DEFAULT:      types.BoolValue(def),
			DISPLAY_NAME: types.StringValue(displayName),
			HIDDEN:       types.BoolValue(hidden),
		})
		diags.Append(d...)
		elems[unitKey] = obj
	}

	unitMap, d := types.MapValue(types.ObjectType{AttrTypes: experimentationSettingsUnitAttrTypes}, elems)
	diags.Append(d...)
	data.RandomizationUnits = unitMap
}
