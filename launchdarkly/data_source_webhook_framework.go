package launchdarkly

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var _ datasource.DataSource = &WebhookDataSource{}

type WebhookDataSource struct {
	client *Client
}

type WebhookDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	URL        types.String `tfsdk:"url"`
	Secret     types.String `tfsdk:"secret"`
	On         types.Bool   `tfsdk:"on"`
	Name       types.String `tfsdk:"name"`
	Statements types.List   `tfsdk:"statements"`
	Tags       types.Set    `tfsdk:"tags"`
}

func NewWebhookDataSource() datasource.DataSource {
	return &WebhookDataSource{}
}

func (d *WebhookDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (d *WebhookDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly webhook data source.\n\nThis data source allows you to retrieve webhook information from your LaunchDarkly organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The unique webhook ID.",
			},
			URL: schema.StringAttribute{
				Computed:    true,
				Description: "The URL of the remote webhook.",
			},
			SECRET: schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The secret used to sign the webhook.",
			},
			ON: schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the webhook is enabled.",
			},
			NAME: schema.StringAttribute{
				Computed:    true,
				Description: "The webhook's human-readable name.",
			},
			TAGS: schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Tags associated with the webhook.",
			},
		},
		Blocks: map[string]schema.Block{
			STATEMENTS: frameworkPolicyStatementsDataSourceBlock("List of policy statement blocks used to filter webhook events. For more information on webhook policy filters read [Adding a policy filter](https://docs.launchdarkly.com/integrations/webhooks#adding-a-policy-filter)."),
		},
	}
}

func (d *WebhookDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *WebhookDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data WebhookDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()

	var webhook *ldapi.Webhook
	var res *http.Response
	var err error
	err = d.client.withConcurrency(d.client.ctx, func() error {
		webhook, res, err = d.client.ld.WebhooksApi.GetWebhook(d.client.ctx, id).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			resp.Diagnostics.AddError("Webhook not found", fmt.Sprintf("Webhook with id %q not found.", id))
			return
		}
		addLdapiError(&resp.Diagnostics, "Failed to get webhook", err)
		return
	}

	data.ID = types.StringValue(webhook.Id)
	data.URL = types.StringValue(webhook.Url)
	data.On = types.BoolValue(webhook.On)
	if webhook.Name != nil {
		data.Name = types.StringValue(*webhook.Name)
	} else {
		data.Name = types.StringValue("")
	}
	if webhook.Secret != nil {
		data.Secret = types.StringValue(*webhook.Secret)
	} else {
		data.Secret = types.StringValue("")
	}

	tagsSet, diags := setFromStringSlice(ctx, webhook.Tags)
	resp.Diagnostics.Append(diags...)
	data.Tags = tagsSet

	stmts, diags := frameworkPolicyStatementsValue(ctx, webhook.Statements)
	resp.Diagnostics.Append(diags...)
	data.Statements = stmts

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
