package launchdarkly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var (
	_ resource.Resource                = &AccessTokenResource{}
	_ resource.ResourceWithImportState = &AccessTokenResource{}
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
		Description: "Provides a LaunchDarkly access token resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			NAME: schema.StringAttribute{
				Optional:    true,
				Description: "Human-friendly name for the access token.",
			},
			ROLE: schema.StringAttribute{
				Optional:    true,
				Description: "Built-in role: reader, writer, or admin.",
				Validators: []validator.String{
					oneOfValidator{allowed: []string{"reader", "writer", "admin"}},
				},
			},
			CUSTOM_ROLES: schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Custom role IDs used as access limits.",
			},
			SERVICE_TOKEN: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the token is a service token.",
				PlanModifiers: []planmodifier.Bool{
					&forceReplaceBool{},
				},
			},
			DEFAULT_API_VERSION: schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Default API version for this token.",
			},
			TOKEN: schema.StringAttribute{
				Computed:      true,
				Sensitive:     true,
				Description:   "The plaintext token. Only exposed on create / reset.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			EXPIRE: schema.Int64Attribute{
				Optional:           true,
				Description:        "Deprecated. Expiry epoch — setting it resets the token.",
				DeprecationMessage: "'expire' is deprecated and will be removed in the next major release of the LaunchDarkly provider",
			},
		},
		Blocks: map[string]schema.Block{
			POLICY_STATEMENTS: frameworkPolicyStatementsResourceBlock(
				false,
				"Deprecated inline-role definition; use inline_roles.",
				"'policy_statements' is deprecated in favor of 'inline_roles'. This field will be removed in the next major release of the LaunchDarkly provider",
			),
			INLINE_ROLES: frameworkPolicyStatementsResourceBlock(false, "Inline custom-role policy statements.", ""),
		},
	}
}

// forceReplaceBool is a tiny inline replacement for
// boolplanmodifier.RequiresReplace; vendored to avoid pulling in the
// import while keeping the schema declaration above tidy.
type forceReplaceBool struct{}

func (forceReplaceBool) Description(context.Context) string         { return "service_token forces replace" }
func (forceReplaceBool) MarkdownDescription(context.Context) string { return "" }
func (forceReplaceBool) PlanModifyBool(_ context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if req.StateValue.IsNull() || req.PlanValue.IsNull() {
		return
	}
	if !req.PlanValue.Equal(req.StateValue) {
		resp.RequiresReplace = true
	}
}

func (r *AccessTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
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

	// SDKv2 precedence: policy_statements > inline_roles > custom_roles > role.
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
	if !plan.PolicyStatements.Equal(state.PolicyStatements) || !plan.InlineRoles.Equal(state.InlineRoles) {
		if len(inline) == 0 {
			patch = append(patch, patchRemove("/inlineRole"))
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

	// expire reset
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
	diags interface{ AddError(string, string) },
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
	data.Name = stringValueFromPointer(accessToken.Name)
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

// resetAccessToken: ported from SDKv2 — the official Reset endpoint in
// ldapi v22 still omits Content-Type, so we issue a raw HTTP POST via
// fallbackClient.
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
