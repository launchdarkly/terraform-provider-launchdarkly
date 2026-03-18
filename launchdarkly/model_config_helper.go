package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

func baseModelConfigSchema(isDataSource bool) map[string]*schema.Schema {
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
			Description:      addForceNewDescription("The model config's unique key.", !isDataSource),
			ForceNew:         !isDataSource,
			ValidateDiagFunc: validateKey(),
		},
		NAME: {
			Type:        schema.TypeString,
			Required:    !isDataSource,
			Computed:    isDataSource,
			Description: addForceNewDescription("The model config's human-readable name.", !isDataSource),
			ForceNew:    !isDataSource,
		},
		MODEL_ID: {
			Type:        schema.TypeString,
			Required:    !isDataSource,
			Computed:    isDataSource,
			Description: addForceNewDescription("The model identifier (e.g. `gpt-4`, `claude-3`).", !isDataSource),
			ForceNew:    !isDataSource,
		},
		ICON: {
			Type:        schema.TypeString,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: addForceNewDescription("The icon for the model config.", !isDataSource),
			ForceNew:    !isDataSource,
		},
		PROVIDER_NAME: {
			Type:        schema.TypeString,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: addForceNewDescription("The provider name for the model config (e.g. `openai`, `anthropic`).", !isDataSource),
			ForceNew:    !isDataSource,
		},
		GLOBAL: {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "Whether the model config is available globally.",
		},
		PARAMS: {
			Type:             schema.TypeString,
			Optional:         !isDataSource,
			Computed:         isDataSource,
			Description:      addForceNewDescription("A JSON string representing the model parameters (e.g. `{\"temperature\": 0.7, \"maxTokens\": 4096}`).", !isDataSource),
			ForceNew:         !isDataSource,
			ValidateFunc:     emptyValueIfDataSource(validateJsonStringFunc, isDataSource),
			DiffSuppressFunc: emptyValueIfDataSource(suppressEquivalentJsonDiffs, isDataSource),
		},
		CUSTOM_PARAMETERS: {
			Type:             schema.TypeString,
			Optional:         !isDataSource,
			Computed:         isDataSource,
			Description:      addForceNewDescription("A JSON string representing custom parameters for the model config.", !isDataSource),
			ForceNew:         !isDataSource,
			ValidateFunc:     emptyValueIfDataSource(validateJsonStringFunc, isDataSource),
			DiffSuppressFunc: emptyValueIfDataSource(suppressEquivalentJsonDiffs, isDataSource),
		},
		TAGS: tagsSchema(tagsSchemaOptions{isDataSource: isDataSource}),
		VERSION: {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The version of the model config.",
		},
		COST_PER_INPUT_TOKEN: {
			Type:        schema.TypeFloat,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: addForceNewDescription("The cost per input token for the model.", !isDataSource),
			ForceNew:    !isDataSource,
		},
		COST_PER_OUTPUT_TOKEN: {
			Type:        schema.TypeFloat,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: addForceNewDescription("The cost per output token for the model.", !isDataSource),
			ForceNew:    !isDataSource,
		},
	}

	if isDataSource {
		schemaMap = removeInvalidFieldsForDataSource(schemaMap)
	}

	return schemaMap
}

func modelConfigRead(ctx context.Context, d *schema.ResourceData, meta interface{}, isDataSource bool) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*Client)

	projectKey := d.Get(PROJECT_KEY).(string)
	modelConfigKey := d.Get(KEY).(string)

	var modelConfig *ldapi.ModelConfig
	var res *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		modelConfig, res, err = client.ld.AIConfigsApi.GetModelConfig(client.ctx, projectKey, modelConfigKey).Execute()
		return err
	})

	if isStatusNotFound(res) && !isDataSource {
		log.Printf("[WARN] failed to find model config with key %q in project %q, removing from state if present", modelConfigKey, projectKey)
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("[WARN] failed to find model config with key %q in project %q, removing from state if present", modelConfigKey, projectKey),
		})
		d.SetId("")
		return diags
	}
	if err != nil {
		return diag.Errorf("failed to get model config with key %q in project %q: %s", modelConfigKey, projectKey, handleLdapiErr(err))
	}

	if isDataSource {
		d.SetId(fmt.Sprintf("%s/%s", projectKey, modelConfig.Key))
	}

	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(KEY, modelConfig.Key)
	_ = d.Set(NAME, modelConfig.Name)
	_ = d.Set(MODEL_ID, modelConfig.Id)
	_ = d.Set(GLOBAL, modelConfig.Global)
	_ = d.Set(VERSION, modelConfig.Version)

	icon := ""
	if modelConfig.Icon != nil {
		icon = *modelConfig.Icon
	}
	_ = d.Set(ICON, icon)

	provider := ""
	if modelConfig.Provider != nil {
		provider = *modelConfig.Provider
	}
	_ = d.Set(PROVIDER_NAME, provider)

	paramsJSON, err := mapToJsonString(modelConfig.Params)
	if err != nil {
		return diag.Errorf("failed to serialize params for model config %q: %s", modelConfigKey, err)
	}
	_ = d.Set(PARAMS, paramsJSON)

	customParamsJSON, err := mapToJsonString(modelConfig.CustomParams)
	if err != nil {
		return diag.Errorf("failed to serialize custom_parameters for model config %q: %s", modelConfigKey, err)
	}
	_ = d.Set(CUSTOM_PARAMETERS, customParamsJSON)

	err = d.Set(TAGS, modelConfig.Tags)
	if err != nil {
		return diag.Errorf("could not set tags on model config with key %q: %v", modelConfigKey, err)
	}

	costPerInputToken := 0.0
	if modelConfig.CostPerInputToken != nil {
		costPerInputToken = *modelConfig.CostPerInputToken
	}
	_ = d.Set(COST_PER_INPUT_TOKEN, costPerInputToken)

	costPerOutputToken := 0.0
	if modelConfig.CostPerOutputToken != nil {
		costPerOutputToken = *modelConfig.CostPerOutputToken
	}
	_ = d.Set(COST_PER_OUTPUT_TOKEN, costPerOutputToken)

	return diags
}

func modelConfigIdToKeys(id string) (projectKey, modelConfigKey string, err error) {
	parts := splitID(id, 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("import ID must be in the format project_key/model_config_key, got: %q", id)
	}
	return parts[0], parts[1], nil
}
