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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	ldapi "github.com/launchdarkly/api-client-go/v23"
)

var (
	_ resource.Resource                 = &FlagTemplatesResource{}
	_ resource.ResourceWithImportState  = &FlagTemplatesResource{}
	_ resource.ResourceWithUpgradeState = &FlagTemplatesResource{}
)

type FlagTemplatesResource struct {
	client *Client
}

type FlagTemplatesResourceModel struct {
	ID              types.String `tfsdk:"id"`
	ProjectKey      types.String `tfsdk:"project_key"`
	Tags            types.Set    `tfsdk:"tags"`
	Temporary       types.Bool   `tfsdk:"temporary"`
	BooleanDefaults types.Object `tfsdk:"boolean_defaults"`
}

// FlagTemplatesResourceModelV0 is the pre-object state shape:
// boolean_defaults was a single-element list (v2.x SDKv2 MaxItems:1
// block and 3.0.0-beta nested attribute).
type FlagTemplatesResourceModelV0 struct {
	ID              types.String `tfsdk:"id"`
	ProjectKey      types.String `tfsdk:"project_key"`
	Tags            types.Set    `tfsdk:"tags"`
	Temporary       types.Bool   `tfsdk:"temporary"`
	BooleanDefaults types.List   `tfsdk:"boolean_defaults"`
}

func NewFlagTemplatesResource() resource.Resource {
	return &FlagTemplatesResource{}
}

func (r *FlagTemplatesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flag_templates"
}

func (r *FlagTemplatesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the Custom flag-template settings for a LaunchDarkly project.",
		Version:     1,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The project key.",
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			TAGS: schema.SetAttribute{
				Optional: true, Computed: true,
				ElementType: types.StringType,
			},
			TEMPORARY: schema.BoolAttribute{
				Optional: true, Computed: true,
				Default: booldefault.StaticBool(false),
			},
			BOOLEAN_DEFAULTS: schema.SingleNestedAttribute{
				Required:    true,
				Description: "Default boolean variation settings.",
				Attributes: map[string]schema.Attribute{
					TRUE_DISPLAY_NAME:  schema.StringAttribute{Required: true},
					FALSE_DISPLAY_NAME: schema.StringAttribute{Required: true},
					TRUE_DESCRIPTION:   schema.StringAttribute{Required: true},
					FALSE_DESCRIPTION:  schema.StringAttribute{Required: true},
					ON_VARIATION:       schema.Int64Attribute{Required: true},
					OFF_VARIATION:      schema.Int64Attribute{Required: true},
				},
			},
		},
	}
}

// UpgradeState projects the v0 (pre-object) single-element
// boolean_defaults list into the object shape. Version 0 covers both
// genuine v2.x SDKv2 state and 3.0.0-beta state.
func (r *FlagTemplatesResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	priorSchema := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":        schema.StringAttribute{Computed: true},
			PROJECT_KEY: schema.StringAttribute{Required: true},
			TAGS: schema.SetAttribute{
				Optional: true, Computed: true, ElementType: types.StringType,
			},
			TEMPORARY: schema.BoolAttribute{Optional: true, Computed: true},
			BOOLEAN_DEFAULTS: schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						TRUE_DISPLAY_NAME:  schema.StringAttribute{Required: true},
						FALSE_DISPLAY_NAME: schema.StringAttribute{Required: true},
						TRUE_DESCRIPTION:   schema.StringAttribute{Required: true},
						FALSE_DESCRIPTION:  schema.StringAttribute{Required: true},
						ON_VARIATION:       schema.Int64Attribute{Required: true},
						OFF_VARIATION:      schema.Int64Attribute{Required: true},
					},
				},
			},
		},
	}
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &priorSchema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior FlagTemplatesResourceModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}
				bdObj, d := flagTemplatesBooleanDefaultsObjectFromV0List(ctx, prior.BooleanDefaults)
				resp.Diagnostics.Append(d...)
				if resp.Diagnostics.HasError() {
					return
				}
				data := FlagTemplatesResourceModel{
					ID:              prior.ID,
					ProjectKey:      prior.ProjectKey,
					Tags:            prior.Tags,
					Temporary:       prior.Temporary,
					BooleanDefaults: bdObj,
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			},
		},
	}
}

// flagTemplatesBooleanDefaultsObjectFromV0List projects a v0
// single-element boolean_defaults list into the object shape. Returns a
// null object for null/empty input.
func flagTemplatesBooleanDefaultsObjectFromV0List(ctx context.Context, l types.List) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	if l.IsNull() || l.IsUnknown() || len(l.Elements()) == 0 {
		return types.ObjectNull(flagTemplatesBooleanDefaultsAttrTypes), diags
	}
	var models []flagTemplatesBooleanDefaultsModel
	diags.Append(l.ElementsAs(ctx, &models, false)...)
	if diags.HasError() || len(models) == 0 {
		return types.ObjectNull(flagTemplatesBooleanDefaultsAttrTypes), diags
	}
	m := models[0]
	obj, d := types.ObjectValue(flagTemplatesBooleanDefaultsAttrTypes, map[string]attr.Value{
		TRUE_DISPLAY_NAME:  types.StringValue(m.TrueDisplayName),
		FALSE_DISPLAY_NAME: types.StringValue(m.FalseDisplayName),
		TRUE_DESCRIPTION:   types.StringValue(m.TrueDescription),
		FALSE_DESCRIPTION:  types.StringValue(m.FalseDescription),
		ON_VARIATION:       types.Int64Value(m.OnVariation),
		OFF_VARIATION:      types.Int64Value(m.OffVariation),
	})
	diags.Append(d...)
	return obj, diags
}

func (r *FlagTemplatesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

type flagTemplatesBooleanDefaultsModel struct {
	TrueDisplayName  string `tfsdk:"true_display_name"`
	FalseDisplayName string `tfsdk:"false_display_name"`
	TrueDescription  string `tfsdk:"true_description"`
	FalseDescription string `tfsdk:"false_description"`
	OnVariation      int64  `tfsdk:"on_variation"`
	OffVariation     int64  `tfsdk:"off_variation"`
}

func (r *FlagTemplatesResource) upsert(ctx context.Context, plan *FlagTemplatesResourceModel, diags *diag.Diagnostics) error {
	projectKey := plan.ProjectKey.ValueString()

	csa, err := getCurrentCSA(r.client, projectKey)
	if err != nil {
		return fmt.Errorf("failed to read CSA: %s", handleLdapiErr(err))
	}

	tags, _ := stringSliceFromSet(ctx, plan.Tags)

	if plan.BooleanDefaults.IsNull() || plan.BooleanDefaults.IsUnknown() {
		return fmt.Errorf("boolean_defaults must be set")
	}
	var bd flagTemplatesBooleanDefaultsModel
	if d := plan.BooleanDefaults.As(ctx, &bd, basetypes.ObjectAsOptions{}); d.HasError() {
		return fmt.Errorf("decode boolean_defaults: %v", d)
	}
	payload := *ldapi.NewUpsertFlagDefaultsPayload(
		tags,
		plan.Temporary.ValueBool(),
		*ldapi.NewBooleanFlagDefaults(
			bd.TrueDisplayName,
			bd.FalseDisplayName,
			bd.TrueDescription,
			bd.FalseDescription,
			int32(bd.OnVariation),
			int32(bd.OffVariation),
		),
		*csa,
	)
	return r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.ProjectsApi.PutFlagDefaultsByProject(r.client.ctx, projectKey).UpsertFlagDefaultsPayload(payload).Execute()
		return e
	})
}

func (r *FlagTemplatesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FlagTemplatesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.upsert(ctx, &plan, &resp.Diagnostics); err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to create flag templates", err)
		return
	}
	plan.ID = types.StringValue(plan.ProjectKey.ValueString())
	r.readIntoModel(ctx, plan.ProjectKey.ValueString(), &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FlagTemplatesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FlagTemplatesResourceModel
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

func (r *FlagTemplatesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FlagTemplatesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.upsert(ctx, &plan, &resp.Diagnostics); err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to update flag templates", err)
		return
	}
	r.readIntoModel(ctx, plan.ProjectKey.ValueString(), &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FlagTemplatesResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Flag templates always exist for a project; destroying just
	// removes the entry from state.
}

func (r *FlagTemplatesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *FlagTemplatesResource) readIntoModel(
	ctx context.Context,
	projectKey string,
	data *FlagTemplatesResourceModel,
	diags *diag.Diagnostics,
) {
	var flagDefaults *ldapi.FlagDefaultsRep
	var res *http.Response
	var err error
	err = r.client.withConcurrency(r.client.ctx, func() error {
		flagDefaults, res, err = r.client.ld.ProjectsApi.GetFlagDefaultsByProject(r.client.ctx, projectKey).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError("Failed to read flag templates", handleLdapiErr(err).Error())
		return
	}
	data.ID = types.StringValue(projectKey)
	data.ProjectKey = types.StringValue(projectKey)

	tags := flagDefaults.Tags
	if tags == nil {
		tags = []string{}
	}
	tagsSet, _ := setFromStringSlice(ctx, tags)
	data.Tags = tagsSet

	if flagDefaults.Temporary != nil {
		data.Temporary = types.BoolValue(*flagDefaults.Temporary)
	} else {
		data.Temporary = types.BoolValue(false)
	}

	if flagDefaults.BooleanDefaults != nil {
		bd := flagDefaults.BooleanDefaults
		obj, _ := types.ObjectValue(flagTemplatesBooleanDefaultsAttrTypes, map[string]attr.Value{
			TRUE_DISPLAY_NAME:  types.StringValue(bd.GetTrueDisplayName()),
			FALSE_DISPLAY_NAME: types.StringValue(bd.GetFalseDisplayName()),
			TRUE_DESCRIPTION:   types.StringValue(bd.GetTrueDescription()),
			FALSE_DESCRIPTION:  types.StringValue(bd.GetFalseDescription()),
			ON_VARIATION:       types.Int64Value(int64(bd.GetOnVariation())),
			OFF_VARIATION:      types.Int64Value(int64(bd.GetOffVariation())),
		})
		data.BooleanDefaults = obj
	} else {
		data.BooleanDefaults = types.ObjectNull(flagTemplatesBooleanDefaultsAttrTypes)
	}
}
