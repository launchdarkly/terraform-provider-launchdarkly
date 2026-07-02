package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var _ datasource.DataSource = &TeamDataSource{}

type TeamDataSource struct {
	client *Client
}

type TeamDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Key            types.String `tfsdk:"key"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Maintainers    types.Set    `tfsdk:"maintainers"`
	ProjectKeys    types.Set    `tfsdk:"project_keys"`
	CustomRoleKeys types.Set    `tfsdk:"custom_role_keys"`
	RoleAttributes types.Map    `tfsdk:"role_attributes"`
}

var teamMaintainerAttrTypes = map[string]attr.Type{
	EMAIL:      types.StringType,
	ID:         types.StringType,
	FIRST_NAME: types.StringType,
	LAST_NAME:  types.StringType,
	ROLE:       types.StringType,
}

func NewTeamDataSource() datasource.DataSource {
	return &TeamDataSource{}
}

func (d *TeamDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (d *TeamDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly team data source.\n\nThis data source allows you to retrieve team information from your LaunchDarkly organization.\n\n-> **Note:** Teams are available to customers on an Enterprise LaunchDarkly plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).",
		Attributes: map[string]schema.Attribute{
			"id":        schema.StringAttribute{Computed: true, Description: "The team key."},
			KEY:         schema.StringAttribute{Required: true, Description: "The team key."},
			DESCRIPTION: schema.StringAttribute{Computed: true, Description: "The team description."},
			NAME:        schema.StringAttribute{Computed: true, Description: "Human-readable name for the team."},
			PROJECT_KEYS: schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "The list of keys of the projects that the team has any write access to.",
			},
			CUSTOM_ROLE_KEYS: schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "The list of keys of the custom roles assigned to the team.",
			},
			MAINTAINERS: schema.SetNestedAttribute{
				Computed:    true,
				Description: "Team maintainers.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						EMAIL:      schema.StringAttribute{Computed: true, Description: "Email of the maintainer."},
						ID:         schema.StringAttribute{Computed: true, Description: "Member ID."},
						FIRST_NAME: schema.StringAttribute{Computed: true, Description: "First name."},
						LAST_NAME:  schema.StringAttribute{Computed: true, Description: "Last name."},
						ROLE:       schema.StringAttribute{Computed: true, Description: "Role."},
					},
				},
			},
			ROLE_ATTRIBUTES: frameworkRoleAttributesDataSourceAttribute(),
		},
	}
}

func (d *TeamDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *TeamDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data TeamDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamKey := data.Key.ValueString()

	var team *ldapi.Team
	var err error
	err = d.client.withConcurrency(d.client.ctx, func() error {
		team, _, err = d.client.ld.TeamsApi.GetTeam(d.client.ctx, teamKey).Expand("roles,projects,maintainers,roleAttributes").Execute()
		return err
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Failed to get team %q", teamKey), err)
		return
	}

	data.ID = types.StringValue(teamKey)
	if team.Key != nil {
		data.Key = types.StringValue(*team.Key)
	}
	if team.Name != nil {
		data.Name = types.StringValue(*team.Name)
	}
	if team.Description != nil {
		data.Description = types.StringValue(*team.Description)
	} else {
		data.Description = types.StringValue("")
	}

	projects := []string{}
	if team.Projects != nil {
		projects = make([]string, len(team.Projects.Items))
		for i, v := range team.Projects.Items {
			projects[i] = v.Key
		}
	}
	projectSet, diags := setFromStringSlice(ctx, projects)
	resp.Diagnostics.Append(diags...)
	data.ProjectKeys = projectSet

	// Paginated; see getAllTeamCustomRoleKeys in team_helper.go.
	customRoleKeys, err := getAllTeamCustomRoleKeys(d.client, teamKey)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list team custom roles", err.Error())
		return
	}
	customRoleSet, diags := setFromStringSlice(ctx, customRoleKeys)
	resp.Diagnostics.Append(diags...)
	data.CustomRoleKeys = customRoleSet

	maintainersList, err := getAllTeamMaintainers(d.client, teamKey)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list team maintainers", err.Error())
		return
	}
	maintainerType := types.ObjectType{AttrTypes: teamMaintainerAttrTypes}
	maintainerElements := make([]attr.Value, 0, len(maintainersList))
	for _, m := range maintainersList {
		obj, d := types.ObjectValue(teamMaintainerAttrTypes, map[string]attr.Value{
			EMAIL:      types.StringValue(m.Email),
			ID:         types.StringValue(m.Id),
			FIRST_NAME: stringValueFromPointer(m.FirstName),
			LAST_NAME:  stringValueFromPointer(m.LastName),
			ROLE:       types.StringValue(m.Role),
		})
		resp.Diagnostics.Append(d...)
		maintainerElements = append(maintainerElements, obj)
	}
	maintainerSet, diags := types.SetValue(maintainerType, maintainerElements)
	resp.Diagnostics.Append(diags...)
	data.Maintainers = maintainerSet

	roleAttrs, diags := frameworkRoleAttributesValue(ctx, team.RoleAttributes)
	resp.Diagnostics.Append(diags...)
	data.RoleAttributes = roleAttrs

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
