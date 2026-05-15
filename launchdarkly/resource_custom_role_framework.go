package launchdarkly

import (
	"context"
	"net/http"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var (
	_ resource.Resource                = &CustomRoleResource{}
	_ resource.ResourceWithImportState = &CustomRoleResource{}
)

type CustomRoleResource struct {
	client *Client
}

type CustomRoleResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Key                  types.String `tfsdk:"key"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	BasePermissions      types.String `tfsdk:"base_permissions"`
	Policy               types.Set    `tfsdk:"policy"`
	PolicyStatements     types.List   `tfsdk:"policy_statements"`
	PolicyStatementsJSON types.String `tfsdk:"policy_statements_json"`
}

func NewCustomRoleResource() resource.Resource {
	return &CustomRoleResource{}
}

func (r *CustomRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_role"
}

func (r *CustomRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly custom role resource.\n\n-> **Note:** Custom roles are available to customers on an Enterprise LaunchDarkly plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).\n\nThis resource allows you to create and manage custom roles within your LaunchDarkly organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			KEY: schema.StringAttribute{
				Required:    true,
				Description: addForceNewDescription("A unique key that will be used to reference the custom role in your code.", true),
				Validators:  []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			NAME: schema.StringAttribute{
				Required:    true,
				Description: "A name for the custom role. This must be unique within your organization.",
			},
			DESCRIPTION: schema.StringAttribute{
				Optional:    true,
				Description: "Description of the custom role.",
			},
			BASE_PERMISSIONS: schema.StringAttribute{
				Optional:    true,
				Default:     stringdefault.StaticString("reader"),
				Computed:    true,
				Description: "The base permission level - either `reader` or `no_access`. While newer API versions default to `no_access`, this field defaults to `reader` in keeping with previous API versions.",
				Validators: []validator.String{
					oneOfValidator{allowed: []string{"reader", "no_access"}},
				},
			},
			POLICY: schema.SetNestedAttribute{
				Optional:           true,
				DeprecationMessage: "'policy' is now deprecated. Please migrate to 'policy_statements' to maintain future compatability.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						RESOURCES: schema.ListAttribute{
							Required:    true,
							ElementType: types.StringType,
						},
						ACTIONS: schema.ListAttribute{
							Required:    true,
							ElementType: types.StringType,
						},
						EFFECT: schema.StringAttribute{
							Required: true,
						},
					},
				},
			},
			POLICY_STATEMENTS: frameworkPolicyStatementsResourceAttribute(false, "An array of the policy statements that define the permissions for the custom role. This field accepts [role attributes](https://docs.launchdarkly.com/home/getting-started/vocabulary#role-attribute). To use role attributes, use the syntax `$${roleAttribute/<YOUR_ROLE_ATTRIBUTE>}` in lieu of your usual resource keys.", ""),
			POLICY_STATEMENTS_JSON: schema.StringAttribute{
				Optional:    true,
				Description: "Policy statements expressed as a single JSON document — an array of statement objects with the same keys as the `policy_statements` attribute (`resources`, `not_resources`, `actions`, `not_actions`, `effect`). Mutually exclusive with `policy_statements` and `policy`. Use this form when reading the policy from a file or templating it dynamically (for example with `jsonencode(...)` or `file(\"policy.json\")`). To use [role attributes](https://docs.launchdarkly.com/home/getting-started/vocabulary#role-attribute), escape the `$` as `$${roleAttribute/<YOUR_ROLE_ATTRIBUTE>}` inside HCL strings.",
				Validators:  []validator.String{jsonStringValidator{}},
				PlanModifiers: []planmodifier.String{
					jsonNormalizePlanModifier{},
				},
			},
		},
	}
}

func (r *CustomRoleResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{customRolePolicyConflictValidator{}}
}

type customRolePolicyConflictValidator struct{}

func (customRolePolicyConflictValidator) Description(context.Context) string {
	return "policy, policy_statements, and policy_statements_json are mutually exclusive"
}
func (customRolePolicyConflictValidator) MarkdownDescription(ctx context.Context) string {
	return ""
}
func (customRolePolicyConflictValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data CustomRoleResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	policySet := !data.Policy.IsNull() && !data.Policy.IsUnknown() && len(data.Policy.Elements()) > 0
	stmtSet := !data.PolicyStatements.IsNull() && !data.PolicyStatements.IsUnknown() && len(data.PolicyStatements.Elements()) > 0
	jsonSet := !data.PolicyStatementsJSON.IsNull() && !data.PolicyStatementsJSON.IsUnknown() && strings.TrimSpace(data.PolicyStatementsJSON.ValueString()) != ""
	if policySet && stmtSet {
		resp.Diagnostics.AddAttributeError(
			path.Root(POLICY_STATEMENTS),
			"Conflicting policy fields",
			"policy (deprecated) and policy_statements cannot both be set.",
		)
	}
	if jsonSet && stmtSet {
		resp.Diagnostics.AddAttributeError(
			path.Root(POLICY_STATEMENTS_JSON),
			"Conflicting policy fields",
			"policy_statements_json and policy_statements cannot both be set.",
		)
	}
	if jsonSet && policySet {
		resp.Diagnostics.AddAttributeError(
			path.Root(POLICY_STATEMENTS_JSON),
			"Conflicting policy fields",
			"policy_statements_json and policy (deprecated) cannot both be set.",
		)
	}
}

func (r *CustomRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

type customRolePolicyModel struct {
	Resources []string `tfsdk:"resources"`
	Actions   []string `tfsdk:"actions"`
	Effect    string   `tfsdk:"effect"`
}

func (r *CustomRoleResource) policiesFromModel(ctx context.Context, data *CustomRoleResourceModel, diags *diag.Diagnostics) []ldapi.StatementPost {
	// Prefer policy_statements_json when set, then policy_statements, then
	// the deprecated policy block. ConfigValidators guarantees at most one
	// is set at a time.
	if !data.PolicyStatementsJSON.IsNull() && !data.PolicyStatementsJSON.IsUnknown() {
		if raw := strings.TrimSpace(data.PolicyStatementsJSON.ValueString()); raw != "" {
			out, err := policyStatementsFromJSON(raw)
			if err != nil {
				diags.AddAttributeError(path.Root(POLICY_STATEMENTS_JSON), "Invalid policy_statements_json", err.Error())
				return nil
			}
			return out
		}
	}
	if !data.PolicyStatements.IsNull() && len(data.PolicyStatements.Elements()) > 0 {
		out, _ := frameworkPolicyStatementsFromList(ctx, data.PolicyStatements)
		return out
	}
	if data.Policy.IsNull() || data.Policy.IsUnknown() {
		return nil
	}
	var policies []customRolePolicyModel
	data.Policy.ElementsAs(ctx, &policies, false)
	out := make([]ldapi.StatementPost, 0, len(policies))
	for _, p := range policies {
		resources := make([]string, len(p.Resources))
		copy(resources, p.Resources)
		sort.Strings(resources)

		actions := make([]string, len(p.Actions))
		copy(actions, p.Actions)
		sort.Strings(actions)

		stmt := ldapi.StatementPost{Effect: p.Effect}
		stmt.SetResources(resources)
		stmt.SetActions(actions)
		out = append(out, stmt)
	}
	return out
}

func (r *CustomRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CustomRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key := plan.Key.ValueString()
	name := plan.Name.ValueString()
	desc := plan.Description.ValueString()
	basePerms := plan.BasePermissions.ValueString()

	policies := r.policiesFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	body := ldapi.CustomRolePost{
		Key:         key,
		Name:        name,
		Description: ldapi.PtrString(desc),
		Policy:      policies,
	}
	if basePerms != "" {
		body.BasePermissions = ldapi.PtrString(basePerms)
	}

	var created *ldapi.CustomRole
	err := r.client.withConcurrency(r.client.ctx, func() error {
		var e error
		created, _, e = r.client.ld.CustomRolesApi.PostCustomRole(r.client.ctx).CustomRolePost(body).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to create custom role", err)
		return
	}
	id := key
	if created != nil && created.Key != "" {
		id = created.Key
	}
	plan.ID = types.StringValue(id)

	r.readIntoModel(ctx, id, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CustomRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CustomRoleResourceModel
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

func (r *CustomRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CustomRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key := plan.Key.ValueString()
	name := plan.Name.ValueString()
	desc := plan.Description.ValueString()
	basePerms := plan.BasePermissions.ValueString()
	policies := r.policiesFromModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	patch := ldapi.PatchWithComment{Patch: []ldapi.PatchOperation{
		patchReplace("/name", &name),
		patchReplace("/description", &desc),
		patchReplace("/policy", &policies),
	}}
	if basePerms != "" {
		patch.Patch = append(patch.Patch, patchReplace("/basePermissions", &basePerms))
	}

	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.CustomRolesApi.PatchCustomRole(r.client.ctx, key).PatchWithComment(patch).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to update custom role", err)
		return
	}

	r.readIntoModel(ctx, key, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CustomRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CustomRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, e := r.client.ld.CustomRolesApi.DeleteCustomRole(r.client.ctx, data.ID.ValueString()).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to delete custom role", err)
	}
}

func (r *CustomRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(KEY), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *CustomRoleResource) readIntoModel(
	ctx context.Context,
	id string,
	data *CustomRoleResourceModel,
	diags *diag.Diagnostics,
) {
	var customRole *ldapi.CustomRole
	var res *http.Response
	var err error
	err = r.client.withConcurrency(r.client.ctx, func() error {
		customRole, res, err = r.client.ld.CustomRolesApi.GetCustomRole(r.client.ctx, id).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError("Failed to get custom role", handleLdapiErr(err).Error())
		return
	}
	data.ID = types.StringValue(customRole.Key)
	data.Key = types.StringValue(customRole.Key)
	data.Name = types.StringValue(customRole.Name)
	// Optional-only attr: write null when API returns empty/nil so
	// terraform-core's plan-apply consistency check doesn't see
	// plan(null) vs apply(""). See stringValueOrNullFromPointer.
	data.Description = stringValueOrNullFromPointer(customRole.Description)
	if customRole.BasePermissions != nil {
		data.BasePermissions = types.StringValue(*customRole.BasePermissions)
	} else {
		data.BasePermissions = types.StringValue("reader")
	}

	// Refresh whichever of {policy, policy_statements, policy_statements_json}
	// was already set. If none was set, default to policy_statements (the
	// modern path).
	policySet := !data.Policy.IsNull() && len(data.Policy.Elements()) > 0
	jsonSet := !data.PolicyStatementsJSON.IsNull() && !data.PolicyStatementsJSON.IsUnknown() && strings.TrimSpace(data.PolicyStatementsJSON.ValueString()) != ""
	if jsonSet {
		encoded, jerr := policyStatementsToJSON(customRole.Policy)
		if jerr != nil {
			diags.AddError("Failed to encode policy_statements_json", jerr.Error())
			return
		}
		// Preserve prior value when semantically equivalent (avoids
		// plan-apply consistency check failures from key reordering).
		prior := data.PolicyStatementsJSON.ValueString()
		if !jsonEqual(prior, encoded) {
			data.PolicyStatementsJSON = types.StringValue(encoded)
		}
		// Clear the alternate forms so they don't show diffs.
		data.PolicyStatements = types.ListNull(types.ObjectType{AttrTypes: frameworkPolicyStatementsObjectAttrTypes})
		data.Policy = types.SetNull(data.Policy.ElementType(ctx))
		return
	}
	if policySet {
		// Refresh deprecated policy block from API response.
		// Sort Resources and Actions alphabetically to match policyHash parity
		// (SDKv2 sorts in policiesFromResourceData/policyFromResourceData).
		elems := make([]customRolePolicyModel, 0, len(customRole.Policy))
		for _, stmt := range customRole.Policy {
			resources := make([]string, len(stmt.Resources))
			copy(resources, stmt.Resources)
			sort.Strings(resources)

			actions := make([]string, len(stmt.Actions))
			copy(actions, stmt.Actions)
			sort.Strings(actions)

			elems = append(elems, customRolePolicyModel{
				Resources: resources,
				Actions:   actions,
				Effect:    stmt.Effect,
			})
		}
		setVal, d := types.SetValueFrom(ctx, data.Policy.ElementType(ctx), elems)
		diags.Append(d...)
		if diags.HasError() {
			return
		}
		data.Policy = setVal
	} else {
		stmts, d := frameworkPolicyStatementsValue(ctx, customRole.Policy)
		diags.Append(d...)
		data.PolicyStatements = stmts
	}
}
