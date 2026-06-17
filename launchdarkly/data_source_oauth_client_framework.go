package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var _ datasource.DataSource = &OAuthClientDataSource{}

type OAuthClientDataSource struct {
	client *Client
}

type OAuthClientDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	ClientID     types.String `tfsdk:"client_id"`
	Name         types.String `tfsdk:"name"`
	RedirectURI  types.String `tfsdk:"redirect_uri"`
	Description  types.String `tfsdk:"description"`
	AccountID    types.String `tfsdk:"account_id"`
	CreationDate types.Int64  `tfsdk:"creation_date"`
}

func NewOAuthClientDataSource() datasource.DataSource {
	return &OAuthClientDataSource{}
}

func (d *OAuthClientDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oauth_client"
}

func (d *OAuthClientDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly OAuth 2.0 client data source.\n\nThis data source allows you to retrieve information about a registered LaunchDarkly OAuth 2.0 client by its client ID.\n\n-> **Note:** The client secret is not available through this data source. LaunchDarkly only returns it once, upon creation.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The OAuth 2.0 client's unique client ID. This is the same value as `client_id`.",
			},
			CLIENT_ID: schema.StringAttribute{
				Required:    true,
				Description: "The OAuth 2.0 client's unique client ID.",
			},
			NAME: schema.StringAttribute{
				Computed:    true,
				Description: "The human-friendly name for the OAuth 2.0 client.",
			},
			REDIRECT_URI: schema.StringAttribute{
				Computed:    true,
				Description: "The redirect URI for the OAuth 2.0 client.",
			},
			DESCRIPTION: schema.StringAttribute{
				Computed:    true,
				Description: "The description of the OAuth 2.0 client.",
			},
			ACCOUNT_ID: schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the account the OAuth 2.0 client is registered under.",
			},
			CREATION_DATE: schema.Int64Attribute{
				Computed:    true,
				Description: "The OAuth 2.0 client's creation date represented as a Unix epoch timestamp in milliseconds.",
			},
		},
	}
}

func (d *OAuthClientDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *OAuthClientDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data OAuthClientDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientID := data.ClientID.ValueString()

	var client *ldapi.Client
	var err error
	err = d.client.withConcurrency(d.client.ctx, func() error {
		client, _, err = d.client.ld.OAuth2ClientsApi.GetOAuthClientById(d.client.ctx, clientID).Execute()
		return err
	})
	if err != nil {
		resp.Diagnostics.AddError(handleLdapiErr(err).Error(), "")
		return
	}

	data.ID = types.StringValue(client.ClientId)
	data.ClientID = types.StringValue(client.ClientId)
	data.AccountID = types.StringValue(client.AccountId)
	data.Name = types.StringValue(client.Name)
	data.RedirectURI = types.StringValue(client.RedirectUri)
	data.CreationDate = types.Int64Value(client.CreationDate)
	if client.Description != nil {
		data.Description = types.StringValue(*client.Description)
	} else {
		data.Description = types.StringValue("")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
