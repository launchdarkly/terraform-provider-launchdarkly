package launchdarkly

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v23"
)

var (
	_ resource.Resource                = &OAuthClientResource{}
	_ resource.ResourceWithImportState = &OAuthClientResource{}
)

type OAuthClientResource struct {
	client *Client
}

type OAuthClientResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	RedirectURI  types.String `tfsdk:"redirect_uri"`
	Description  types.String `tfsdk:"description"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	AccountID    types.String `tfsdk:"account_id"`
	CreationDate types.Int64  `tfsdk:"creation_date"`
}

func NewOAuthClientResource() resource.Resource {
	return &OAuthClientResource{}
}

func (r *OAuthClientResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oauth_client"
}

func (r *OAuthClientResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provides a LaunchDarkly OAuth 2.0 client resource.

This resource allows you to register and manage LaunchDarkly OAuth 2.0 clients. OAuth 2.0 clients let you build custom integrations that use LaunchDarkly as an identity provider. This is an account-level resource. Your account may register more than one OAuth 2.0 client.

-> **Note:** The client secret is returned by LaunchDarkly only once, when the client is first created. It is stored in Terraform state but cannot be read back from the API afterward, so it is not populated on import. Be sure your state is configured securely before using this resource. To learn more, read [Sensitive data in state](https://www.terraform.io/docs/state/sensitive-data.html).`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The OAuth 2.0 client's unique client ID. This is the same value as `client_id`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			NAME: schema.StringAttribute{
				Required:    true,
				Description: "The human-friendly name for your OAuth 2.0 client.",
			},
			REDIRECT_URI: schema.StringAttribute{
				Required:    true,
				Description: "The redirect URI for your OAuth 2.0 client. This should be an absolute URL conforming with the standard HTTPS protocol.",
			},
			DESCRIPTION: schema.StringAttribute{
				Optional:    true,
				Description: "A description of your OAuth 2.0 client.",
			},
			CLIENT_ID: schema.StringAttribute{
				Computed:      true,
				Description:   "The OAuth 2.0 client's unique client ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			CLIENT_SECRET: schema.StringAttribute{
				Computed:      true,
				Sensitive:     true,
				Description:   "The OAuth 2.0 client secret. LaunchDarkly returns this value only once, upon creation, so it cannot be recovered on a subsequent read or import.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			ACCOUNT_ID: schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of the account the OAuth 2.0 client is registered under.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			CREATION_DATE: schema.Int64Attribute{
				Computed:      true,
				Description:   "The OAuth 2.0 client's creation date represented as a Unix epoch timestamp in milliseconds.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *OAuthClientResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *OAuthClientResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OAuthClientResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()
	redirectURI := plan.RedirectURI.ValueString()
	post := ldapi.OauthClientPost{
		Name:        &name,
		RedirectUri: &redirectURI,
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		desc := plan.Description.ValueString()
		post.Description = &desc
	}

	var client *ldapi.Client
	err := r.client.withConcurrency(r.client.ctx, func() error {
		var e error
		client, _, e = r.client.ld.OAuth2ClientsApi.CreateOAuth2Client(r.client.ctx).OauthClientPost(post).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to create OAuth 2.0 client", err)
		return
	}

	// The client secret is only ever returned on create. Capture it here;
	// readIntoModel intentionally leaves it untouched on subsequent reads.
	plan.ID = types.StringValue(client.ClientId)
	plan.ClientSecret = stringValueFromPointer(client.ClientSecret)

	r.readIntoModel(ctx, client.ClientId, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OAuthClientResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data OAuthClientResourceModel
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

func (r *OAuthClientResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state OAuthClientResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only name, description, and redirectUri are patchable.
	name := plan.Name.ValueString()
	redirectURI := plan.RedirectURI.ValueString()
	patch := []ldapi.PatchOperation{
		patchReplace("/name", &name),
		patchReplace("/redirectUri", &redirectURI),
	}
	if !plan.Description.Equal(state.Description) {
		if plan.Description.IsNull() {
			patch = append(patch, patchRemove("/description"))
		} else {
			desc := plan.Description.ValueString()
			patch = append(patch, patchReplace("/description", &desc))
		}
	}

	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.OAuth2ClientsApi.PatchOAuthClient(r.client.ctx, plan.ID.ValueString()).PatchOperation(patch).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to update OAuth 2.0 client", err)
		return
	}

	r.readIntoModel(ctx, plan.ID.ValueString(), &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OAuthClientResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data OAuthClientResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.withConcurrency(r.client.ctx, func() error {
		res, e := r.client.ld.OAuth2ClientsApi.DeleteOAuthClient(r.client.ctx, data.ID.ValueString()).Execute()
		if isStatusNotFound(res) {
			return nil
		}
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to delete OAuth 2.0 client", err)
	}
}

func (r *OAuthClientResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// readIntoModel refreshes data from the API. It never touches ClientSecret,
// which LaunchDarkly only returns on create; the prior state value (if any)
// is preserved.
func (r *OAuthClientResource) readIntoModel(
	ctx context.Context,
	id string,
	data *OAuthClientResourceModel,
	diags *diag.Diagnostics,
) {
	var client *ldapi.Client
	var res *http.Response
	var err error
	err = r.client.withConcurrency(r.client.ctx, func() error {
		client, res, err = r.client.ld.OAuth2ClientsApi.GetOAuthClientById(r.client.ctx, id).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError("Failed to get OAuth 2.0 client", handleLdapiErr(err).Error())
		return
	}
	data.ID = types.StringValue(client.ClientId)
	data.ClientID = types.StringValue(client.ClientId)
	data.AccountID = types.StringValue(client.AccountId)
	data.Name = types.StringValue(client.Name)
	data.RedirectURI = types.StringValue(client.RedirectUri)
	data.CreationDate = types.Int64Value(client.CreationDate)
	// Optional-only attr: null-when-empty for plan-apply consistency.
	data.Description = stringValueOrNullFromPointer(client.Description)
}
