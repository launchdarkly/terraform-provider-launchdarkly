package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var _ datasource.DataSource = &EnvironmentDataSource{}

type EnvironmentDataSource struct {
	client *Client
}

type EnvironmentDataSourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ProjectKey         types.String `tfsdk:"project_key"`
	Key                types.String `tfsdk:"key"`
	Name               types.String `tfsdk:"name"`
	APIKey             types.String `tfsdk:"api_key"`
	MobileKey          types.String `tfsdk:"mobile_key"`
	ClientSideID       types.String `tfsdk:"client_side_id"`
	Color              types.String `tfsdk:"color"`
	DefaultTTL         types.Int64  `tfsdk:"default_ttl"`
	SecureMode         types.Bool   `tfsdk:"secure_mode"`
	DefaultTrackEvents types.Bool   `tfsdk:"default_track_events"`
	RequireComments    types.Bool   `tfsdk:"require_comments"`
	ConfirmChanges     types.Bool   `tfsdk:"confirm_changes"`
	Critical           types.Bool   `tfsdk:"critical"`
	Tags               types.Set    `tfsdk:"tags"`
	ApprovalSettings   types.List   `tfsdk:"approval_settings"`
}

func NewEnvironmentDataSource() datasource.DataSource {
	return &EnvironmentDataSource{}
}

func (d *EnvironmentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

func (d *EnvironmentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly environment data source.\n\nThis data source allows you to retrieve environment information from your LaunchDarkly organization.",
		Attributes: map[string]schema.Attribute{
			"id":        schema.StringAttribute{Computed: true, Description: "The ID in the format `project_key/key`."},
			PROJECT_KEY: schema.StringAttribute{Required: true, Description: "The environment's project key."},
			KEY:         schema.StringAttribute{Required: true, Description: "The project-unique key for the environment."},
			NAME:        schema.StringAttribute{Computed: true, Description: "The name of the environment."},
			COLOR:       schema.StringAttribute{Computed: true, Description: "The color swatch as an RGB hex value with no leading `#`."},
			API_KEY:     schema.StringAttribute{Computed: true, Sensitive: true, Description: "The environment's SDK key."},
			MOBILE_KEY:  schema.StringAttribute{Computed: true, Sensitive: true, Description: "The environment's mobile key."},
			CLIENT_SIDE_ID: schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The environment's client-side ID.",
			},
			DEFAULT_TTL:          schema.Int64Attribute{Computed: true, Description: "The default TTL (0-60 minutes)."},
			SECURE_MODE:          schema.BoolAttribute{Computed: true, Description: "Whether secure mode is enabled."},
			DEFAULT_TRACK_EVENTS: schema.BoolAttribute{Computed: true, Description: "Whether data export is enabled for new flags."},
			REQUIRE_COMMENTS:     schema.BoolAttribute{Computed: true, Description: "Whether flag/segment changes require comments."},
			CONFIRM_CHANGES:      schema.BoolAttribute{Computed: true, Description: "Whether flag/segment changes require confirmation."},
			CRITICAL:             schema.BoolAttribute{Optional: true, Computed: true, Description: "Denotes whether the environment is critical."},
			TAGS: schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Tags.",
			},
			APPROVAL_SETTINGS: frameworkApprovalSettingsDataSourceAttribute(),
		},
	}
}

func (d *EnvironmentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *EnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data EnvironmentDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	key := data.Key.ValueString()

	var env *ldapi.Environment
	var err error
	err = d.client.withConcurrency(d.client.ctx, func() error {
		env, _, err = d.client.ld.EnvironmentsApi.GetEnvironment(d.client.ctx, projectKey, key).Execute()
		return err
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to get environment with key %q for project key: %q: %s", key, projectKey, handleLdapiErr(err).Error()),
			"",
		)
		return
	}

	data.ID = types.StringValue(projectKey + "/" + key)
	data.Key = types.StringValue(env.Key)
	data.Name = types.StringValue(env.Name)
	data.APIKey = types.StringValue(env.ApiKey)
	data.MobileKey = types.StringValue(env.MobileKey)
	data.ClientSideID = types.StringValue(env.Id)
	data.Color = types.StringValue(env.Color)
	data.DefaultTTL = types.Int64Value(int64(env.DefaultTtl))
	data.SecureMode = types.BoolValue(env.SecureMode)
	data.DefaultTrackEvents = types.BoolValue(env.DefaultTrackEvents)
	data.RequireComments = types.BoolValue(env.RequireComments)
	data.ConfirmChanges = types.BoolValue(env.ConfirmChanges)
	data.Critical = types.BoolValue(env.Critical)

	tagsSet, diags := setFromStringSlice(ctx, env.Tags)
	resp.Diagnostics.Append(diags...)
	data.Tags = tagsSet

	approvals, diags := frameworkApprovalSettingsDataSourceValue(ctx, env.ApprovalSettings)
	resp.Diagnostics.Append(diags...)
	data.ApprovalSettings = approvals

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
