package launchdarkly

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v17"
)

func resourceAIConfig() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAIConfigCreate,
		ReadContext:   resourceAIConfigRead,
		UpdateContext: resourceAIConfigUpdate,
		DeleteContext: resourceAIConfigDelete,
		Exists:        resourceAIConfigExists,

		Importer: &schema.ResourceImporter{
			State: resourceAIConfigImport,
		},

		Schema: map[string]*schema.Schema{
			PROJECT_KEY: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The project key.",
			},
			KEY: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The unique key of the AI Config.",
			},
			NAME: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The human-readable name of the AI Config.",
			},
			DESCRIPTION: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the AI Config.",
			},
			TAGS: {
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Tags associated with the AI Config.",
			},
			MAINTAINER_ID: {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{MAINTAINER_TEAM_KEY},
				Description:   "The ID of the member who maintains this AI Config.",
			},
			MAINTAINER_TEAM_KEY: {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{MAINTAINER_ID},
				Description:   "The key of the team that maintains this AI Config.",
			},
			VERSION: {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The version of the AI Config.",
			},
		},

		Description: `Provides a LaunchDarkly AI Config resource.

This resource allows you to create and manage AI Configs within your LaunchDarkly project.

-> **Note:** AI Configs are currently in beta.`,
	}
}

func resourceAIConfigCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)

	if exists, err := projectExists(projectKey, client); !exists {
		if err != nil {
			return diag.FromErr(err)
		}
		return diag.Errorf("cannot find project with key %q", projectKey)
	}

	key := d.Get(KEY).(string)
	name := d.Get(NAME).(string)
	description := d.Get(DESCRIPTION).(string)
	tags := stringsFromResourceData(d, TAGS)

	aiConfigPost := ldapi.AIConfigPost{
		Key:  key,
		Name: name,
		Tags: tags,
	}

	if description != "" {
		aiConfigPost.Description = &description
	}

	maintainerId, maintainerIdOk := d.GetOk(MAINTAINER_ID)
	maintainerTeamKey, maintainerTeamKeyOk := d.GetOk(MAINTAINER_TEAM_KEY)

	if maintainerIdOk {
		maintainerIdStr := maintainerId.(string)
		aiConfigPost.MaintainerId = &maintainerIdStr
	}
	if maintainerTeamKeyOk {
		maintainerTeamKeyStr := maintainerTeamKey.(string)
		aiConfigPost.MaintainerTeamKey = &maintainerTeamKeyStr
	}

	var res *http.Response
	var err error
	err = client.withConcurrency(ctx, func() error {
		_, res, err = client.ldBeta.AIConfigsBetaApi.PostAIConfig(client.ctx, projectKey).LDAPIVersion("beta").AIConfigPost(aiConfigPost).Execute()
		return err
	})

	if err != nil && res != nil && res.StatusCode >= 200 && res.StatusCode < 300 && isMaintainerOneOfDecodeErr(err) {
		d.SetId(projectKey + "/" + key)
		return resourceAIConfigRead(ctx, d, metaRaw)
	}

	if err != nil {
		return diag.Errorf("failed to create AI config %q in project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	d.SetId(projectKey + "/" + key)
	return resourceAIConfigRead(ctx, d, metaRaw)
}

func resourceAIConfigRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
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
		d.SetId("")
		return diags
	}

	if err != nil && res != nil && res.StatusCode >= 200 && res.StatusCode < 300 && isMaintainerOneOfDecodeErr(err) {
		name, description, tags, version, teamKey, memberID, parseErr := parseAIConfigFromError(err)
		if parseErr != nil {
			return diag.Errorf("failed to parse AI config %q from error response: %s", key, parseErr)
		}

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

func resourceAIConfigUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)
	name := d.Get(NAME).(string)
	description := d.Get(DESCRIPTION).(string)
	tags := stringsFromResourceData(d, TAGS)

	aiConfigPatch := ldapi.AIConfigPatch{
		Name:        &name,
		Description: &description,
		Tags:        tags,
	}

	if d.HasChange(MAINTAINER_ID) || d.HasChange(MAINTAINER_TEAM_KEY) {
		maintainerId, maintainerIdOk := d.GetOk(MAINTAINER_ID)
		maintainerTeamKey, maintainerTeamKeyOk := d.GetOk(MAINTAINER_TEAM_KEY)

		if maintainerIdOk {
			maintainerIdStr := maintainerId.(string)
			aiConfigPatch.MaintainerId = &maintainerIdStr
		}
		if maintainerTeamKeyOk {
			maintainerTeamKeyStr := maintainerTeamKey.(string)
			aiConfigPatch.MaintainerTeamKey = &maintainerTeamKeyStr
		}
	}

	var res *http.Response
	var err error
	err = client.withConcurrency(ctx, func() error {
		_, res, err = client.ldBeta.AIConfigsBetaApi.PatchAIConfig(client.ctx, projectKey, key).LDAPIVersion("beta").AIConfigPatch(aiConfigPatch).Execute()
		return err
	})

	if err != nil && res != nil && res.StatusCode >= 200 && res.StatusCode < 300 && isMaintainerOneOfDecodeErr(err) {
		return resourceAIConfigRead(ctx, d, metaRaw)
	}

	if err != nil {
		return diag.Errorf("failed to update AI config %q in project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	return resourceAIConfigRead(ctx, d, metaRaw)
}

func resourceAIConfigDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)

	var err error
	err = client.withConcurrency(ctx, func() error {
		_, err = client.ldBeta.AIConfigsBetaApi.DeleteAIConfig(client.ctx, projectKey, key).LDAPIVersion("beta").Execute()
		return err
	})

	if err != nil {
		return diag.Errorf("failed to delete AI config %q from project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	return diags
}

func resourceAIConfigExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)

	var res *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, res, err = client.ldBeta.AIConfigsBetaApi.GetAIConfig(client.ctx, projectKey, key).LDAPIVersion("beta").Execute()
		return err
	})

	if isStatusNotFound(res) {
		return false, nil
	}

	if err != nil && res != nil && res.StatusCode >= 200 && res.StatusCode < 300 && isMaintainerOneOfDecodeErr(err) {
		return true, nil
	}

	if err != nil {
		return false, fmt.Errorf("failed to check if AI config %q exists in project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	return true, nil
}

func resourceAIConfigImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()

	projectKey, aiConfigKey, err := aiConfigIdToKeys(id)
	if err != nil {
		return nil, err
	}
	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(KEY, aiConfigKey)

	return []*schema.ResourceData{d}, nil
}

func aiConfigIdToKeys(id string) (projectKey string, aiConfigKey string, err error) {
	if strings.Count(id, "/") != 1 {
		return "", "", fmt.Errorf("found unexpected AI config id format: %q expected format: 'project_key/ai_config_key'", id)
	}
	parts := strings.SplitN(id, "/", 2)
	projectKey, aiConfigKey = parts[0], parts[1]
	return projectKey, aiConfigKey, nil
}

func isMaintainerOneOfDecodeErr(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "oneOf(AIConfigMaintainer)") &&
		strings.Contains(errStr, "Data failed to match schemas")
}

func parseAIConfigFromError(err error) (name, description string, tags []string, version int32, teamKey, memberID *string, parseErr error) {
	if err == nil {
		return "", "", nil, 0, nil, nil, fmt.Errorf("no error provided")
	}

	errStr := err.Error()
	jsonStart := strings.Index(errStr, ": {")
	if jsonStart == -1 {
		return "", "", nil, 0, nil, nil, fmt.Errorf("could not find JSON payload in error")
	}

	jsonStr := errStr[jsonStart+2:] // Skip ": " to get the JSON

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return "", "", nil, 0, nil, nil, fmt.Errorf("failed to parse JSON from error: %w", err)
	}

	if n, ok := data["name"].(string); ok {
		name = n
	}
	if d, ok := data["description"].(string); ok {
		description = d
	}
	if v, ok := data["version"].(float64); ok {
		version = int32(v)
	}
	if t, ok := data["tags"].([]interface{}); ok {
		tags = make([]string, 0, len(t))
		for _, tag := range t {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}

	if maintainer, ok := data["_maintainer"].(map[string]interface{}); ok {
		if kind, ok := maintainer["kind"].(string); ok {
			if kind == "team" {
				if key, ok := maintainer["key"].(string); ok {
					teamKey = &key
				}
			} else if kind == "member" {
				if id, ok := maintainer["_id"].(string); ok {
					memberID = &id
				}
			}
		}
	}

	return name, description, tags, version, teamKey, memberID, nil
}
