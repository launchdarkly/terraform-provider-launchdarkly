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

var _ datasource.DataSource = &AIToolDataSource{}

type AIToolDataSource struct {
	client *Client
}

type AIToolDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	ProjectKey        types.String `tfsdk:"project_key"`
	Key               types.String `tfsdk:"key"`
	Description       types.String `tfsdk:"description"`
	SchemaJSON        types.String `tfsdk:"schema_json"`
	CustomParameters  types.String `tfsdk:"custom_parameters"`
	MaintainerID      types.String `tfsdk:"maintainer_id"`
	MaintainerTeamKey types.String `tfsdk:"maintainer_team_key"`
	Version           types.Int64  `tfsdk:"version"`
	CreationDate      types.Int64  `tfsdk:"creation_date"`
}

func NewAIToolDataSource() datasource.DataSource {
	return &AIToolDataSource{}
}

func (d *AIToolDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_tool"
}

func (d *AIToolDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly AI tool data source.\n\nThis data source allows you to retrieve AI tool information from your LaunchDarkly project.",
		Attributes: map[string]schema.Attribute{
			"id":              schema.StringAttribute{Computed: true, Description: "The ID in the format `project_key/key`."},
			PROJECT_KEY:       schema.StringAttribute{Required: true, Description: "The project key."},
			KEY:               schema.StringAttribute{Required: true, Description: "The AI tool's unique key."},
			DESCRIPTION:       schema.StringAttribute{Computed: true, Description: "The AI tool's description."},
			SCHEMA_JSON:       schema.StringAttribute{Computed: true, Description: "A JSON string representing the JSON Schema for the tool's parameters."},
			CUSTOM_PARAMETERS: schema.StringAttribute{Computed: true, Description: "A JSON string representing custom application-level metadata."},
			MAINTAINER_ID:     schema.StringAttribute{Computed: true, Description: "The member ID of the maintainer."},
			MAINTAINER_TEAM_KEY: schema.StringAttribute{
				Computed:    true,
				Description: "The team key of the maintainer team.",
			},
			VERSION:       schema.Int64Attribute{Computed: true, Description: "The version of the AI tool."},
			CREATION_DATE: schema.Int64Attribute{Computed: true, Description: "Creation timestamp."},
		},
	}
}

func (d *AIToolDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *AIToolDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data AIToolDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	key := data.Key.ValueString()

	var tool *ldapi.AITool
	var res *http.Response
	var err error
	err = d.client.withConcurrency(d.client.ctx, func() error {
		tool, res, err = d.client.ld.AIConfigsApi.GetAITool(d.client.ctx, projectKey, key).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			resp.Diagnostics.AddError("AI tool not found", fmt.Sprintf("AI tool %q in project %q not found.", key, projectKey))
			return
		}
		addLdapiError(&resp.Diagnostics, "Failed to get AI tool", err)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", projectKey, tool.GetKey()))
	data.Key = types.StringValue(tool.GetKey())

	if tool.Description != nil {
		data.Description = types.StringValue(*tool.Description)
	} else {
		data.Description = types.StringValue("")
	}

	schemaJSON, err := mapToJsonString(tool.GetSchema())
	if err != nil {
		resp.Diagnostics.AddError("Failed to serialise schema_json", err.Error())
		return
	}
	data.SchemaJSON = types.StringValue(schemaJSON)

	customParamsJSON, err := mapToJsonString(tool.GetCustomParameters())
	if err != nil {
		resp.Diagnostics.AddError("Failed to serialise custom_parameters", err.Error())
		return
	}
	data.CustomParameters = types.StringValue(customParamsJSON)

	data.Version = types.Int64Value(int64(tool.GetVersion()))
	data.CreationDate = types.Int64Value(tool.GetCreatedAt())

	data.MaintainerID = types.StringValue("")
	data.MaintainerTeamKey = types.StringValue("")
	maintainer := tool.GetMaintainer()
	if maintainer.MaintainerMember != nil {
		data.MaintainerID = types.StringValue(maintainer.MaintainerMember.GetId())
	}
	if maintainer.AiConfigsMaintainerTeam != nil {
		data.MaintainerTeamKey = types.StringValue(maintainer.AiConfigsMaintainerTeam.GetKey())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
