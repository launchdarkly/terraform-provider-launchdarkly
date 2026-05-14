package launchdarkly

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
	strcase "github.com/stoewer/go-strcase"
)

var _ datasource.DataSource = &AuditLogSubscriptionDataSource{}

type AuditLogSubscriptionDataSource struct {
	client *Client
}

type AuditLogSubscriptionDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	IntegrationKey types.String `tfsdk:"integration_key"`
	Name           types.String `tfsdk:"name"`
	Config         types.Map    `tfsdk:"config"`
	Statements     types.List   `tfsdk:"statements"`
	On             types.Bool   `tfsdk:"on"`
	Tags           types.Set    `tfsdk:"tags"`
}

func NewAuditLogSubscriptionDataSource() datasource.DataSource {
	return &AuditLogSubscriptionDataSource{}
}

func (d *AuditLogSubscriptionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_audit_log_subscription"
}

func (d *AuditLogSubscriptionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly audit log subscription data source.\n\nThis data source allows you to retrieve information about LaunchDarkly audit log subscriptions.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The audit log subscription ID.",
			},
			INTEGRATION_KEY: schema.StringAttribute{
				Required:    true,
				Description: fmt.Sprintf("The integration key. Supported integration keys are %s.", oxfordCommaJoin(getValidIntegrationKeys())),
			},
			NAME: schema.StringAttribute{
				Computed:    true,
				Description: "A human-friendly name for your audit log subscription.",
			},
			CONFIG: schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "The set of configuration fields corresponding to the value defined for `integration_key`.",
			},
			ON: schema.BoolAttribute{
				Computed:    true,
				Description: "Whether or not the subscription is enabled.",
			},
			TAGS: schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Tags associated with the audit log subscription.",
			},
			STATEMENTS: frameworkPolicyStatementsDataSourceAttribute("A block representing the resources to which you wish to subscribe."),
		},
	}
}

func (d *AuditLogSubscriptionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *AuditLogSubscriptionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data AuditLogSubscriptionDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := data.ID.ValueString()
	integrationKey := data.IntegrationKey.ValueString()

	var sub *ldapi.Integration
	var err error
	err = d.client.withConcurrency(d.client.ctx, func() error {
		sub, _, err = d.client.ld.IntegrationAuditLogSubscriptionsApi.GetSubscriptionByID(d.client.ctx, integrationKey, id).Execute()
		return err
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to get integration with ID %q: %s", id, handleLdapiErr(err).Error()),
			"",
		)
		return
	}

	if sub.Id != nil {
		data.ID = types.StringValue(*sub.Id)
	}
	if sub.Name != nil {
		data.Name = types.StringValue(*sub.Name)
	}
	if sub.On != nil {
		data.On = types.BoolValue(*sub.On)
	} else {
		data.On = types.BoolValue(false)
	}

	// Config: emit snake_case keys with string values, matching the
	// SDKv2 representation in configToResourceData (which the
	// underlying framework data source schema declares as Map<String>).
	// Secret-typed fields are suppressed because the API never returns
	// the plaintext value; SDKv2 mirrors this by overwriting with the
	// caller's original (empty) input. For a data source, that means
	// dropping the key from state entirely.
	configFormat := getSubscriptionConfigurationMap()[integrationKey]
	configMap := make(map[string]string, len(sub.Config))
	for k, v := range sub.Config {
		if configFormat[k].IsSecret {
			continue
		}
		key := strcase.SnakeCase(k)
		if b, isBool := v.(bool); isBool {
			configMap[key] = strconv.FormatBool(b)
			continue
		}
		configMap[key] = fmt.Sprintf("%v", v)
	}
	mapVal, diags := types.MapValueFrom(ctx, types.StringType, configMap)
	resp.Diagnostics.Append(diags...)
	data.Config = mapVal

	stmts, diags := frameworkPolicyStatementsValue(ctx, sub.Statements)
	resp.Diagnostics.Append(diags...)
	data.Statements = stmts

	tagsSet, diags := setFromStringSlice(ctx, sub.Tags)
	resp.Diagnostics.Append(diags...)
	data.Tags = tagsSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
