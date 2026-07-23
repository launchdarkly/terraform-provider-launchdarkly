package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ProjectDataSource{}

type ProjectDataSource struct {
	client *Client
}

type ProjectDataSourceModel struct {
	ID                                   types.String `tfsdk:"id"`
	Key                                  types.String `tfsdk:"key"`
	Name                                 types.String `tfsdk:"name"`
	DefaultClientSideAvailability        types.Object `tfsdk:"default_client_side_availability"`
	Tags                                 types.Set    `tfsdk:"tags"`
	RequireViewAssociationForNewFlags    types.Bool   `tfsdk:"require_view_association_for_new_flags"`
	RequireViewAssociationForNewSegments types.Bool   `tfsdk:"require_view_association_for_new_segments"`
}

var projectClientSideAvailabilityAttrTypes = map[string]attr.Type{
	USING_ENVIRONMENT_ID: types.BoolType,
	USING_MOBILE_KEY:     types.BoolType,
}

func NewProjectDataSource() datasource.DataSource {
	return &ProjectDataSource{}
}

func (d *ProjectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *ProjectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly project data source.\n\nThis data source allows you to retrieve project information from your LaunchDarkly organization.\n\n-> **Note:** LaunchDarkly data sources do not provide access to the project's environments. If you wish to import environment configurations as data sources you must use the [`launchdarkly_environment` data source](/docs/providers/launchdarkly/d/environment.html).",
		Attributes: map[string]schema.Attribute{
			"id":                                   schema.StringAttribute{Computed: true, Description: "The project's ID."},
			KEY:                                    schema.StringAttribute{Required: true, Description: "The project's unique key."},
			NAME:                                   schema.StringAttribute{Computed: true, Description: "The project's name."},
			TAGS:                                   schema.SetAttribute{Computed: true, ElementType: types.StringType, Description: "Tags."},
			REQUIRE_VIEW_ASSOCIATION_FOR_NEW_FLAGS: schema.BoolAttribute{Computed: true, Description: "Whether new flags created in this project must be associated with at least one view."},
			REQUIRE_VIEW_ASSOCIATION_FOR_NEW_SEGMENTS: schema.BoolAttribute{Computed: true, Description: "Whether new segments created in this project must be associated with at least one view."},
			DEFAULT_CLIENT_SIDE_AVAILABILITY: schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Which client-side SDKs can use new flags by default.",
				Attributes: map[string]schema.Attribute{
					USING_ENVIRONMENT_ID: schema.BoolAttribute{Computed: true},
					USING_MOBILE_KEY:     schema.BoolAttribute{Computed: true},
				},
			},
		},
	}
}

func (d *ProjectDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *ProjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data ProjectDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.Key.ValueString()
	project, _, err := getFullProject(d.client, projectKey)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to get project with key %q: %s", projectKey, handleLdapiErr(err).Error()),
			"",
		)
		return
	}

	data.ID = types.StringValue(project.Id)
	data.Key = types.StringValue(project.Key)
	data.Name = types.StringValue(project.Name)

	tagsSet, diags := setFromStringSlice(ctx, project.Tags)
	resp.Diagnostics.Append(diags...)
	data.Tags = tagsSet

	var csaObj types.Object
	if project.DefaultClientSideAvailability != nil {
		defaultCSA := *project.DefaultClientSideAvailability
		usingEnvID := false
		if defaultCSA.UsingEnvironmentId != nil {
			usingEnvID = *defaultCSA.UsingEnvironmentId
		}
		usingMobile := false
		if defaultCSA.UsingMobileKey != nil {
			usingMobile = *defaultCSA.UsingMobileKey
		}
		obj, d := types.ObjectValue(projectClientSideAvailabilityAttrTypes, map[string]attr.Value{
			USING_ENVIRONMENT_ID: types.BoolValue(usingEnvID),
			USING_MOBILE_KEY:     types.BoolValue(usingMobile),
		})
		resp.Diagnostics.Append(d...)
		csaObj = obj
	} else {
		csaObj = types.ObjectNull(projectClientSideAvailabilityAttrTypes)
	}
	data.DefaultClientSideAvailability = csaObj

	viewSettings, viewSettingsErr := getProjectViewSettings(ctx, d.client, projectKey)
	if viewSettingsErr != nil {
		// Older LD accounts may not return view settings; surface as
		// false (with a warning) rather than failing the data source.
		resp.Diagnostics.AddWarning(
			"Failed to read project view settings",
			fmt.Sprintf("Could not read view settings for project %q: %s. Defaulting require_view_association_for_new_* to false.", projectKey, viewSettingsErr.Error()),
		)
		data.RequireViewAssociationForNewFlags = types.BoolValue(false)
		data.RequireViewAssociationForNewSegments = types.BoolValue(false)
	} else {
		data.RequireViewAssociationForNewFlags = types.BoolValue(viewSettings.RequireViewAssociationForNewFlags)
		data.RequireViewAssociationForNewSegments = types.BoolValue(viewSettings.RequireViewAssociationForNewSegments)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
