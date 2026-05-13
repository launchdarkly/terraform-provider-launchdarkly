package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ViewDataSource{}

type ViewDataSource struct {
	client *Client
}

type ViewDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	ProjectKey        types.String `tfsdk:"project_key"`
	Key               types.String `tfsdk:"key"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	MaintainerID      types.String `tfsdk:"maintainer_id"`
	MaintainerTeamKey types.String `tfsdk:"maintainer_team_key"`
	Tags              types.Set    `tfsdk:"tags"`
	Archived          types.Bool   `tfsdk:"archived"`
	LinkedFlags       types.List   `tfsdk:"linked_flags"`
	LinkedSegments    types.List   `tfsdk:"linked_segments"`
}

var viewLinkedSegmentAttrTypes = map[string]attr.Type{
	SEGMENT_ENVIRONMENT_ID: types.StringType,
	SEGMENT_KEY:            types.StringType,
}

func NewViewDataSource() datasource.DataSource {
	return &ViewDataSource{}
}

func (d *ViewDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_view"
}

func (d *ViewDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly view data source.\n\nThis data source allows you to retrieve view information from your LaunchDarkly project.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, Description: "View ID."},
			PROJECT_KEY:   schema.StringAttribute{Required: true, Description: "The project key."},
			KEY:           schema.StringAttribute{Required: true, Description: "The view's unique key."},
			NAME:          schema.StringAttribute{Computed: true, Description: "The view's name."},
			DESCRIPTION:   schema.StringAttribute{Computed: true, Description: "The view's description."},
			MAINTAINER_ID: schema.StringAttribute{Computed: true, Description: "Member ID of the maintainer."},
			MAINTAINER_TEAM_KEY: schema.StringAttribute{
				Computed:    true,
				Description: "Team key of the maintainer team.",
			},
			TAGS:     schema.SetAttribute{Computed: true, ElementType: types.StringType, Description: "Tags."},
			ARCHIVED: schema.BoolAttribute{Computed: true, Description: "Whether the view is archived."},
			LINKED_FLAGS: schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Feature flag keys linked to this view.",
			},
		},
		Blocks: map[string]schema.Block{
			LINKED_SEGMENTS: schema.ListNestedBlock{
				Description: "Segments linked to this view.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						SEGMENT_ENVIRONMENT_ID: schema.StringAttribute{Computed: true},
						SEGMENT_KEY:            schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *ViewDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *ViewDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data ViewDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	betaClient, err := newBetaClient(d.client.apiKey, d.client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		resp.Diagnostics.AddError("Failed to construct beta client", err.Error())
		return
	}

	projectKey := data.ProjectKey.ValueString()
	viewKey := data.Key.ValueString()

	view, _, err := getView(betaClient, projectKey, viewKey)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to get view with key %q in project %q: %s", viewKey, projectKey, handleLdapiErr(err).Error()),
			"",
		)
		return
	}

	data.ID = types.StringValue(view.Id)
	data.ProjectKey = types.StringValue(view.ProjectKey)
	data.Key = types.StringValue(view.Key)
	data.Name = types.StringValue(view.Name)
	if view.Description != nil {
		data.Description = types.StringValue(*view.Description)
	} else {
		data.Description = types.StringValue("")
	}
	if view.Archived != nil {
		data.Archived = types.BoolValue(*view.Archived)
	} else {
		data.Archived = types.BoolValue(false)
	}

	data.MaintainerID = types.StringValue("")
	data.MaintainerTeamKey = types.StringValue("")
	if view.Maintainer != nil {
		if view.Maintainer.Kind == "member" && view.Maintainer.MaintainerMember != nil {
			data.MaintainerID = types.StringValue(view.Maintainer.MaintainerMember.Id)
		} else if view.Maintainer.Kind == "team" && view.Maintainer.MaintainerTeam != nil {
			data.MaintainerTeamKey = types.StringValue(view.Maintainer.MaintainerTeam.Key)
		}
	}

	tagsSet, diags := setFromStringSlice(ctx, view.Tags)
	resp.Diagnostics.Append(diags...)
	data.Tags = tagsSet

	// linked_flags + linked_segments are best-effort (the SDKv2 version
	// logs a WARN and continues on failure). Surface as empty rather
	// than failing the read.
	flagKeys := []string{}
	if linkedFlags, err := getLinkedResources(betaClient, projectKey, viewKey, FLAGS); err == nil {
		flagKeys = make([]string, len(linkedFlags))
		for i, f := range linkedFlags {
			flagKeys[i] = f.ResourceKey
		}
	}
	flagsList, diags := listFromStringSlice(ctx, flagKeys)
	resp.Diagnostics.Append(diags...)
	data.LinkedFlags = flagsList

	segmentObjectType := types.ObjectType{AttrTypes: viewLinkedSegmentAttrTypes}
	segmentElements := []attr.Value{}
	if linkedSegments, err := getLinkedResources(betaClient, projectKey, viewKey, SEGMENTS); err == nil {
		segmentElements = make([]attr.Value, 0, len(linkedSegments))
		for _, s := range linkedSegments {
			obj, d := types.ObjectValue(viewLinkedSegmentAttrTypes, map[string]attr.Value{
				SEGMENT_ENVIRONMENT_ID: types.StringValue(s.EnvironmentId),
				SEGMENT_KEY:            types.StringValue(s.ResourceKey),
			})
			resp.Diagnostics.Append(d...)
			segmentElements = append(segmentElements, obj)
		}
	}
	segmentList, diags := types.ListValue(segmentObjectType, segmentElements)
	resp.Diagnostics.Append(diags...)
	data.LinkedSegments = segmentList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
