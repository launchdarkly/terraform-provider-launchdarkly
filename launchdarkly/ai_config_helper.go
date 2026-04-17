package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

func baseAIConfigSchema(isDataSource bool) map[string]*schema.Schema {
	schemaMap := map[string]*schema.Schema{
		PROJECT_KEY: {
			Type:             schema.TypeString,
			Required:         true,
			Description:      addForceNewDescription("The project key.", !isDataSource),
			ForceNew:         !isDataSource,
			ValidateDiagFunc: validateKey(),
		},
		KEY: {
			Type:             schema.TypeString,
			Required:         true,
			Description:      addForceNewDescription("The AI Config's unique key.", !isDataSource),
			ForceNew:         !isDataSource,
			ValidateDiagFunc: validateKey(),
		},
		NAME: {
			Type:        schema.TypeString,
			Required:    !isDataSource,
			Computed:    isDataSource,
			Description: "The AI Config's human-readable name.",
		},
		DESCRIPTION: {
			Type:        schema.TypeString,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "The AI Config's description.",
		},
		MODE: {
			Type:             schema.TypeString,
			Optional:         !isDataSource,
			Computed:         isDataSource,
			ForceNew:         !isDataSource,
			Default:          emptyValueIfDataSource("completion", isDataSource),
			Description:      addForceNewDescription("The AI Config's mode. Must be `completion`, `agent`, or `judge`. Defaults to `completion`.", !isDataSource),
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"completion", "agent", "judge"}, false)),
		},
		TAGS: tagsSchema(tagsSchemaOptions{isDataSource: isDataSource}),
		MAINTAINER_ID: {
			Type:             schema.TypeString,
			Optional:         !isDataSource,
			Computed:         true,
			Description:      "The member ID of the maintainer for this AI Config. Conflicts with `maintainer_team_key`.",
			ConflictsWith:    []string{MAINTAINER_TEAM_KEY},
			ValidateDiagFunc: validateID(),
		},
		MAINTAINER_TEAM_KEY: {
			Type:          schema.TypeString,
			Optional:      !isDataSource,
			Computed:      true,
			Description:   "The team key of the maintainer team for this AI Config. Conflicts with `maintainer_id`.",
			ConflictsWith: []string{MAINTAINER_ID},
		},
		EVALUATION_METRIC_KEY: {
			Type:        schema.TypeString,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "The key of the evaluation metric associated with this AI Config.",
		},
		IS_INVERTED: {
			Type:        schema.TypeBool,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "Whether the evaluation metric is inverted.",
		},
		VERSION: {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The version of the AI Config.",
		},
		CREATION_DATE: {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "A timestamp of when the AI Config was created.",
		},
		VARIATIONS: {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "A list of variation summaries for this AI Config.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					KEY: {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "The variation's key.",
					},
					NAME: {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "The variation's name.",
					},
					VARIATION_ID: {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "The variation's ID.",
					},
				},
			},
		},
	}

	if isDataSource {
		schemaMap = removeInvalidFieldsForDataSource(schemaMap)
	}

	return schemaMap
}

func aiConfigRead(ctx context.Context, d *schema.ResourceData, meta interface{}, isDataSource bool) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*Client)

	projectKey := d.Get(PROJECT_KEY).(string)
	configKey := d.Get(KEY).(string)

	var aiConfig *ldapi.AIConfig
	var res *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		aiConfig, res, err = client.ld.AIConfigsApi.GetAIConfig(client.ctx, projectKey, configKey).Execute()
		return err
	})

	if isStatusNotFound(res) && !isDataSource {
		log.Printf("[DEBUG] failed to find AI config with key %q in project %q, removing from state if present", configKey, projectKey)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("[WARN] failed to find AI config with key %q in project %q, removing from state if present", configKey, projectKey),
		})
		d.SetId("")
		return diags
	}
	if err != nil {
		return diag.Errorf("failed to get AI config with key %q in project %q: %s", configKey, projectKey, handleLdapiErr(err))
	}

	if isDataSource {
		d.SetId(fmt.Sprintf("%s/%s", projectKey, aiConfig.Key))
	}

	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(KEY, aiConfig.Key)
	_ = d.Set(NAME, aiConfig.Name)
	_ = d.Set(DESCRIPTION, aiConfig.Description)
	_ = d.Set(VERSION, aiConfig.Version)
	_ = d.Set(CREATION_DATE, aiConfig.CreatedAt)

	// Always set mode, defaulting to "completion" when the API field is nil.
	// The schema already defaults to "completion" on create, so a nil response
	// here likely means the API omitted the field. Log a warning so this doesn't
	// silently mask a non-completion config returning nil.
	mode := "completion"
	if aiConfig.Mode != nil {
		mode = *aiConfig.Mode
	} else {
		log.Printf("[DEBUG] AI config %q in project %q returned nil mode, defaulting to %q", configKey, projectKey, mode)
	}
	_ = d.Set(MODE, mode)

	// Always set evaluation_metric_key, defaulting to empty string when nil.
	evaluationMetricKey := ""
	if aiConfig.EvaluationMetricKey != nil {
		evaluationMetricKey = *aiConfig.EvaluationMetricKey
	}
	_ = d.Set(EVALUATION_METRIC_KEY, evaluationMetricKey)

	isInverted := false
	if aiConfig.IsInverted != nil {
		isInverted = *aiConfig.IsInverted
	}
	_ = d.Set(IS_INVERTED, isInverted)

	// Clear both maintainer fields first, then set the one returned by the API.
	// This prevents stale values from persisting when the maintainer kind changes.
	_ = d.Set(MAINTAINER_ID, "")
	_ = d.Set(MAINTAINER_TEAM_KEY, "")
	maintainer := aiConfig.GetMaintainer()
	if maintainer.MaintainerMember != nil {
		_ = d.Set(MAINTAINER_ID, maintainer.MaintainerMember.GetId())
	}
	if maintainer.AiConfigsMaintainerTeam != nil {
		_ = d.Set(MAINTAINER_TEAM_KEY, maintainer.AiConfigsMaintainerTeam.GetKey())
	}

	err = d.Set(TAGS, aiConfig.Tags)
	if err != nil {
		return diag.Errorf("could not set tags on AI config with key %q: %v", configKey, err)
	}

	// Set computed variations summary
	variations := make([]map[string]interface{}, len(aiConfig.Variations))
	for i, v := range aiConfig.Variations {
		variations[i] = map[string]interface{}{
			KEY:          v.Key,
			NAME:         v.Name,
			VARIATION_ID: v.Id,
		}
	}
	_ = d.Set(VARIATIONS, variations)

	return diags
}

func aiConfigIdToKeys(id string) (projectKey, configKey string, err error) {
	parts := splitID(id, 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("import ID must be in the format project_key/config_key, got: %q", id)
	}
	return parts[0], parts[1], nil
}
