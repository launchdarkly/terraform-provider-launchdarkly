package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var _ datasource.DataSource = &FlagTriggerDataSource{}

type FlagTriggerDataSource struct {
	client *Client
}

type FlagTriggerDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectKey     types.String `tfsdk:"project_key"`
	EnvKey         types.String `tfsdk:"env_key"`
	FlagKey        types.String `tfsdk:"flag_key"`
	IntegrationKey types.String `tfsdk:"integration_key"`
	Instructions   types.Object `tfsdk:"instructions"`
	TriggerURL     types.String `tfsdk:"trigger_url"`
	MaintainerID   types.String `tfsdk:"maintainer_id"`
	Enabled        types.Bool   `tfsdk:"enabled"`
}

var flagTriggerInstructionAttrTypes = map[string]attr.Type{
	KIND: types.StringType,
}

func NewFlagTriggerDataSource() datasource.DataSource {
	return &FlagTriggerDataSource{}
}

func (d *FlagTriggerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flag_trigger"
}

func (d *FlagTriggerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly flag trigger data source.\n\n-> **Note:** Flag triggers are available to customers on an Enterprise LaunchDarkly plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).\n\nThis data source allows you to retrieve information about flag triggers from your LaunchDarkly organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The Terraform trigger ID. The unique trigger ID can be found in your saved trigger URL:\n\n```\nhttps://app.launchdarkly.com/webhook/triggers/THIS_IS_YOUR_TRIGGER_ID/aff25a53-17d9-4112-a9b8-12718d1a2e79\n```\n\nPlease note that if you did not save this upon creation of the resource, you will have to reset it to get a new value, which can cause breaking changes.",
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The unique key of the project encompassing the associated flag.",
			},
			ENV_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The unique key of the environment the flag trigger will work in.",
			},
			FLAG_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The unique key of the associated flag.",
			},
			INTEGRATION_KEY: schema.StringAttribute{
				Computed:    true,
				Description: fmt.Sprintf("The unique identifier of the integration you intend to set your trigger up with. Currently supported are %s. `generic-trigger` should be used for integrations not explicitly supported.", oxfordCommaJoin(VALID_TRIGGER_INTEGRATIONS)),
			},
			TRIGGER_URL: schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The unique URL used to invoke the trigger.",
			},
			MAINTAINER_ID: schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the member responsible for maintaining the flag trigger.",
			},
			ENABLED: schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the trigger is currently active or not.",
			},
			INSTRUCTIONS: schema.SingleNestedAttribute{
				Computed:    true,
				Description: "The instruction containing the action to perform when invoking the trigger. Currently supported flag actions are `turnFlagOn` and `turnFlagOff`.",
				Attributes: map[string]schema.Attribute{
					KIND: schema.StringAttribute{
						Computed:    true,
						Description: "The action to perform when triggering. Currently supported flag actions are `turnFlagOn` and `turnFlagOff`.",
					},
				},
			},
		},
	}
}

func (d *FlagTriggerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *FlagTriggerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data FlagTriggerDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	triggerID := data.ID.ValueString()
	projectKey := data.ProjectKey.ValueString()
	envKey := data.EnvKey.ValueString()
	flagKey := data.FlagKey.ValueString()

	var trigger *ldapi.TriggerWorkflowRep
	var err error
	// integration_key is computed-only on the data source — start empty.
	integrationKey := ""
	err = d.client.withConcurrency(d.client.ctx, func() error {
		trigger, _, err = d.client.ld.FlagTriggersApi.GetTriggerWorkflowById(d.client.ctx, projectKey, flagKey, envKey, triggerID).Execute()
		return err
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to get %s trigger with ID %q: %s", integrationKey, triggerID, handleLdapiErr(err).Error()),
			"",
		)
		return
	}

	if trigger.Id != nil {
		data.ID = types.StringValue(*trigger.Id)
	}
	data.ProjectKey = types.StringValue(projectKey)
	data.EnvKey = types.StringValue(envKey)
	data.FlagKey = types.StringValue(flagKey)
	if trigger.IntegrationKey != nil {
		data.IntegrationKey = types.StringValue(*trigger.IntegrationKey)
	}
	if trigger.MaintainerId != nil {
		data.MaintainerID = types.StringValue(*trigger.MaintainerId)
	} else {
		data.MaintainerID = types.StringValue("")
	}
	if trigger.Enabled != nil {
		data.Enabled = types.BoolValue(*trigger.Enabled)
	} else {
		data.Enabled = types.BoolValue(false)
	}
	data.TriggerURL = types.StringValue("")

	data.Instructions = flagTriggerInstructionObject(trigger.Instructions)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
