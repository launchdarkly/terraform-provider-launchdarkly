package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v23"
)

var _ datasource.DataSource = &AIConfigVariationDataSource{}

type AIConfigVariationDataSource struct {
	client *Client
}

type AIConfigVariationDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectKey     types.String `tfsdk:"project_key"`
	AIConfigKey    types.String `tfsdk:"config_key"`
	Key            types.String `tfsdk:"key"`
	Name           types.String `tfsdk:"name"`
	Messages       types.List   `tfsdk:"messages"`
	Model          types.String `tfsdk:"model"`
	ModelConfigKey types.String `tfsdk:"model_config_key"`
	Description    types.String `tfsdk:"description"`
	Instructions   types.String `tfsdk:"instructions"`
	ToolKeys       types.Set    `tfsdk:"tool_keys"`
	State          types.String `tfsdk:"state"`
	VariationID    types.String `tfsdk:"variation_id"`
	Version        types.Int64  `tfsdk:"version"`
	CreationDate   types.Int64  `tfsdk:"creation_date"`
}

var aiConfigVariationMessageAttrTypes = map[string]attr.Type{
	ROLE:    types.StringType,
	CONTENT: types.StringType,
}

func NewAIConfigVariationDataSource() datasource.DataSource {
	return &AIConfigVariationDataSource{}
}

func (d *AIConfigVariationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_config_variation"
}

func (d *AIConfigVariationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly AI Config variation data source.\n\nThis data source allows you to retrieve AI Config variation information from your LaunchDarkly project.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, Description: "The ID in the format `project_key/config_key/key`."},
			PROJECT_KEY:   schema.StringAttribute{Required: true, Description: "The project key."},
			AI_CONFIG_KEY: schema.StringAttribute{Required: true, Description: "The AI Config key that this variation belongs to."},
			KEY:           schema.StringAttribute{Required: true, Description: "The variation's unique key."},
			NAME:          schema.StringAttribute{Computed: true, Description: "The variation's human-readable name."},
			MODEL:         schema.StringAttribute{Computed: true, Description: "A JSON string representing the inline model configuration."},
			MODEL_CONFIG_KEY: schema.StringAttribute{
				Computed:    true,
				Description: "The key of a model config resource used for this variation.",
			},
			DESCRIPTION:  schema.StringAttribute{Computed: true, Description: "The variation's description. Used in agent mode."},
			INSTRUCTIONS: schema.StringAttribute{Computed: true, Description: "The variation's instructions. Used in agent mode."},
			TOOL_KEYS: schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "A set of AI tool keys to associate with this variation. **Note:** The API does not currently return tool associations on read, so Terraform cannot detect drift for this field. Changes made outside of Terraform is not reflected in state.",
			},
			STATE:         schema.StringAttribute{Computed: true, Description: "The state of the variation. Must be `archived` or `published`."},
			VARIATION_ID:  schema.StringAttribute{Computed: true, Description: "The internal ID of the variation."},
			VERSION:       schema.Int64Attribute{Computed: true, Description: "The version number of the variation."},
			CREATION_DATE: schema.Int64Attribute{Computed: true, Description: "The creation timestamp of the variation."},
			MESSAGES: schema.ListNestedAttribute{
				Computed:    true,
				Description: "A list of messages for completion mode.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						ROLE:    schema.StringAttribute{Computed: true, Description: "Role of the message."},
						CONTENT: schema.StringAttribute{Computed: true, Description: "Content of the message."},
					},
				},
			},
		},
	}
}

func (d *AIConfigVariationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *AIConfigVariationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data AIConfigVariationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	configKey := data.AIConfigKey.ValueString()
	variationKey := data.Key.ValueString()

	var variationsResp *ldapi.AIConfigVariationsResponse
	var err error
	err = d.client.withConcurrency(d.client.ctx, func() error {
		variationsResp, _, err = d.client.ld.AgentControlApi.GetAIConfigVariation(d.client.ctx, projectKey, configKey, variationKey).Execute()
		return err
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to get AI config variation with key %q in config %q project %q: %s", variationKey, configKey, projectKey, handleLdapiErr(err).Error()),
			"",
		)
		return
	}

	if variationsResp == nil || len(variationsResp.Items) == 0 {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to get AI config variation with key %q in config %q project %q: no versions found", variationKey, configKey, projectKey),
			"",
		)
		return
	}

	// Pick the highest-version item. AI Config variations are versioned;
	// see memory/ai-config-variations.md.
	variation := variationsResp.Items[0]
	for _, v := range variationsResp.Items[1:] {
		if v.Version > variation.Version {
			variation = v
		}
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", projectKey, configKey, variationKey))
	data.ProjectKey = types.StringValue(projectKey)
	data.AIConfigKey = types.StringValue(configKey)
	data.Key = types.StringValue(variation.Key)
	data.Name = types.StringValue(variation.Name)
	data.VariationID = types.StringValue(variation.Id)
	data.Version = types.Int64Value(int64(variation.Version))
	data.CreationDate = types.Int64Value(variation.CreatedAt)

	if variation.Description != nil {
		data.Description = types.StringValue(*variation.Description)
	} else {
		data.Description = types.StringValue("")
	}
	if variation.Instructions != nil {
		data.Instructions = types.StringValue(*variation.Instructions)
	} else {
		data.Instructions = types.StringValue("")
	}
	if variation.ModelConfigKey != nil {
		data.ModelConfigKey = types.StringValue(*variation.ModelConfigKey)
	} else {
		data.ModelConfigKey = types.StringValue("")
	}
	if variation.State != nil {
		data.State = types.StringValue(*variation.State)
	} else {
		data.State = types.StringValue("")
	}

	cleanedModel := stripEmptyMapValues(variation.Model)
	if len(cleanedModel) > 0 && !isEmptyModelMap(cleanedModel) {
		modelJSON, err := mapToJsonString(cleanedModel)
		if err != nil {
			resp.Diagnostics.AddError("Failed to serialise model", err.Error())
			return
		}
		data.Model = types.StringValue(modelJSON)
	} else {
		data.Model = types.StringValue("")
	}

	messagesType := types.ObjectType{AttrTypes: aiConfigVariationMessageAttrTypes}
	elements := make([]attr.Value, 0, len(variation.Messages))
	for _, m := range variation.Messages {
		obj, d := types.ObjectValue(aiConfigVariationMessageAttrTypes, map[string]attr.Value{
			ROLE:    types.StringValue(m.Role),
			CONTENT: types.StringValue(m.Content),
		})
		resp.Diagnostics.Append(d...)
		elements = append(elements, obj)
	}
	msgs, diags := types.ListValue(messagesType, elements)
	resp.Diagnostics.Append(diags...)
	data.Messages = msgs

	toolKeys := make([]string, 0, len(variation.Tools))
	for _, t := range variation.Tools {
		toolKeys = append(toolKeys, t.Key)
	}
	toolSet, diags := setFromStringSlice(ctx, toolKeys)
	resp.Diagnostics.Append(diags...)
	data.ToolKeys = toolSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
