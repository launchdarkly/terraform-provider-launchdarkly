package launchdarkly

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v23"
)

var _ datasource.DataSource = &FlagTemplatesDataSource{}

type FlagTemplatesDataSource struct {
	client *Client
}

type FlagTemplatesDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	ProjectKey      types.String `tfsdk:"project_key"`
	Tags            types.Set    `tfsdk:"tags"`
	Temporary       types.Bool   `tfsdk:"temporary"`
	BooleanDefaults types.Object `tfsdk:"boolean_defaults"`
}

var flagTemplatesBooleanDefaultsAttrTypes = map[string]attr.Type{
	TRUE_DISPLAY_NAME:  types.StringType,
	FALSE_DISPLAY_NAME: types.StringType,
	TRUE_DESCRIPTION:   types.StringType,
	FALSE_DESCRIPTION:  types.StringType,
	ON_VARIATION:       types.Int64Type,
	OFF_VARIATION:      types.Int64Type,
}

func NewFlagTemplatesDataSource() datasource.DataSource {
	return &FlagTemplatesDataSource{}
}

func (d *FlagTemplatesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flag_templates"
}

func (d *FlagTemplatesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly flag templates data source.\n\nThis data source allows you to retrieve the \"Custom\" flag template settings for a LaunchDarkly project.",
		Attributes: map[string]schema.Attribute{
			"id":        schema.StringAttribute{Computed: true, Description: "Project key (the ID)."},
			PROJECT_KEY: schema.StringAttribute{Required: true, Description: "The project key."},
			TAGS:        schema.SetAttribute{Computed: true, ElementType: types.StringType, Description: "Tags applied by default."},
			TEMPORARY:   schema.BoolAttribute{Computed: true, Description: "Whether new flags should be temporary by default."},
			BOOLEAN_DEFAULTS: schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Default boolean flag variation settings.",
				Attributes: map[string]schema.Attribute{
					TRUE_DISPLAY_NAME:  schema.StringAttribute{Computed: true, Description: "Display name for the true variation."},
					FALSE_DISPLAY_NAME: schema.StringAttribute{Computed: true, Description: "Display name for the false variation."},
					TRUE_DESCRIPTION:   schema.StringAttribute{Computed: true, Description: "Description for the true variation."},
					FALSE_DESCRIPTION:  schema.StringAttribute{Computed: true, Description: "Description for the false variation."},
					ON_VARIATION:       schema.Int64Attribute{Computed: true, Description: "Variation index served when targeting is on (0 or 1)."},
					OFF_VARIATION:      schema.Int64Attribute{Computed: true, Description: "Variation index served when targeting is off (0 or 1)."},
				},
			},
		},
	}
}

func (d *FlagTemplatesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *FlagTemplatesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data FlagTemplatesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()

	var flagDefaults *ldapi.FlagDefaultsRep
	var res *http.Response
	var err error
	err = d.client.withConcurrency(d.client.ctx, func() error {
		flagDefaults, res, err = d.client.ld.ProjectsApi.GetFlagDefaultsByProject(d.client.ctx, projectKey).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			resp.Diagnostics.AddError("Flag templates not found", fmt.Sprintf("Flag templates for project %q not found.", projectKey))
			return
		}
		addLdapiError(&resp.Diagnostics, "Failed to get flag templates", err)
		return
	}

	data.ID = types.StringValue(projectKey)
	data.ProjectKey = types.StringValue(projectKey)

	tags := flagDefaults.Tags
	if tags == nil {
		tags = []string{}
	}
	tagsSet, diags := setFromStringSlice(ctx, tags)
	resp.Diagnostics.Append(diags...)
	data.Tags = tagsSet

	if flagDefaults.Temporary != nil {
		data.Temporary = types.BoolValue(*flagDefaults.Temporary)
	} else {
		data.Temporary = types.BoolValue(false)
	}

	if flagDefaults.BooleanDefaults != nil {
		bd := flagDefaults.BooleanDefaults
		obj, d := types.ObjectValue(flagTemplatesBooleanDefaultsAttrTypes, map[string]attr.Value{
			TRUE_DISPLAY_NAME:  types.StringValue(bd.GetTrueDisplayName()),
			FALSE_DISPLAY_NAME: types.StringValue(bd.GetFalseDisplayName()),
			TRUE_DESCRIPTION:   types.StringValue(bd.GetTrueDescription()),
			FALSE_DESCRIPTION:  types.StringValue(bd.GetFalseDescription()),
			ON_VARIATION:       types.Int64Value(int64(bd.GetOnVariation())),
			OFF_VARIATION:      types.Int64Value(int64(bd.GetOffVariation())),
		})
		resp.Diagnostics.Append(d...)
		data.BooleanDefaults = obj
	} else {
		data.BooleanDefaults = types.ObjectNull(flagTemplatesBooleanDefaultsAttrTypes)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
