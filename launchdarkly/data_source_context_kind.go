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

var (
	_ datasource.DataSource              = &ContextKindDataSource{}
	_ datasource.DataSourceWithConfigure = &ContextKindDataSource{}
)

type ContextKindDataSource struct {
	client *Client
}

type ContextKindDataSourceModel struct {
	ProjectKey      types.String `tfsdk:"project_key"`
	Key             types.String `tfsdk:"key"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	HideInTargeting types.Bool   `tfsdk:"hide_in_targeting"`
	Archived        types.Bool   `tfsdk:"archived"`
	Version         types.Int64  `tfsdk:"version"`
	CreationDate    types.Int64  `tfsdk:"creation_date"`
	LastModified    types.Int64  `tfsdk:"last_modified"`
	CreatedFrom     types.String `tfsdk:"created_from"`
	ID              types.String `tfsdk:"id"`
}

func NewContextKindDataSource() datasource.DataSource {
	return &ContextKindDataSource{}
}

func (d *ContextKindDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_context_kind"
}

func (d *ContextKindDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a LaunchDarkly context kind by project + key. Useful for inspecting the built-in `user` kind " +
			"or any kind managed outside Terraform.",
		Attributes: map[string]schema.Attribute{
			"project_key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The LaunchDarkly project key that scopes the context kind.",
			},
			"key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The unique identifier of the context kind within the project.",
			},
			"name":              schema.StringAttribute{Computed: true, MarkdownDescription: "The human-readable name of the context kind."},
			"description":       schema.StringAttribute{Computed: true, MarkdownDescription: "The description of the context kind."},
			"hide_in_targeting": schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the context kind is hidden from targeting UIs."},
			"archived":          schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the context kind is archived. Archived kinds are unavailable for targeting."},
			"version":           schema.Int64Attribute{Computed: true, MarkdownDescription: "The LaunchDarkly-assigned version."},
			"creation_date":     schema.Int64Attribute{Computed: true, MarkdownDescription: "Unix epoch (milliseconds) at which the context kind was created."},
			"last_modified":     schema.Int64Attribute{Computed: true, MarkdownDescription: "Unix epoch (milliseconds) of the last server-side modification."},
			"created_from":      schema.StringAttribute{Computed: true, MarkdownDescription: "How the context kind was first created."},
			"id":                schema.StringAttribute{Computed: true, MarkdownDescription: "The composite identifier `<project_key>/<key>`."},
		},
	}
}

func (d *ContextKindDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected DataSource Configure Type", fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData))
		return
	}
	d.client = client
}

func (d *ContextKindDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ContextKindDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	key := data.Key.ValueString()

	var items []ldapi.ContextKindRep
	var res *http.Response
	err := d.client.withConcurrency(d.client.ctx, func() error {
		rep, httpRes, listErr := d.client.ld.ContextsApi.GetContextKindsByProjectKey(d.client.ctx, projectKey).Execute()
		res = httpRes
		if listErr != nil {
			return listErr
		}
		if rep != nil {
			items = rep.Items
		}
		return nil
	})
	if err != nil {
		if isStatusNotFound(res) {
			resp.Diagnostics.AddError(
				"Project not found",
				fmt.Sprintf("LaunchDarkly project %q does not exist.", projectKey),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to read context kinds",
			fmt.Sprintf("Received an error listing context kinds for project %q: %s", projectKey, handleLdapiErr(err)),
		)
		return
	}

	kind, ok := findContextKindByKey(items, key)
	if !ok {
		resp.Diagnostics.AddError(
			"Context kind not found",
			fmt.Sprintf("No context kind with key %q exists in project %q.", key, projectKey),
		)
		return
	}

	data.ProjectKey = types.StringValue(projectKey)
	data.Key = types.StringValue(kind.Key)
	data.Name = types.StringValue(kind.Name)
	data.Description = types.StringValue(kind.Description)
	if kind.HideInTargeting != nil {
		data.HideInTargeting = types.BoolValue(*kind.HideInTargeting)
	} else {
		data.HideInTargeting = types.BoolValue(false)
	}
	if kind.Archived != nil {
		data.Archived = types.BoolValue(*kind.Archived)
	} else {
		data.Archived = types.BoolValue(false)
	}
	data.Version = types.Int64Value(int64(kind.Version))
	data.CreationDate = types.Int64Value(kind.CreationDate)
	data.LastModified = types.Int64Value(kind.LastModified)
	data.CreatedFrom = types.StringValue(kind.CreatedFrom)
	data.ID = types.StringValue(projectKey + "/" + key)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
