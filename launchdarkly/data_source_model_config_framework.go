package launchdarkly

// data_source_model_config_framework.go is the terraform-plugin-framework
// implementation of launchdarkly_model_config. All attributes are
// Computed (no validators) per data-source semantics. The schema is
// flat — model_config is all scalars + a string-tag set.

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var _ datasource.DataSource = &ModelConfigDataSource{}

type ModelConfigDataSource struct {
	client *Client
}

// ModelConfigDataSourceModel holds the framework-typed values for a
// model_config data source read.
type ModelConfigDataSourceModel struct {
	ID                 types.String  `tfsdk:"id"`
	ProjectKey         types.String  `tfsdk:"project_key"`
	Key                types.String  `tfsdk:"key"`
	Name               types.String  `tfsdk:"name"`
	ModelID            types.String  `tfsdk:"model_id"`
	Icon               types.String  `tfsdk:"icon"`
	ProviderName       types.String  `tfsdk:"model_provider"`
	Global             types.Bool    `tfsdk:"global"`
	Params             types.String  `tfsdk:"params"`
	CustomParameters   types.String  `tfsdk:"custom_parameters"`
	Tags               types.Set     `tfsdk:"tags"`
	Version            types.Int64   `tfsdk:"version"`
	CostPerInputToken  types.Float64 `tfsdk:"cost_per_input_token"`
	CostPerOutputToken types.Float64 `tfsdk:"cost_per_output_token"`
}

func NewModelConfigDataSource() datasource.DataSource {
	return &ModelConfigDataSource{}
}

func (d *ModelConfigDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_model_config"
}

func (d *ModelConfigDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly model config data source.\n\nThis data source allows you to retrieve AI model configuration information from your LaunchDarkly project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the model config in the format `project_key/model_config_key`.",
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The project key.",
			},
			KEY: schema.StringAttribute{
				Required:    true,
				Description: "The model config's unique key.",
			},
			NAME: schema.StringAttribute{
				Computed:    true,
				Description: "The model config's human-readable name.",
			},
			MODEL_ID: schema.StringAttribute{
				Computed:    true,
				Description: "The model identifier (e.g. `gpt-4`, `claude-3`).",
			},
			ICON: schema.StringAttribute{
				Computed:    true,
				Description: "The icon for the model config.",
			},
			PROVIDER_NAME: schema.StringAttribute{
				Computed:    true,
				Description: "The provider name for the model config (e.g. `openai`, `anthropic`).",
			},
			GLOBAL: schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the model config is available globally.",
			},
			PARAMS: schema.StringAttribute{
				Computed:    true,
				Description: "A JSON string representing the model parameters (e.g. `{\"temperature\": 0.7, \"maxTokens\": 4096}`).",
			},
			CUSTOM_PARAMETERS: schema.StringAttribute{
				Computed:    true,
				Description: "A JSON string representing custom parameters for the model config.",
			},
			TAGS: schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Tags associated with the model config.",
			},
			VERSION: schema.Int64Attribute{
				Computed:    true,
				Description: "The version of the model config.",
			},
			COST_PER_INPUT_TOKEN: schema.Float64Attribute{
				Computed:    true,
				Description: "The cost per input token for the model.",
			},
			COST_PER_OUTPUT_TOKEN: schema.Float64Attribute{
				Computed:    true,
				Description: "The cost per output token for the model.",
			},
		},
	}
}

func (d *ModelConfigDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *ModelConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data ModelConfigDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	key := data.Key.ValueString()

	var modelConfig *ldapi.ModelConfig
	var err error
	err = d.client.withConcurrency(d.client.ctx, func() error {
		modelConfig, _, err = d.client.ld.AIConfigsApi.GetModelConfig(d.client.ctx, projectKey, key).Execute()
		return err
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to get model config with key %q in project %q: %s", key, projectKey, handleLdapiErr(err).Error()),
			"",
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", projectKey, modelConfig.Key))
	data.Key = types.StringValue(modelConfig.Key)
	data.Name = types.StringValue(modelConfig.Name)
	data.ModelID = types.StringValue(modelConfig.Id)
	data.Global = types.BoolValue(modelConfig.Global)
	data.Version = types.Int64Value(int64(modelConfig.Version))

	if modelConfig.Icon != nil {
		data.Icon = types.StringValue(*modelConfig.Icon)
	} else {
		data.Icon = types.StringValue("")
	}

	if modelConfig.Provider != nil {
		data.ProviderName = types.StringValue(*modelConfig.Provider)
	} else {
		data.ProviderName = types.StringValue("")
	}

	paramsJSON, err := mapToJsonString(modelConfig.Params)
	if err != nil {
		resp.Diagnostics.AddError("Failed to serialise params", err.Error())
		return
	}
	data.Params = types.StringValue(paramsJSON)

	customParamsJSON, err := mapToJsonString(modelConfig.CustomParams)
	if err != nil {
		resp.Diagnostics.AddError("Failed to serialise custom_parameters", err.Error())
		return
	}
	data.CustomParameters = types.StringValue(customParamsJSON)

	tagsSet, diags := setFromStringSlice(ctx, modelConfig.Tags)
	resp.Diagnostics.Append(diags...)
	data.Tags = tagsSet

	if modelConfig.CostPerInputToken != nil {
		data.CostPerInputToken = types.Float64Value(*modelConfig.CostPerInputToken)
	} else {
		data.CostPerInputToken = types.Float64Value(0)
	}
	if modelConfig.CostPerOutputToken != nil {
		data.CostPerOutputToken = types.Float64Value(*modelConfig.CostPerOutputToken)
	} else {
		data.CostPerOutputToken = types.Float64Value(0)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
