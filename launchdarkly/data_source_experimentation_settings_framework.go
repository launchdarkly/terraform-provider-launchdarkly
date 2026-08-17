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

var _ datasource.DataSource = &ExperimentationSettingsDataSource{}

type ExperimentationSettingsDataSource struct {
	client *Client
}

type ExperimentationSettingsDataSourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ProjectKey         types.String `tfsdk:"project_key"`
	RandomizationUnits types.Map    `tfsdk:"randomization_units"`
}

func NewExperimentationSettingsDataSource() datasource.DataSource {
	return &ExperimentationSettingsDataSource{}
}

func (d *ExperimentationSettingsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_experimentation_settings"
}

func (d *ExperimentationSettingsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly experimentation settings data source.\n\nThis data source lets you retrieve the randomization units configured for experiments within a LaunchDarkly project.",
		Attributes: map[string]schema.Attribute{
			ID: schema.StringAttribute{
				Computed:    true,
				Description: "The unique resource ID, which is equal to the `project_key`.",
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The project key.",
			},
			RANDOMIZATION_UNITS: schema.MapNestedAttribute{
				Computed:    true,
				Description: "The randomization units available for experiments in this project, keyed by the randomization unit.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						DEFAULT: schema.BoolAttribute{
							Computed:    true,
							Description: "Whether this randomization unit is the default for experiments in the project.",
						},
						DISPLAY_NAME: schema.StringAttribute{
							Computed:    true,
							Description: "The display name for the randomization unit, shown in the LaunchDarkly user interface.",
						},
						HIDDEN: schema.BoolAttribute{
							Computed:    true,
							Description: "Whether this randomization unit is hidden in the LaunchDarkly user interface.",
						},
					},
				},
			},
		},
	}
}

func (d *ExperimentationSettingsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *ExperimentationSettingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data ExperimentationSettingsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()

	var settings *ldapi.RandomizationSettingsRep
	var err error
	err = d.client.withConcurrency(d.client.ctx, func() error {
		settings, _, err = d.client.ld.ExperimentsApi.GetExperimentationSettings(d.client.ctx, projectKey).Execute()
		return err
	})
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to get experimentation settings for project %q", projectKey), handleLdapiErr(err).Error())
		return
	}

	data.ID = types.StringValue(projectKey)

	elems := make(map[string]attr.Value, len(settings.RandomizationUnits))
	for _, u := range settings.RandomizationUnits {
		unitKey := ""
		if u.RandomizationUnit != nil {
			unitKey = *u.RandomizationUnit
		}
		if unitKey == "" {
			continue
		}
		def := false
		if u.Default != nil {
			def = *u.Default
		}
		hidden := false
		if u.Hidden != nil {
			hidden = *u.Hidden
		}
		displayName := ""
		if u.DisplayName != nil {
			displayName = *u.DisplayName
		}
		obj, diags := types.ObjectValue(experimentationSettingsUnitAttrTypes, map[string]attr.Value{
			DEFAULT:      types.BoolValue(def),
			DISPLAY_NAME: types.StringValue(displayName),
			HIDDEN:       types.BoolValue(hidden),
		})
		resp.Diagnostics.Append(diags...)
		elems[unitKey] = obj
	}

	unitMap, diags := types.MapValue(types.ObjectType{AttrTypes: experimentationSettingsUnitAttrTypes}, elems)
	resp.Diagnostics.Append(diags...)
	data.RandomizationUnits = unitMap

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
