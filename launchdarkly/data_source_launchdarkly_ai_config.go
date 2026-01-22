package launchdarkly

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v17"
)

func dataSourceAIConfig() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceAIConfigRead,
		Schema: map[string]*schema.Schema{
			PROJECT_KEY: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The project key.",
			},
			KEY: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The unique key of the AI Config.",
			},
			NAME: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The human-readable name of the AI Config.",
			},
			DESCRIPTION: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The description of the AI Config.",
			},
			TAGS: {
				Type:        schema.TypeSet,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Tags associated with the AI Config.",
			},
			MAINTAINER_ID: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the member who maintains this AI Config.",
			},
			MAINTAINER_TEAM_KEY: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The key of the team that maintains this AI Config.",
			},
			VERSION: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The version of the AI Config.",
			},
		},
		Description: `Provides a LaunchDarkly AI Config data source.

This data source allows you to retrieve AI Config information from your LaunchDarkly project.

-> **Note:** AI Configs are currently in beta.`,
	}
}

func dataSourceAIConfigRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)

	var aiConfig *ldapi.AIConfig
	var res *http.Response
	var err error
	err = client.withConcurrency(ctx, func() error {
		aiConfig, res, err = client.ldBeta.AIConfigsBetaApi.GetAIConfig(client.ctx, projectKey, key).LDAPIVersion("beta").Execute()
		return err
	})

	if isStatusNotFound(res) {
		return diag.Errorf("failed to get AI config %q in project %q: not found", key, projectKey)
	}

	if err != nil && res != nil && res.StatusCode >= 200 && res.StatusCode < 300 && isMaintainerOneOfDecodeErr(err) {
		name, description, tags, version, teamKey, memberID, parseErr := parseAIConfigFromResponse(res)
		if parseErr != nil {
			return diag.Errorf("failed to parse AI config %q from response: %s", key, parseErr)
		}

		d.SetId(projectKey + "/" + key)
		_ = d.Set(NAME, name)
		_ = d.Set(DESCRIPTION, description)
		_ = d.Set(TAGS, tags)
		_ = d.Set(VERSION, version)

		if teamKey != nil {
			_ = d.Set(MAINTAINER_TEAM_KEY, *teamKey)
		}
		if memberID != nil {
			_ = d.Set(MAINTAINER_ID, *memberID)
		}

		return diags
	}
	if err != nil {
		return diag.Errorf("failed to get AI config %q in project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	d.SetId(projectKey + "/" + key)
	_ = d.Set(NAME, aiConfig.Name)
	_ = d.Set(DESCRIPTION, aiConfig.Description)
	_ = d.Set(TAGS, aiConfig.Tags)
	_ = d.Set(VERSION, aiConfig.Version)

	if aiConfig.Maintainer != nil {
		if aiConfig.Maintainer.MaintainerMember != nil {
			_ = d.Set(MAINTAINER_ID, aiConfig.Maintainer.MaintainerMember.Id)
		}
		if aiConfig.Maintainer.AiConfigsMaintainerTeam != nil {
			_ = d.Set(MAINTAINER_TEAM_KEY, aiConfig.Maintainer.AiConfigsMaintainerTeam.Key)
		}
	}

	return diags
}
