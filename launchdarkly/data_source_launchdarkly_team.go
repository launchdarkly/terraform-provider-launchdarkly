package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v17"
)

func teamSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		KEY: {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The team key.",
		},
		DESCRIPTION: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The team description.",
		},
		NAME: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Human-readable name for the team.",
		},
		MAINTAINERS: {
			Type:     schema.TypeSet,
			Computed: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					EMAIL: {
						Type:     schema.TypeString,
						Required: true,
					},
					ID: {
						Type:     schema.TypeString,
						Computed: true,
						Optional: true,
					},
					FIRST_NAME: {
						Type:     schema.TypeString,
						Computed: true,
					},
					LAST_NAME: {
						Type:     schema.TypeString,
						Computed: true,
					},
					ROLE: {
						Type:     schema.TypeString,
						Computed: true,
					},
				},
			},
			Description: "The list of team maintainers as [team member objects](/providers/launchdarkly/launchdarkly/latest/docs/data-sources/team_member).",
		},
		PROJECT_KEYS: {
			Type:        schema.TypeSet,
			Computed:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "The list of keys of the projects that the team has any write access to.",
		},
		CUSTOM_ROLE_KEYS: {
			Type:        schema.TypeSet,
			Computed:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "The list of the keys of the custom roles that you have assigned to the team.",
		},
		ROLE_ATTRIBUTES: roleAttributesSchema(true),
	}
}

func dataSourceTeam() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceTeamRead,
		Schema:      teamSchema(),

		Description: `Provides a LaunchDarkly team data source.

This data source allows you to retrieve team information from your LaunchDarkly organization.

-> **Note:** Teams are available to customers on an Enterprise LaunchDarkly plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).`,
	}
}

func dataSourceTeamRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*Client)
	teamKey := d.Get(KEY).(string)
	var team *ldapi.Team
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		team, _, err = client.ld.TeamsApi.GetTeam(client.ctx, teamKey).Expand("roles,projects,maintainers,roleAttributes").Execute()
		return err
	})

	if err != nil {
		return diag.Errorf("Error when calling `TeamsApi.GetTeam`: %v\n\n request: %v", err, team)
	}

	projects := make([]string, len(team.Projects.Items))
	for i, v := range team.Projects.Items {
		projects[i] = v.Key
	}

	// Fetch all custom role keys with pagination
	// The expand=roles parameter only returns the first page (default 25 items)
	// See: https://launchdarkly.atlassian.net/browse/REL-11737
	customRoleKeys, err := getAllTeamCustomRoleKeys(client, teamKey)
	if err != nil {
		return diag.Errorf("failed to get custom roles for team %q: %s", teamKey, err)
	}

	// Fetch all maintainers with pagination
	// The expand=maintainers parameter only returns the first page (default 25 items)
	// See: https://launchdarkly.atlassian.net/browse/REL-11737
	maintainersList, err := getAllTeamMaintainers(client, teamKey)
	if err != nil {
		return diag.Errorf("failed to get maintainers for team %q: %s", teamKey, err)
	}

	maintainers := make([]map[string]interface{}, 0, len(maintainersList))
	for _, m := range maintainersList {
		maintainer := make(map[string]interface{})
		maintainer[ID] = m.Id
		maintainer[EMAIL] = m.Email
		maintainer[FIRST_NAME] = m.FirstName
		maintainer[LAST_NAME] = m.LastName
		maintainer[ROLE] = m.Role
		maintainers = append(maintainers, maintainer)
	}

	d.SetId(teamKey)
	_ = d.Set(KEY, team.Key)
	_ = d.Set(NAME, team.Name)
	_ = d.Set(DESCRIPTION, team.Description)
	_ = d.Set(MAINTAINERS, maintainers)
	_ = d.Set(PROJECT_KEYS, projects)
	_ = d.Set(CUSTOM_ROLE_KEYS, customRoleKeys)
	err = d.Set(ROLE_ATTRIBUTES, roleAttributesToResourceData(team.RoleAttributes))
	if err != nil {
		return diag.Errorf("failed to set role attributes on team %q: %v", teamKey, err)
	}

	return diags
}
