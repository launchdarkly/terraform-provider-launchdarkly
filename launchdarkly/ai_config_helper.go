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

func baseAiConfigSchema(isDataSource bool) map[string]*schema.Schema {
	schemaMap := map[string]*schema.Schema{
		PROJECT_KEY: {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			Description:      addForceNewDescription("The AI Config's project key.", !isDataSource),
			ValidateDiagFunc: validateKey(),
		},
		KEY: {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			ValidateDiagFunc: validateKey(),
			Description:      addForceNewDescription("The unique key that references the AI Config.", !isDataSource),
		},
		NAME: {
			Type:        schema.TypeString,
			Required:    !isDataSource,
			Computed:    isDataSource,
			Description: "The human-friendly name for the AI Config.",
		},
		DESCRIPTION: {
			Type:        schema.TypeString,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "The description of the AI Config.",
		},
		TAGS: tagsSchema(tagsSchemaOptions{isDataSource: isDataSource}),
		MODE: {
			Type:             schema.TypeString,
			Optional:         !isDataSource,
			Computed:         true,
			ForceNew:         true,
			Description:      addForceNewDescription("The mode for the AI Config. Available choices are `agent`, `completion`, and `judge`. Defaults to `completion`.", !isDataSource),
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"agent", "completion", "judge"}, false)),
		},
		MAINTAINER_ID: {
			Type:             schema.TypeString,
			Optional:         !isDataSource,
			Computed:         true,
			Description:      "The LaunchDarkly member ID of the member who will maintain the AI Config. If not set, the API will automatically apply the member associated with your Terraform API key or the most recently-set maintainer.",
			ValidateDiagFunc: validateID(),
		},
		MAINTAINER_TEAM_KEY: {
			Type:        schema.TypeString,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "The key of the team that will maintain the AI Config.",
		},
		EVALUATION_METRIC_KEY: {
			Type:        schema.TypeString,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "The evaluation metric key for this AI Config.",
		},
		IS_INVERTED: {
			Type:        schema.TypeBool,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "Whether the evaluation metric is inverted, meaning a lower value is better if set as true.",
		},
		VERSION: {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The version of the AI Config.",
		},
	}

	if isDataSource {
		return removeInvalidFieldsForDataSource(schemaMap)
	}

	return schemaMap
}

func aiConfigRead(ctx context.Context, d *schema.ResourceData, meta interface{}, isDataSource bool) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*Client)

	projectKey := d.Get(PROJECT_KEY).(string)
	configKey := d.Get(KEY).(string)

	var aiConfig *ldapi.AIConfig
	var resp *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		aiConfig, resp, err = client.ld.AIConfigsApi.GetAIConfig(client.ctx, projectKey, configKey).Execute()
		return err
	})

	if isStatusNotFound(resp) {
		if isDataSource {
			return diag.Errorf("failed to find AI Config %q in project %q", configKey, projectKey)
		}
		log.Printf("[WARN] AI Config %q in project %q not found, removing from state", configKey, projectKey)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("[WARN] AI Config %q in project %q not found, removing from state", configKey, projectKey),
		})
		d.SetId("")
		return diags
	}
	if err != nil {
		return diag.Errorf("failed to get AI Config %q in project %q: %s", configKey, projectKey, handleLdapiErr(err))
	}

	_ = d.Set(KEY, aiConfig.Key)
	_ = d.Set(NAME, aiConfig.Name)
	_ = d.Set(DESCRIPTION, aiConfig.Description)

	if aiConfig.Tags != nil {
		_ = d.Set(TAGS, aiConfig.Tags)
	} else {
		_ = d.Set(TAGS, []string{})
	}

	if aiConfig.Mode != nil {
		_ = d.Set(MODE, *aiConfig.Mode)
	}
	_ = d.Set(VERSION, aiConfig.Version)

	if aiConfig.EvaluationMetricKey != nil {
		_ = d.Set(EVALUATION_METRIC_KEY, *aiConfig.EvaluationMetricKey)
	}

	if aiConfig.IsInverted != nil {
		_ = d.Set(IS_INVERTED, *aiConfig.IsInverted)
	}

	if aiConfig.HasMaintainer() {
		maintainer := aiConfig.GetMaintainer()
		if maintainer.MaintainerMember != nil {
			_ = d.Set(MAINTAINER_ID, maintainer.MaintainerMember.Id)
		}
		// For team maintainers, we don't have the key from the API response directly.
		// Keep the existing value from state if it's set.
	}

	d.SetId(projectKey + "/" + configKey)

	return diags
}

func aiConfigIdToKeys(id string) (projectKey string, configKey string, err error) {
	parts := splitID(id, 2)
	if parts == nil {
		return "", "", fmt.Errorf("found unexpected AI Config id format: %q expected format: 'project_key/config_key'", id)
	}
	return parts[0], parts[1], nil
}
