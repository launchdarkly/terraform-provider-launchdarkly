package launchdarkly

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v23"
)

var (
	_ resource.Resource                = &RelayProxyConfigResource{}
	_ resource.ResourceWithImportState = &RelayProxyConfigResource{}
)

type RelayProxyConfigResource struct {
	client *Client
}

type RelayProxyConfigResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Policy     types.List   `tfsdk:"policy"`
	FullKey    types.String `tfsdk:"full_key"`
	DisplayKey types.String `tfsdk:"display_key"`
}

func NewRelayProxyConfigResource() resource.Resource {
	return &RelayProxyConfigResource{}
}

func (r *RelayProxyConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_relay_proxy_configuration"
}

func (r *RelayProxyConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provides a LaunchDarkly Relay Proxy configuration resource for use with the Relay Proxy's [automatic configuration feature](https://docs.launchdarkly.com/home/relay-proxy/automatic-configuration).

-> **Note:** Relay Proxy automatic configuration is available to customers on an Enterprise LaunchDarkly plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).

This resource allows you to create and manage Relay Proxy configurations within your LaunchDarkly organization.

-> **Note:** This resource stores the full plaintext secret for your Relay Proxy configuration's unique key in Terraform state. Be sure your state is configured securely before using this resource. To learn more, read [Sensitive data in state](https://www.terraform.io/docs/state/sensitive-data.html).`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The Relay Proxy configuration's unique ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			NAME: schema.StringAttribute{
				Required:    true,
				Description: "The human-readable name for your Relay Proxy configuration.",
			},
			FULL_KEY: schema.StringAttribute{
				Computed:      true,
				Sensitive:     true,
				Description:   "The Relay Proxy configuration's unique key. Because the `full_key` is only exposed upon creation, it will not be available if the resource is imported.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			DISPLAY_KEY: schema.StringAttribute{
				Computed:      true,
				Description:   "The last 4 characters of the Relay Proxy configuration's unique key.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			POLICY: frameworkPolicyStatementsResourceAttribute(true, "The Relay Proxy configuration's rule policy. This determines what content the Relay Proxy receives. To learn more, read [Understanding policies](https://docs.launchdarkly.com/home/members/role-policies#understanding-policies).", ""),
		},
	}
}

func (r *RelayProxyConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *RelayProxyConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RelayProxyConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, diags := frameworkPolicyStatementsFromList(ctx, plan.Policy)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	post := ldapi.RelayAutoConfigPost{
		Name:   plan.Name.ValueString(),
		Policy: statementPostsToStatementReps(policy),
	}

	var proxyConfig *ldapi.RelayAutoConfigRep
	err := r.client.withConcurrency(r.client.ctx, func() error {
		var e error
		proxyConfig, _, e = r.client.ld.RelayProxyConfigurationsApi.PostRelayAutoConfig(r.client.ctx).RelayAutoConfigPost(post).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to create Relay Proxy configuration", err)
		return
	}

	plan.ID = types.StringValue(proxyConfig.Id)
	plan.FullKey = stringValueFromPointer(proxyConfig.FullKey)
	r.readIntoModel(ctx, proxyConfig.Id, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RelayProxyConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RelayProxyConfigResourceModel
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

func (r *RelayProxyConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RelayProxyConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, diags := frameworkPolicyStatementsFromList(ctx, plan.Policy)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	name := plan.Name.ValueString()
	patch := []ldapi.PatchOperation{
		patchReplace("/name", &name),
		patchReplace("/policy", &policy),
	}
	pwc := ldapi.PatchWithComment{Patch: patch, Comment: ldapi.PtrString("Terraform")}

	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.RelayProxyConfigurationsApi.PatchRelayAutoConfig(r.client.ctx, plan.ID.ValueString()).PatchWithComment(pwc).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to update Relay Proxy configuration", err)
		return
	}

	r.readIntoModel(ctx, plan.ID.ValueString(), &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RelayProxyConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RelayProxyConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, e := r.client.ld.RelayProxyConfigurationsApi.DeleteRelayAutoConfig(r.client.ctx, data.ID.ValueString()).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to delete Relay Proxy configuration", err)
	}
}

func (r *RelayProxyConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *RelayProxyConfigResource) readIntoModel(
	ctx context.Context,
	id string,
	data *RelayProxyConfigResourceModel,
	diags *diag.Diagnostics,
) {
	var proxyConfig *ldapi.RelayAutoConfigRep
	var res *http.Response
	var err error
	err = r.client.withConcurrency(r.client.ctx, func() error {
		proxyConfig, res, err = r.client.ld.RelayProxyConfigurationsApi.GetRelayProxyConfig(r.client.ctx, id).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError("Failed to get Relay Proxy configuration", handleLdapiErr(err).Error())
		return
	}
	data.ID = types.StringValue(proxyConfig.Id)
	data.Name = types.StringValue(proxyConfig.Name)
	data.DisplayKey = types.StringValue(proxyConfig.DisplayKey)
	policyList, d := frameworkPolicyStatementsValue(ctx, proxyConfig.Policy)
	diags.Append(d...)
	data.Policy = policyList
}
