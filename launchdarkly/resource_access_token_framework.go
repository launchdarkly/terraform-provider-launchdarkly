package launchdarkly

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v23"
)

var (
	_ resource.Resource                     = &AccessTokenResource{}
	_ resource.ResourceWithImportState      = &AccessTokenResource{}
	_ resource.ResourceWithConfigValidators = &AccessTokenResource{}
	_ resource.ResourceWithModifyPlan       = &AccessTokenResource{}
	_ resource.ResourceWithUpgradeState     = &AccessTokenResource{}
)

type AccessTokenResource struct {
	client *Client
}

type AccessTokenResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Role              types.String `tfsdk:"role"`
	CustomRoles       types.Set    `tfsdk:"custom_roles"`
	InlineRoles       types.List   `tfsdk:"inline_roles"`
	ServiceToken      types.Bool   `tfsdk:"service_token"`
	DefaultAPIVersion types.Int64  `tfsdk:"default_api_version"`
	Token             types.String `tfsdk:"token"`
}

func NewAccessTokenResource() resource.Resource {
	return &AccessTokenResource{}
}

func (r *AccessTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_token"
}

func (r *AccessTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 1,
		Description: `Provides a LaunchDarkly access token resource.

This resource allows you to create and manage access tokens within your LaunchDarkly organization.

-> **Note:** This resource stores the full plaintext secret for your access token in Terraform state. Be sure your state is configured securely before using this resource. To learn more, read [Sensitive data in state](https://www.terraform.io/docs/state/sensitive-data.html).

The resource must contain either a "role", "custom_role" or an "inline_roles" block.`,
		Attributes: accessTokenSchemaAttributes(),
	}
}

func accessTokenSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		NAME: schema.StringAttribute{
			Optional:    true,
			Description: "A human-friendly name for the access token.",
		},
		ROLE: schema.StringAttribute{
			Optional:    true,
			Description: "A built-in LaunchDarkly role. Can be `reader`, `writer`, or `admin`",
			Validators: []validator.String{
				oneOfValidator{allowed: []string{"reader", "writer", "admin"}},
			},
		},
		CUSTOM_ROLES: schema.SetAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: "A list of custom role IDs to use as access limits for the access token.",
		},
		SERVICE_TOKEN: schema.BoolAttribute{
			Optional: true,
			// framework requires Computed: true alongside Default.
			Computed:    true,
			Default:     booldefault.StaticBool(false),
			Description: addForceNewDescription("Whether the token is a [service token](https://launchdarkly.com/docs/home/account/api#service-tokens).", true),
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.RequiresReplace(),
			},
		},
		DEFAULT_API_VERSION: schema.Int64Attribute{
			Optional:    true,
			Computed:    true,
			Description: addForceNewDescription("The default API version for this token. Defaults to the latest API version.", true),
			PlanModifiers: []planmodifier.Int64{
				// Per-attribute UseStateForUnknown here trips the
				// .token inconsistent-sensitive-attr check on
				// expire-triggered resets (TestAccAccessToken_Reset).
				// Use resource-level ModifyPlan instead — it runs
				// after per-attribute modifiers and doesn't disturb
				// the framework's Computed-coupling that the token
				// reset path relies on.
				int64planmodifier.RequiresReplace(),
			},
			Validators: []validator.Int64{
				apiVersionValidator{},
			},
		},
		TOKEN: schema.StringAttribute{
			Computed:      true,
			Sensitive:     true,
			Description:   "The access token used to authorize usage of the LaunchDarkly API.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		INLINE_ROLES: frameworkPolicyStatementsResourceAttribute(
			false,
			"Define inline custom roles. An array of statements with three attributes: effect, resources, actions. May be used in place of a built-in or custom role. [Using polices](https://launchdarkly.com/docs/home/account/roles/role-policies).",
			"",
		),
	}
}

func (r *AccessTokenResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	priorSchema := schema.Schema{Attributes: accessTokenSchemaAttributesV0()}
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &priorSchema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior AccessTokenResourceModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}
				priorPS := nullIfEmptyList(ctx, prior.PolicyStatements)
				priorIR := nullIfEmptyList(ctx, prior.InlineRoles)
				psSet := !priorPS.IsNull() && !priorPS.IsUnknown() && len(priorPS.Elements()) > 0
				irSet := !priorIR.IsNull() && !priorIR.IsUnknown() && len(priorIR.Elements()) > 0
				if psSet && irSet {
					resp.Diagnostics.AddError(
						"Cannot upgrade access_token state: both policy_statements and inline_roles set",
						"v2 ConfigValidator should have prevented this. Resolve by manually editing state to drop one of the two attributes, then re-apply.",
					)
					return
				}
				data := AccessTokenResourceModel{
					ID:                prior.ID,
					Name:              prior.Name,
					Role:              prior.Role,
					CustomRoles:       prior.CustomRoles,
					InlineRoles:       priorIR,
					ServiceToken:      prior.ServiceToken,
					DefaultAPIVersion: prior.DefaultAPIVersion,
					Token:             prior.Token,
				}
				// policy_statements -> inline_roles: identical shape (both
				// built from frameworkPolicyStatementsResourceAttribute), so
				// just move the list onto inline_roles when only PS was set.
				if psSet {
					data.InlineRoles = priorPS
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			},
		},
	}
}

func (r *AccessTokenResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.Conflicting(
			path.MatchRoot(ROLE),
			path.MatchRoot(CUSTOM_ROLES),
			path.MatchRoot(INLINE_ROLES),
		),
	}
}

func (r *AccessTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

// ModifyPlan preserves default_api_version across upgrades from v2.x
// state where the attribute was implicit. Per-attribute
// UseStateForUnknown on default_api_version was historically avoided
// to keep the framework's Computed-coupling intact for the
// expire-driven reset path; expire has been removed in v3 but the
// state-preservation logic is still useful when upgrading from
// states where default_api_version was unset.
func (r *AccessTokenResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || req.State.Raw.IsNull() {
		return
	}
	var config, state, plan AccessTokenResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !config.DefaultAPIVersion.IsNull() {
		return
	}
	if state.DefaultAPIVersion.IsNull() || state.DefaultAPIVersion.IsUnknown() {
		return
	}
	if !plan.DefaultAPIVersion.IsUnknown() {
		return
	}
	plan.DefaultAPIVersion = state.DefaultAPIVersion
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r *AccessTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AccessTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()
	serviceToken := plan.ServiceToken.ValueBool()
	body := ldapi.AccessTokenPost{
		Name:         ldapi.PtrString(name),
		ServiceToken: ldapi.PtrBool(serviceToken),
	}
	if !plan.DefaultAPIVersion.IsNull() && !plan.DefaultAPIVersion.IsUnknown() && plan.DefaultAPIVersion.ValueInt64() != 0 {
		v := int32(plan.DefaultAPIVersion.ValueInt64())
		body.DefaultApiVersion = &v
	}

	// Precedence: inline_roles > custom_roles > role.
	inline, diags := frameworkPolicyStatementsFromList(ctx, plan.InlineRoles)
	resp.Diagnostics.Append(diags...)
	customRoles, diags := stringSliceFromSet(ctx, plan.CustomRoles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	switch {
	case len(inline) > 0:
		body.InlineRole = inline
	case len(customRoles) > 0:
		body.CustomRoleIds = customRoles
	case plan.Role.ValueString() != "":
		v := plan.Role.ValueString()
		body.Role = &v
	default:
		resp.Diagnostics.AddError(
			"Missing role configuration",
			"access_token must contain either 'role', 'custom_roles', or 'inline_roles'.",
		)
		return
	}

	var token *ldapi.Token
	err := r.client.withConcurrency(r.client.ctx, func() error {
		var e error
		token, _, e = r.client.ld.AccessTokensApi.PostToken(r.client.ctx).AccessTokenPost(body).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to create access token", err)
		return
	}
	plan.ID = types.StringValue(token.Id)
	plan.Token = stringValueFromPointer(token.Token)

	r.readIntoModel(ctx, token.Id, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AccessTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AccessTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readIntoModel(ctx, data.ID.ValueString(), &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccessTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state AccessTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := plan.ID.ValueString()
	name := plan.Name.ValueString()
	role := plan.Role.ValueString()
	customRoleKeys, diags := stringSliceFromSet(ctx, plan.CustomRoles)
	resp.Diagnostics.Append(diags...)
	customRoleIds, err := customRoleKeysToIDs(r.client, customRoleKeys)
	if err != nil {
		resp.Diagnostics.AddError("Failed to look up custom roles", err.Error())
		return
	}

	inline, diags := frameworkPolicyStatementsFromList(ctx, plan.InlineRoles)
	resp.Diagnostics.Append(diags...)

	patch := []ldapi.PatchOperation{patchReplace("/name", &name)}

	if !plan.Role.Equal(state.Role) {
		if role == "" {
			patch = append(patch, patchRemove("/role"))
		} else {
			patch = append(patch, patchReplace("/role", &role))
		}
	}
	if !plan.CustomRoles.Equal(state.CustomRoles) {
		if len(customRoleIds) == 0 {
			patch = append(patch, patchRemove("/customRoleIds"))
		} else {
			patch = append(patch, patchReplace("/customRoleIds", &customRoleIds))
		}
	}
	// Patch inlineRole only when the semantic value actually changes.
	// state == [] and plan == null both mean "no inline roles"; treating
	// them as different (framework Equal does) and issuing patchRemove
	// for a token that never had an inline role returns 422 from LD.
	emptyList := func(l types.List) bool {
		return l.IsNull() || l.IsUnknown() || len(l.Elements()) == 0
	}
	hadInline := !emptyList(state.InlineRoles)
	wantInline := !emptyList(plan.InlineRoles)
	if hadInline != wantInline || (wantInline && !plan.InlineRoles.Equal(state.InlineRoles)) {
		if len(inline) == 0 {
			if hadInline {
				patch = append(patch, patchRemove("/inlineRole"))
			}
		} else {
			patch = append(patch, patchReplace("/inlineRole", &inline))
		}
	}

	err = r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.AccessTokensApi.PatchToken(r.client.ctx, id).PatchOperation(patch).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to update access token", err)
		return
	}

	r.readIntoModel(ctx, id, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AccessTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AccessTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, e := r.client.ld.AccessTokensApi.DeleteToken(r.client.ctx, data.ID.ValueString()).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to delete access token", err)
	}
}

func (r *AccessTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *AccessTokenResource) readIntoModel(
	ctx context.Context,
	id string,
	data *AccessTokenResourceModel,
	diags *diag.Diagnostics,
) {
	var accessToken *ldapi.Token
	var res *http.Response
	var err error
	err = r.client.withConcurrency(r.client.ctx, func() error {
		accessToken, res, err = r.client.ld.AccessTokensApi.GetToken(r.client.ctx, id).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError("Failed to get access token", handleLdapiErr(err).Error())
		return
	}
	data.ID = types.StringValue(accessToken.Id)
	data.Name = stringValueOrNullFromPointer(accessToken.Name)
	if accessToken.Role != nil {
		data.Role = types.StringValue(*accessToken.Role)
	}
	if accessToken.ServiceToken != nil {
		data.ServiceToken = types.BoolValue(*accessToken.ServiceToken)
	}
	if accessToken.DefaultApiVersion != nil {
		data.DefaultAPIVersion = types.Int64Value(int64(*accessToken.DefaultApiVersion))
	}

	if len(accessToken.CustomRoleIds) > 0 {
		customRoleKeys, err := customRoleIDsToKeys(r.client, accessToken.CustomRoleIds)
		if err != nil {
			diags.AddError("Failed to resolve custom role keys", err.Error())
			return
		}
		s, _ := setFromStringSlice(ctx, customRoleKeys)
		data.CustomRoles = s
	}

	if len(accessToken.InlineRole) > 0 {
		stmts, _ := frameworkPolicyStatementsValue(ctx, accessToken.InlineRole)
		data.InlineRoles = stmts
	}
}

// apiVersionValidator accepts 0 (unset), 20240415, 20191212, 20160426.
type apiVersionValidator struct{}

func (apiVersionValidator) Description(_ context.Context) string {
	return "value must be one of `20240415`, `20191212`, or `20160426`"
}
func (apiVersionValidator) MarkdownDescription(ctx context.Context) string {
	return apiVersionValidator{}.Description(ctx)
}
func (apiVersionValidator) ValidateInt64(_ context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	v := req.ConfigValue.ValueInt64()
	switch v {
	case 0, 20240415, 20191212, 20160426:
		// valid
	default:
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid API version",
			fmt.Errorf("%q must be one of `20240415`, `20191212`, or `20160426`. Got: %v", DEFAULT_API_VERSION, v).Error(),
		)
	}
}

// noZeroValuesInt64Validator rejects int64 values equal to zero.
type noZeroValuesInt64Validator struct{}

func (noZeroValuesInt64Validator) Description(_ context.Context) string {
	return "value must not be zero"
}
func (noZeroValuesInt64Validator) MarkdownDescription(ctx context.Context) string {
	return noZeroValuesInt64Validator{}.Description(ctx)
}
func (noZeroValuesInt64Validator) ValidateInt64(_ context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	v := req.ConfigValue.ValueInt64()
	if v == 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid value",
			fmt.Errorf("expected %q to not be an empty value, got %v", EXPIRE, v).Error(),
		)
	}
}
