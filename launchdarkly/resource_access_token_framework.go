package launchdarkly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

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
	ldapi "github.com/launchdarkly/api-client-go/v22"
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
	PolicyStatements  types.List   `tfsdk:"policy_statements"`
	InlineRoles       types.List   `tfsdk:"inline_roles"`
	ServiceToken      types.Bool   `tfsdk:"service_token"`
	DefaultAPIVersion types.Int64  `tfsdk:"default_api_version"`
	Token             types.String `tfsdk:"token"`
	Expire            types.Int64  `tfsdk:"expire"`
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

The resource must contain either a "role", "custom_role" or an "inline_roles" (previously "policy_statements") block. As of v1.7.0, "policy_statements" has been deprecated in favor of "inline_roles".`,
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
			Description: addForceNewDescription("Whether the token will be a [service token](https://docs.launchdarkly.com/home/account-security/api-access-tokens#service-tokens).", true),
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
		EXPIRE: schema.Int64Attribute{
			Optional:           true,
			Description:        "An expiration time for the current token secret, expressed as a Unix epoch time. Replace the computed token secret with a new value. The expired secret will no longer be able to authorize usage of the LaunchDarkly API. This field argument is **deprecated**. Please update your config to remove `expire` to maintain compatibility with future versions",
			DeprecationMessage: "'expire' is deprecated and will be removed in the next major release of the LaunchDarkly provider",
			Validators: []validator.Int64{
				noZeroValuesInt64Validator{},
			},
		},
		POLICY_STATEMENTS: frameworkPolicyStatementsResourceAttribute(
			false,
			"Define inline custom roles. An array of statements with three attributes: effect, resources, actions. May be used in place of a built-in or custom role. This field argument is **deprecated**. Update your config to use `inline_role` to maintain compatibility with future versions.",
			"'policy_statements' is deprecated in favor of 'inline_roles'. This field will be removed in the next major release of the LaunchDarkly provider",
		),
		INLINE_ROLES: frameworkPolicyStatementsResourceAttribute(
			false,
			"Define inline custom roles. An array of statements with three attributes: effect, resources, actions. May be used in place of a built-in or custom role. [Using polices](https://docs.launchdarkly.com/home/members/role-policies).",
			"",
		),
	}
}

func (r *AccessTokenResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	priorSchema := schema.Schema{Attributes: accessTokenSchemaAttributes()}
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &priorSchema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var data AccessTokenResourceModel
				resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
				if resp.Diagnostics.HasError() {
					return
				}
				data.InlineRoles = nullIfEmptyList(ctx, data.InlineRoles)
				data.PolicyStatements = nullIfEmptyList(ctx, data.PolicyStatements)
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
			path.MatchRoot(POLICY_STATEMENTS),
			path.MatchRoot(INLINE_ROLES),
		),
	}
}

func (r *AccessTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

// ModifyPlan preserves default_api_version across upgrades from v2.x
// state where the attribute was implicit. Per-attribute
// UseStateForUnknown on default_api_version trips the .token
// inconsistent-sensitive-attr check in TestAccAccessToken_Reset, so
// this resource-level plan modifier replicates UseStateForUnknown's
// behaviour selectively — skipped when expire is changing (the path
// that triggers a token reset and needs the framework's default
// Computed-coupling intact).
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
	if !plan.Expire.Equal(state.Expire) {
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

	// Precedence: policy_statements > inline_roles > custom_roles > role.
	inline, diags := frameworkPolicyStatementsFromList(ctx, plan.PolicyStatements)
	resp.Diagnostics.Append(diags...)
	if len(inline) == 0 {
		inline, diags = frameworkPolicyStatementsFromList(ctx, plan.InlineRoles)
		resp.Diagnostics.Append(diags...)
	}
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
			"access_token must contain either 'role', 'custom_roles', 'policy_statements', or 'inline_roles'.",
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

	inline, diags := frameworkPolicyStatementsFromList(ctx, plan.PolicyStatements)
	resp.Diagnostics.Append(diags...)
	if len(inline) == 0 {
		inline, diags = frameworkPolicyStatementsFromList(ctx, plan.InlineRoles)
		resp.Diagnostics.Append(diags...)
	}

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
	hadInline := !emptyList(state.PolicyStatements) || !emptyList(state.InlineRoles)
	wantInline := !emptyList(plan.PolicyStatements) || !emptyList(plan.InlineRoles)
	if hadInline != wantInline ||
		(wantInline && (!plan.PolicyStatements.Equal(state.PolicyStatements) || !plan.InlineRoles.Equal(state.InlineRoles))) {
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

	// expire reset — must come AFTER readIntoModel so that plan.Token set
	// here is not overwritten by the GET response (which omits the secret).
	if !plan.Expire.Equal(state.Expire) {
		newExpire := plan.Expire.ValueInt64()
		if newExpire != 0 {
			token, err := resetAccessTokenFramework(r.client, id, int(newExpire))
			if err != nil {
				addLdapiError(&resp.Diagnostics, "Failed to reset access token", err)
				return
			}
			plan.Token = stringValueFromPointer(token.Token)
		}
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
		// Preserve which of the two blocks was set in the existing state.
		if !data.PolicyStatements.IsNull() && len(data.PolicyStatements.Elements()) > 0 {
			data.PolicyStatements = stmts
		} else {
			data.InlineRoles = stmts
		}
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

// resetAccessTokenFramework issues a raw HTTP POST to the token Reset
// endpoint. The ldapi v22 client omits Content-Type, so we fall back
// to fallbackClient.
func resetAccessTokenFramework(client *Client, accessTokenID string, expiry int) (ldapi.Token, error) {
	var token ldapi.Token
	endpoint := fmt.Sprintf("%s/api/v2/tokens/%s/reset", client.apiHost, accessTokenID)
	if !strings.HasPrefix(endpoint, "http") {
		endpoint = "https://" + endpoint
	}
	var body io.Reader
	if expiry > 0 {
		raw, err := json.Marshal(map[string]int{"expiry": expiry})
		if err != nil {
			return token, err
		}
		body = bytes.NewBuffer(raw)
	}
	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		return token, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", client.apiKey)

	var resp *http.Response
	err = client.withConcurrency(client.ctx, func() error {
		resp, err = client.fallbackClient.Do(req)
		return err
	})
	if err != nil {
		return token, err
	}
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return token, err
	}
	if err := json.Unmarshal(rawBody, &token); err != nil {
		return token, err
	}
	return token, nil
}
