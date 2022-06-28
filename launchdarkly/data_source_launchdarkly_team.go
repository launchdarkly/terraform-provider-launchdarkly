package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func teamSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		KEY: {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The team's unique key",
		},
		DESCRIPTION: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The team's description",
		},
		NAME: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The team's human-readable name",
		},
		MAINTAINERS: {
			Type:     schema.TypeList,
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
			Description: "A list of maintainers as 'member' objects",
		},
		PROJECT_KEYS: {
			Type:        schema.TypeList,
			Computed:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "A list of keys of projects that this team owns",
		},
		CUSTOM_ROLE_KEYS: {
			Type:        schema.TypeList,
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "A list of keys for custom roles the team has",
		},
	}
}

func dataSourceTeam() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceTeamRead,
		Schema:      teamSchema(),
	}
}

func dataSourceTeamRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*Client)
	teamKey := d.Get(KEY).(string)
	team, _, err := client.ld.TeamsApi.GetTeam(client.ctx, teamKey).Expand("roles,projects,maintainers").Execute()

	if err != nil {
		return diag.Errorf("Error when calling `TeamsApi.GetTeam`: %v\n\n request: %v", err, team)
	}

	projects := make([]string, len(team.Projects.Items))
	for i, v := range team.Projects.Items {
		projects[i] = *v.Key
	}

	customRoleKeys := make([]string, len(team.Roles.Items))
	for i, v := range team.Roles.Items {
		customRoleKeys[i] = *v.Key
	}

	maintainers := make([]map[string]interface{}, 0, len(team.Maintainers.Items))
	for _, m := range team.Maintainers.Items {
		maintainer := make(map[string]interface{})
		maintainer[ID] = m.Id
		maintainer[EMAIL] = m.Email
		maintainer[FIRST_NAME] = m.FirstName
		maintainer[LAST_NAME] = m.LastName
		maintainer[ROLE] = m.Role
		maintainers = append(maintainers, maintainer)
	}

	if err != nil {
		return diag.Errorf("failed to get team %q: %s", teamKey, handleLdapiErr(err))
	}

	d.SetId(teamKey)
	_ = d.Set(KEY, team.Key)
	_ = d.Set(NAME, team.Name)
	_ = d.Set(DESCRIPTION, team.Description)
	_ = d.Set(MAINTAINERS, maintainers)
	_ = d.Set(PROJECT_KEYS, projects)
	_ = d.Set(CUSTOM_ROLE_KEYS, customRoleKeys)

	return diags
}
