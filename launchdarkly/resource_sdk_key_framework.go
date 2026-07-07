package launchdarkly

import (
	"context"
	"fmt"

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
	_ resource.Resource                = &SdkKeyResource{}
	_ resource.ResourceWithImportState = &SdkKeyResource{}
	_ resource.ResourceWithModifyPlan  = &SdkKeyResource{}
)

type SdkKeyResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectKey     types.String `tfsdk:"project_key"`
	EnvironmentKey types.String `tfsdk:"environment_key"`
	Key            types.String `tfsdk:"key"`
	Kind           types.String `tfsdk:"kind"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Expiry         types.Int64  `tfsdk:"expiry"`
	Value          types.String `tfsdk:"value"`
	IsDefault      types.Bool   `tfsdk:"is_default"`
	Version        types.Int64  `tfsdk:"version"`
}

type SdkKeyResource struct {
	client *Client
	beta   *Client
}

func NewSdkKeyResource() resource.Resource {
	return &SdkKeyResource{}
}

func (r *SdkKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sdk_key"
}

func (r *SdkKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provides a LaunchDarkly SDK key resource.

~> **Beta:** This resource uses a beta API. Beta resources may change or be removed in future versions.

This resource allows you to create and manage a server-side or mobile SDK key for a specific project environment. The generated key value is available in the ` + "`value`" + ` attribute.`,
		Attributes: sdkKeySchemaAttributes(),
	}
}

func sdkKeySchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:      true,
			Description:   "The unique resource ID in the format `project_key/environment_key/key`.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		PROJECT_KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The project key.", true),
			Validators:    []validator.String{keyValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		ENVIRONMENT_KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The environment key.", true),
			Validators:    []validator.String{keyValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The user-defined identifying key of the SDK key. This is distinct from the `value` attribute, which is the actual SDK key value used by your SDK.", true),
			Validators:    []validator.String{keyValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		KIND: schema.StringAttribute{
			Optional: true,
			Computed: true,
			// No schema Default: a static default would override the state
			// value whenever the configuration omits kind, planning a replace
			// of an imported `mobile` key to `sdk` — destructive, and doomed
			// because deleted SDK key identifiers are tombstoned (POST 409).
			// UseStateForUnknown keeps the stored kind when the configuration
			// omits it; the API defaults new keys to `sdk` server-side.
			Description: addForceNewDescription("The kind of SDK key. Must be either `sdk` (server-side) or `mobile`. New keys default to `sdk`.", true),
			Validators:  []validator.String{oneOfValidator{allowed: []string{"sdk", "mobile"}}},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
				stringplanmodifier.RequiresReplace(),
			},
		},
		NAME: schema.StringAttribute{
			Required:    true,
			Description: "The human-readable name of the SDK key.",
		},
		DESCRIPTION: schema.StringAttribute{
			Optional:    true,
			Description: "The description of the SDK key.",
		},
		EXPIRY: schema.Int64Attribute{
			Optional:    true,
			Description: "An expiration date for the SDK key, expressed as a Unix epoch time in milliseconds. When set, the key becomes invalid after this time. Once set, an expiry cannot be removed: the beta API cannot clear a scheduled expiry in place, and a deleted SDK key identifier cannot be recreated in the same environment.",
		},
		VALUE: schema.StringAttribute{
			Computed:      true,
			Sensitive:     true,
			Description:   "The actual SDK key value. Use this when configuring your SDK. This value is generated by LaunchDarkly and cannot be set.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		IS_DEFAULT: schema.BoolAttribute{
			Computed:    true,
			Description: "Whether this SDK key is the system-defined default for the environment.",
		},
		VERSION: schema.Int64Attribute{
			Computed:    true,
			Description: "The auto-incremented version number of the SDK key.",
		},
	}
}

// ModifyPlan rejects, at plan time, any attempt to remove a previously-set
// expiry in place. Removal is unsupported end to end: the beta patch model
// marshals expiry with `omitempty` and cannot emit a null to clear it (a
// PATCH with expiry=0 is rejected 400), and replacing the resource at the
// same key is not viable either because a deleted SDK key identifier is
// tombstoned by the backend (POST returns 409 "SDK key already exists").
// Surfacing a plan error is therefore safer than RequiresReplace, which
// would destroy the key and then fail to recreate it.
//
// When a RequiresReplace attribute changes in the same plan, the resource is
// being replaced under a fresh identifier, where omitting expiry is valid —
// that transition is allowed. This check lives in resource-level ModifyPlan
// rather than an attribute plan modifier because it must observe the other
// attributes' final planned values: attribute plan modifiers run in
// unspecified order, so an expiry-attribute guard could read `kind` before
// UseStateForUnknown resolves it and misread the plan as a replacement.
func (r *SdkKeyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		// Destroy or create: no in-place expiry transition to guard.
		return
	}
	var plan, state SdkKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.Expiry.IsNull() || !plan.Expiry.IsNull() {
		return
	}
	for _, pair := range [][2]types.String{
		{plan.ProjectKey, state.ProjectKey},
		{plan.EnvironmentKey, state.EnvironmentKey},
		{plan.Key, state.Key},
		{plan.Kind, state.Kind},
	} {
		if !pair[0].Equal(pair[1]) {
			return
		}
	}
	resp.Diagnostics.AddAttributeError(
		path.Root(EXPIRY),
		"Cannot remove expiry from an SDK key",
		"The SDK key beta API cannot clear a scheduled expiry once set, and a deleted SDK key identifier cannot be recreated in the same environment. To manage an SDK key without an expiry, create a new resource with a different key.",
	)
}

func (r *SdkKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
	if r.client == nil {
		return
	}
	beta, err := newBetaClient(r.client.apiKey, r.client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build LaunchDarkly beta client", err.Error())
		return
	}
	r.beta = beta
}

func (r *SdkKeyResource) betaClient() (*Client, error) {
	if r.beta != nil {
		return r.beta, nil
	}
	return newBetaClient(r.client.apiKey, r.client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
}

func (r *SdkKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SdkKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	beta, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	environmentKey := plan.EnvironmentKey.ValueString()
	sdkKeyKey := plan.Key.ValueString()

	post := ldapi.NewSdkKeyPost(sdkKeyKey, plan.Name.ValueString())
	if !plan.Kind.IsNull() && !plan.Kind.IsUnknown() {
		post.SetKind(plan.Kind.ValueString())
	}
	if !plan.Description.IsNull() && plan.Description.ValueString() != "" {
		post.SetDescription(plan.Description.ValueString())
	}
	if !plan.Expiry.IsNull() {
		post.SetExpiry(plan.Expiry.ValueInt64())
	}

	if _, err := createSdkKey(beta, projectKey, environmentKey, *post); err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Failed to create SDK key %q in environment %q", sdkKeyKey, environmentKey), err)
		return
	}

	plan.ID = types.StringValue(sdkKeyID(projectKey, environmentKey, sdkKeyKey))
	r.readIntoModel(ctx, projectKey, environmentKey, sdkKeyKey, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SdkKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SdkKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readIntoModel(ctx, data.ProjectKey.ValueString(), data.EnvironmentKey.ValueString(), data.Key.ValueString(), &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SdkKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state SdkKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	beta, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	environmentKey := plan.EnvironmentKey.ValueString()
	sdkKeyKey := plan.Key.ValueString()

	// The SDK key PATCH endpoint accepts only name, description, and expiry.
	// project_key, environment_key, key, and kind are RequiresReplace, so they
	// never reach Update.
	patch := ldapi.NewSdkKeyPatch()
	changed := false
	if !plan.Name.Equal(state.Name) {
		patch.SetName(plan.Name.ValueString())
		changed = true
	}
	if !plan.Description.Equal(state.Description) {
		patch.SetDescription(plan.Description.ValueString())
		changed = true
	}
	// Setting or moving expiry to a new value is an in-place patch. Removing it
	// (plan null, state set) never reaches Update: expiryRemovalGuard rejects
	// that transition at plan time, because the beta patch model cannot emit a
	// null expiry to clear a scheduled expiration and the key cannot be
	// recreated at the same identifier.
	if !plan.Expiry.Equal(state.Expiry) && !plan.Expiry.IsNull() {
		patch.SetExpiry(plan.Expiry.ValueInt64())
		changed = true
	}

	if changed {
		if _, err := patchSdkKey(beta, projectKey, environmentKey, sdkKeyKey, *patch); err != nil {
			addLdapiError(&resp.Diagnostics, fmt.Sprintf("Failed to update SDK key %q in environment %q", sdkKeyKey, environmentKey), err)
			return
		}
	}

	r.readIntoModel(ctx, projectKey, environmentKey, sdkKeyKey, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SdkKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SdkKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	beta, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}
	res, err := deleteSdkKey(beta, data.ProjectKey.ValueString(), data.EnvironmentKey.ValueString(), data.Key.ValueString())
	if err != nil {
		if isStatusNotFound(res) {
			return
		}
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Failed to delete SDK key %q", data.Key.ValueString()), err)
	}
}

func (r *SdkKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectKey, environmentKey, sdkKeyKey, err := sdkKeyIDToKeys(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), projectKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(ENVIRONMENT_KEY), environmentKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(KEY), sdkKeyKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *SdkKeyResource) readIntoModel(
	ctx context.Context,
	projectKey, environmentKey, sdkKeyKey string,
	data *SdkKeyResourceModel,
	diags *diag.Diagnostics,
) {
	beta, err := r.betaClient()
	if err != nil {
		diags.AddError("Failed to build beta client", err.Error())
		return
	}
	sdkKey, res, err := getSdkKey(beta, projectKey, environmentKey, sdkKeyKey)
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError(fmt.Sprintf("Failed to get SDK key %q", sdkKeyKey), handleLdapiErr(err).Error())
		return
	}

	data.ID = types.StringValue(sdkKeyID(projectKey, environmentKey, sdkKeyKey))
	data.ProjectKey = types.StringValue(projectKey)
	data.EnvironmentKey = types.StringValue(environmentKey)
	data.Key = types.StringValue(sdkKey.Key)
	data.Kind = types.StringValue(string(sdkKey.Kind))
	data.Name = types.StringValue(sdkKey.Name)
	// Optional-only attr: null-when-empty for plan-apply consistency.
	data.Description = stringValueOrNullFromPointer(sdkKey.Description)
	if sdkKey.Expiry != nil {
		data.Expiry = types.Int64Value(*sdkKey.Expiry)
	} else {
		data.Expiry = types.Int64Null()
	}
	data.Value = types.StringValue(sdkKey.Value)
	data.IsDefault = types.BoolValue(sdkKey.IsDefault)
	data.Version = types.Int64Value(int64(sdkKey.Version))
}
