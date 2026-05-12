package launchdarkly

import (
	"context"
	"net/http"

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
	ID               types.String `tfsdk:"id"`
	Key              types.String `tfsdk:"key"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	BasePermissions  types.String `tfsdk:"base_permissions"`
	Policy           types.Set    `tfsdk:"policy"`
	PolicyStatements types.List   `tfsdk:"policy_statements"`
}

func NewCustomRoleResource() resource.Resource {
	return &CustomRoleResource{}
}

func (r *CustomRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_role"
}

func (r *CustomRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly custom role resource (Enterprise plan).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			KEY: schema.StringAttribute{
				Required:    true,
				Description: "A unique key used to reference the custom role in code.",
				Validators:  []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			NAME: schema.StringAttribute{
				Required:    true,
				Description: "A name for the custom role.",
			},
			DESCRIPTION: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Description of the custom role.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			BASE_PERMISSIONS: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("reader"),
				Description: "Base permission level (`reader` or `no_access`). Defaults to `reader`.",
				Validators: []validator.String{
					oneOfValidator{allowed: []string{"reader", "no_access"}},
				},
			},
		},
		Blocks: map[string]schema.Block{
			POLICY: schema.SetNestedBlock{
				Description:        "Deprecated: use policy_statements.",
				DeprecationMessage: "'policy' is now deprecated. Please migrate to 'policy_statements' to maintain future compatability.",
				NestedObject: schema.NestedBlockObject{
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
			POLICY_STATEMENTS: frameworkPolicyStatementsResourceBlock(false, "Policy statements defining the role's permissions.", ""),
		},
	}
}

func (r *CustomRoleResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{customRolePolicyConflictValidator{}}
}

type customRolePolicyConflictValidator struct{}

func (customRolePolicyConflictValidator) Description(context.Context) string {
	return "policy and policy_statements are mutually exclusive"
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
	if policySet && stmtSet {
		resp.Diagnostics.AddAttributeError(
			path.Root(POLICY_STATEMENTS),
			"Conflicting policy fields",
			"policy (deprecated) and policy_statements cannot both be set.",
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

func (r *CustomRoleResource) policiesFromModel(ctx context.Context, data *CustomRoleResourceModel, diags interface {
	Append(...interface{ AppendDiagnostic() })
},
) []ldapi.StatementPost {
	// Use the new policy_statements if set; otherwise fall back to the
	// deprecated policy block.
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
		stmt := ldapi.StatementPost{Effect: p.Effect}
		stmt.SetResources(p.Resources)
		stmt.SetActions(p.Actions)
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

	policies := r.policiesFromModel(ctx, &plan, nil)

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
	policies := r.policiesFromModel(ctx, &plan, nil)

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
	diags interface{ AddError(string, string) },
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
	if customRole.Description != nil {
		data.Description = types.StringValue(*customRole.Description)
	} else {
		data.Description = types.StringValue("")
	}
	if customRole.BasePermissions != nil {
		data.BasePermissions = types.StringValue(*customRole.BasePermissions)
	} else {
		data.BasePermissions = types.StringValue("reader")
	}

	// Refresh whichever of {policy, policy_statements} was already set.
	// If neither was set, default to policy_statements (the modern path).
	policySet := !data.Policy.IsNull() && len(data.Policy.Elements()) > 0
	if policySet {
		// Refresh deprecated policy block.
		// (We need an attr.Value list for the Set.)
		// Leave existing policy set as-is; the API doesn't distinguish
		// between policy and policy_statements at read time, so keeping
		// state stable avoids drift.
		_ = policySet
	} else {
		stmts, d := frameworkPolicyStatementsValue(ctx, customRole.Policy)
		if d.HasError() {
			for _, e := range d.Errors() {
				diags.AddError(e.Summary(), e.Detail())
			}
		}
		data.PolicyStatements = stmts
	}
}
