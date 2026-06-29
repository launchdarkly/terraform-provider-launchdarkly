package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var _ datasource.DataSource = &ExperimentationSettingsDataSource{}

type ExperimentationSettingsDataSource struct {
	client *Client
}

type ExperimentationSettingsDataSourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ProjectKey         types.String `tfsdk:"project_key"`
	RandomizationUnits types.List   `tfsdk:"randomization_units"`
}

func NewExperimentationSettingsDataSource() datasource.DataSource {
	return &ExperimentationSettingsDataSource{}
}

func (d *ExperimentationSettingsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_experimentation_settings"
}

func (d *ExperimentationSettingsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly experimentation settings data source.\n\nThis data source allows you to retrieve the randomization units configured for experiments in a project.",
		Attributes: map[string]schema.Attribute{
			ID: schema.StringAttribute{
				Computed:    true,
				Description: "The ID of this resource. Equal to the project key.",
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The project key.",
			},
			RANDOMIZATION_UNITS: schema.ListNestedAttribute{
				Computed:    true,
				Description: "The randomization units allowed for experiments in this project.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						RANDOMIZATION_UNIT: schema.StringAttribute{
							Computed:    true,
							Description: "The unit of randomization.",
						},
						DEFAULT: schema.BoolAttribute{
							Computed:    true,
							Description: "Whether new experiment iterations default to using this randomization unit.",
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
	err := d.client.withConcurrency(d.client.ctx, func() error {
		var e error
		settings, _, e = d.client.ld.ExperimentsApi.GetExperimentationSettings(d.client.ctx, projectKey).Execute()
		return e
	})
	if err != nil {
		resp.Diagnostics.AddError(handleLdapiErr(err).Error(), "")
		return
	}

	data.ID = types.StringValue(projectKey)
	data.ProjectKey = types.StringValue(projectKey)

	objType := types.ObjectType{AttrTypes: randomizationUnitAttrTypes}
	elems := make([]attr.Value, 0, len(settings.RandomizationUnits))
	for _, u := range settings.RandomizationUnits {
		if u.Hidden != nil && *u.Hidden {
			continue
		}
		obj, dg := types.ObjectValue(randomizationUnitAttrTypes, map[string]attr.Value{
			RANDOMIZATION_UNIT: stringValueFromPointer(u.RandomizationUnit),
			DEFAULT:            types.BoolValue(u.Default != nil && *u.Default),
		})
		resp.Diagnostics.Append(dg...)
		elems = append(elems, obj)
	}
	list, dg := types.ListValue(objType, elems)
	resp.Diagnostics.Append(dg...)
	data.RandomizationUnits = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
