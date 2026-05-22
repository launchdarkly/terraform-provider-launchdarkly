package launchdarkly

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var (
	_ resource.Resource                = &EnvironmentResource{}
	_ resource.ResourceWithImportState = &EnvironmentResource{}
)

type EnvironmentResource struct {
	client *Client
}

type EnvironmentResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ProjectKey         types.String `tfsdk:"project_key"`
	Key                types.String `tfsdk:"key"`
	Name               types.String `tfsdk:"name"`
	Color              types.String `tfsdk:"color"`
	APIKey             types.String `tfsdk:"api_key"`
	MobileKey          types.String `tfsdk:"mobile_key"`
	ClientSideID       types.String `tfsdk:"client_side_id"`
	DefaultTTL         types.Int64  `tfsdk:"default_ttl"`
	SecureMode         types.Bool   `tfsdk:"secure_mode"`
	DefaultTrackEvents types.Bool   `tfsdk:"default_track_events"`
	RequireComments    types.Bool   `tfsdk:"require_comments"`
	ConfirmChanges     types.Bool   `tfsdk:"confirm_changes"`
	Critical           types.Bool   `tfsdk:"critical"`
	Tags               types.Set    `tfsdk:"tags"`
	ApprovalSettings   types.List   `tfsdk:"approval_settings"`
}

func NewEnvironmentResource() resource.Resource {
	return &EnvironmentResource{}
}

func (r *EnvironmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (r *EnvironmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly environment resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The LaunchDarkly project key.",
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The project-unique key for the environment.",
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			NAME: schema.StringAttribute{
				Required:    true,
				Description: "Human-readable name.",
			},
			COLOR: schema.StringAttribute{
				Required:    true,
				Description: "RGB hex color (no leading #).",
			},
			API_KEY:        schema.StringAttribute{Computed: true, Sensitive: true},
			MOBILE_KEY:     schema.StringAttribute{Computed: true, Sensitive: true},
			CLIENT_SIDE_ID: schema.StringAttribute{Computed: true, Sensitive: true},
			DEFAULT_TTL: schema.Int64Attribute{
				Optional: true, Computed: true,
				Default:     int64default.StaticInt64(0),
				Description: "TTL (0-60 minutes).",
			},
			SECURE_MODE: schema.BoolAttribute{
				Optional: true, Computed: true,
				Default: booldefault.StaticBool(false),
			},
			DEFAULT_TRACK_EVENTS: schema.BoolAttribute{
				Optional: true, Computed: true,
				Default: booldefault.StaticBool(false),
			},
			REQUIRE_COMMENTS: schema.BoolAttribute{
				Optional: true, Computed: true,
				Default: booldefault.StaticBool(false),
			},
			CONFIRM_CHANGES: schema.BoolAttribute{
				Optional: true, Computed: true,
				Default: booldefault.StaticBool(false),
			},
			CRITICAL: schema.BoolAttribute{
				Optional: true, Computed: true,
				Default: booldefault.StaticBool(false),
			},
			TAGS: schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			APPROVAL_SETTINGS: frameworkApprovalSettingsResourceAttribute(),
		},
	}
}

// environmentExists + environmentExistsInProject are shared helpers
// used by the project, segment, and feature_flag_environment resources.
func environmentExists(projectKey, envKey string, client *Client) (bool, error) {
	var res *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, res, err = client.ld.EnvironmentsApi.GetEnvironment(client.ctx, projectKey, envKey).Execute()
		return err
	})
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get environment %q in project %q: %s", envKey, projectKey, handleLdapiErr(err))
	}
	return true, nil
}

func environmentExistsInProject(project ldapi.Project, envKey string) bool {
	if project.Environments == nil {
		return false
	}
	for _, env := range project.Environments.Items {
		if env.Key == envKey {
			return true
		}
	}
	return false
}

func (r *EnvironmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *EnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EnvironmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	key := plan.Key.ValueString()
	defaultTTL := int32(plan.DefaultTTL.ValueInt64())
	secureMode := plan.SecureMode.ValueBool()
	defaultTrack := plan.DefaultTrackEvents.ValueBool()
	requireComments := plan.RequireComments.ValueBool()
	confirmChanges := plan.ConfirmChanges.ValueBool()
	critical := plan.Critical.ValueBool()
	tags, diags := stringSliceFromSet(ctx, plan.Tags)
	resp.Diagnostics.Append(diags...)

	envPost := ldapi.EnvironmentPost{
		Name:               plan.Name.ValueString(),
		Key:                key,
		Color:              plan.Color.ValueString(),
		DefaultTtl:         &defaultTTL,
		SecureMode:         &secureMode,
		DefaultTrackEvents: &defaultTrack,
		Tags:               tags,
		RequireComments:    &requireComments,
		ConfirmChanges:     &confirmChanges,
		Critical:           &critical,
	}

	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.EnvironmentsApi.PostEnvironment(r.client.ctx, projectKey).EnvironmentPost(envPost).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to create environment", err)
		return
	}
	plan.ID = types.StringValue(projectKey + "/" + key)

	// Approval settings, if any, applied via patch.
	if !plan.ApprovalSettings.IsNull() && !plan.ApprovalSettings.IsUnknown() && len(plan.ApprovalSettings.Elements()) > 0 {
		if d := r.applyApprovalPatch(ctx, projectKey, key, plan.ApprovalSettings, types.ListNull(plan.ApprovalSettings.ElementType(ctx))); d != nil {
			resp.Diagnostics.AddError("Failed to apply approval_settings", d.Error())
			return
		}
	}

	r.readIntoModel(ctx, projectKey, key, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data EnvironmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readIntoModel(ctx, data.ProjectKey.ValueString(), data.Key.ValueString(), &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state EnvironmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	envKey := plan.Key.ValueString()
	name := plan.Name.ValueString()
	color := plan.Color.ValueString()
	defaultTTL := int32(plan.DefaultTTL.ValueInt64())
	secureMode := plan.SecureMode.ValueBool()
	defaultTrack := plan.DefaultTrackEvents.ValueBool()
	requireComments := plan.RequireComments.ValueBool()
	confirmChanges := plan.ConfirmChanges.ValueBool()
	critical := plan.Critical.ValueBool()
	tags, diags := stringSliceFromSet(ctx, plan.Tags)
	resp.Diagnostics.Append(diags...)

	patch := []ldapi.PatchOperation{
		patchReplace("/name", name),
		patchReplace("/color", color),
		patchReplace("/defaultTtl", defaultTTL),
		patchReplace("/secureMode", secureMode),
		patchReplace("/defaultTrackEvents", defaultTrack),
		patchReplace("/requireComments", requireComments),
		patchReplace("/confirmChanges", confirmChanges),
		patchReplace("/critical", critical),
		patchReplace("/tags", &tags),
	}
	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.EnvironmentsApi.PatchEnvironment(r.client.ctx, projectKey, envKey).PatchOperation(patch).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to update environment", err)
		return
	}

	if !plan.ApprovalSettings.Equal(state.ApprovalSettings) {
		if d := r.applyApprovalPatch(ctx, projectKey, envKey, plan.ApprovalSettings, state.ApprovalSettings); d != nil {
			resp.Diagnostics.AddError("Failed to update approval_settings", d.Error())
		}
	}

	r.readIntoModel(ctx, projectKey, envKey, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data EnvironmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, e := r.client.ld.EnvironmentsApi.DeleteEnvironment(r.client.ctx, data.ProjectKey.ValueString(), data.Key.ValueString()).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to delete environment", err)
	}
}

func (r *EnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "expected project_key/env_key")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(KEY), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// applyApprovalPatch applies the diff between the planned and stored
// approval_settings as a JSON-patch against the environment. Returns
// nil on success.
func (r *EnvironmentResource) applyApprovalPatch(ctx context.Context, projectKey, envKey string, planList, stateList types.List) error {
	// Map plan -> ApprovalSettings via the deserialised model.
	planEmpty := planList.IsNull() || planList.IsUnknown() || len(planList.Elements()) == 0
	stateEmpty := stateList.IsNull() || stateList.IsUnknown() || len(stateList.Elements()) == 0
	if planEmpty && stateEmpty {
		return nil
	}
	if planEmpty {
		// Remove both possible nullable fields.
		patch := []ldapi.PatchOperation{
			patchRemove("/approvalSettings/required"),
			patchRemove("/approvalSettings/requiredApprovalTags"),
		}
		return r.client.withConcurrency(r.client.ctx, func() error {
			_, _, e := r.client.ld.EnvironmentsApi.PatchEnvironment(r.client.ctx, projectKey, envKey).PatchOperation(patch).Execute()
			return e
		})
	}

	// Use framework types in the intermediate model so Computed inner
	// attrs (which can be Unknown at plan time) don't trip the strict
	// `[]string` / `map[string]string` decoders in ElementsAs.
	type approvalModel struct {
		Required                 types.Bool   `tfsdk:"required"`
		CanReviewOwnRequest      types.Bool   `tfsdk:"can_review_own_request"`
		MinNumApprovals          types.Int64  `tfsdk:"min_num_approvals"`
		CanApplyDeclinedChanges  types.Bool   `tfsdk:"can_apply_declined_changes"`
		RequiredApprovalTags     types.List   `tfsdk:"required_approval_tags"`
		ServiceKind              types.String `tfsdk:"service_kind"`
		ServiceConfig            types.Map    `tfsdk:"service_config"`
		AutoApplyApprovedChanges types.Bool   `tfsdk:"auto_apply_approved_changes"`
	}
	var models []approvalModel
	if d := planList.ElementsAs(ctx, &models, false); d.HasError() {
		return fmt.Errorf("decode approval_settings: %v", d)
	}
	if len(models) == 0 {
		return nil
	}
	m := models[0]

	requiredApprovalTags, d := stringSliceFromList(ctx, m.RequiredApprovalTags)
	if d.HasError() {
		return fmt.Errorf("decode required_approval_tags: %v", d)
	}
	if m.Required.ValueBool() && len(requiredApprovalTags) > 0 {
		return fmt.Errorf("invalid approval_settings config: required and required_approval_tags cannot be set simultaneously")
	}
	serviceKind := m.ServiceKind.ValueString()
	autoApply := m.AutoApplyApprovedChanges.ValueBool()
	if serviceKind == "launchdarkly" && autoApply {
		return fmt.Errorf("invalid approval_settings config: auto_apply_approved_changes cannot be set to true for service_kind of launchdarkly")
	}

	serviceConfig := make(map[string]interface{})
	if !m.ServiceConfig.IsNull() && !m.ServiceConfig.IsUnknown() {
		raw, d := mapStringFromAttr(ctx, m.ServiceConfig)
		if d.HasError() {
			return fmt.Errorf("decode service_config: %v", d)
		}
		for k, v := range raw {
			serviceConfig[k] = v
		}
	}
	patch := []ldapi.PatchOperation{
		patchReplace("/approvalSettings/required", m.Required.ValueBool()),
		patchReplace("/approvalSettings/canReviewOwnRequest", m.CanReviewOwnRequest.ValueBool()),
		patchReplace("/approvalSettings/minNumApprovals", m.MinNumApprovals.ValueInt64()),
		patchReplace("/approvalSettings/canApplyDeclinedChanges", m.CanApplyDeclinedChanges.ValueBool()),
		patchReplace("/approvalSettings/requiredApprovalTags", requiredApprovalTags),
		patchReplace("/approvalSettings/serviceKind", serviceKind),
		patchReplace("/approvalSettings/serviceConfig", serviceConfig),
		patchReplace("/approvalSettings/autoApplyApprovedChanges", autoApply),
	}
	return r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.EnvironmentsApi.PatchEnvironment(r.client.ctx, projectKey, envKey).PatchOperation(patch).Execute()
		return e
	})
}

func (r *EnvironmentResource) readIntoModel(
	ctx context.Context,
	projectKey, envKey string,
	data *EnvironmentResourceModel,
	diags *diag.Diagnostics,
) {
	// Import context: ImportState sets only KEY/PROJECT_KEY/id. Other
	// fields are null/unknown until this Read populates them. On the
	var env *ldapi.Environment
	var res *http.Response
	var err error
	err = r.client.withConcurrency(r.client.ctx, func() error {
		env, res, err = r.client.ld.EnvironmentsApi.GetEnvironment(r.client.ctx, projectKey, envKey).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError("Failed to get environment", handleLdapiErr(err).Error())
		return
	}
	data.ID = types.StringValue(projectKey + "/" + envKey)
	data.ProjectKey = types.StringValue(projectKey)
	data.Key = types.StringValue(env.Key)
	data.Name = types.StringValue(env.Name)
	data.Color = types.StringValue(env.Color)
	data.APIKey = types.StringValue(env.ApiKey)
	data.MobileKey = types.StringValue(env.MobileKey)
	data.ClientSideID = types.StringValue(env.Id)
	data.DefaultTTL = types.Int64Value(int64(env.DefaultTtl))
	data.SecureMode = types.BoolValue(env.SecureMode)
	data.DefaultTrackEvents = types.BoolValue(env.DefaultTrackEvents)
	data.RequireComments = types.BoolValue(env.RequireComments)
	data.ConfirmChanges = types.BoolValue(env.ConfirmChanges)
	data.Critical = types.BoolValue(env.Critical)

	tagsSet, d := setFromStringSliceOrNull(ctx, env.Tags)
	diags.Append(d...)
	data.Tags = tagsSet

	approvals, d := frameworkApprovalSettingsValue(ctx, env.ApprovalSettings, data.ApprovalSettings)
	diags.Append(d...)
	data.ApprovalSettings = approvals
}
