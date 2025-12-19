package launchdarkly

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	ldapi "github.com/launchdarkly/api-client-go/v17"
)

func baseAIModelConfigSchema(isDataSource bool) map[string]*schema.Schema {
	schemaMap := map[string]*schema.Schema{
		PROJECT_KEY: {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         !isDataSource,
			Description:      addForceNewDescription("The AI model config's project key. A change in this field will force the destruction of the existing resource and the creation of a new one.", !isDataSource),
			ValidateDiagFunc: validateKey(),
		},
		KEY: {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         !isDataSource,
			ValidateDiagFunc: validateKey(),
			Description:      addForceNewDescription("The unique key that references the AI model config. A change in this field will force the destruction of the existing resource and the creation of a new one.", !isDataSource),
		},
		NAME: {
			Type:        schema.TypeString,
			Required:    !isDataSource,
			Computed:    isDataSource,
			Description: "The human-friendly name for the AI model config.",
		},
		MODEL_ID: {
			Type:        schema.TypeString,
			Required:    !isDataSource,
			Computed:    isDataSource,
			Description: "Identifier for the model, for use with third party providers.",
		},
		MODEL_PROVIDER: {
			Type:        schema.TypeString,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "Provider for the model.",
		},
		ICON: {
			Type:        schema.TypeString,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "Icon for the model.",
		},
		PARAMS: {
			Type:        schema.TypeMap,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "Model parameters as a map of key-value pairs.",
		},
		CUSTOM_PARAMS: {
			Type:        schema.TypeMap,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "Custom model parameters as a map of key-value pairs.",
		},
		TAGS: tagsSchema(tagsSchemaOptions{isDataSource: isDataSource}),
		COST_PER_INPUT_TOKEN: {
			Type:        schema.TypeFloat,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "Cost per input token in USD.",
		},
		COST_PER_OUTPUT_TOKEN: {
			Type:        schema.TypeFloat,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			Description: "Cost per output token in USD.",
		},
		VERSION: {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "Version of the AI model config.",
		},
		GLOBAL: {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "Whether the model is global.",
		},
	}

	if isDataSource {
		return removeInvalidFieldsForDataSource(schemaMap)
	}

	return schemaMap
}

func aiModelConfigRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}, isDataSource bool) diag.Diagnostics {
	client := metaRaw.(*Client)

	var diags diag.Diagnostics

	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)

	var modelConfig *ldapi.ModelConfig
	var res *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		modelConfig, res, err = client.ld.AIConfigsBetaApi.GetModelConfig(client.ctx, projectKey, key).Execute()
		return err
	})

	if isStatusNotFound(res) && !isDataSource {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "AI model config not found",
			Detail:   fmt.Sprintf("[WARN] AI model config %q in project %q not found, removing from state", key, projectKey),
		})
		d.SetId("")
		return diags
	}
	if err != nil {
		return diag.FromErr(err)
	}

	_ = d.Set(KEY, modelConfig.Key)
	_ = d.Set(NAME, modelConfig.Name)
	_ = d.Set(MODEL_ID, modelConfig.Id)
	_ = d.Set(MODEL_PROVIDER, modelConfig.Provider)
	_ = d.Set(ICON, modelConfig.Icon)
	_ = d.Set(TAGS, modelConfig.Tags)
	_ = d.Set(VERSION, modelConfig.Version)
	_ = d.Set(GLOBAL, modelConfig.Global)

	if modelConfig.CostPerInputToken != nil {
		_ = d.Set(COST_PER_INPUT_TOKEN, *modelConfig.CostPerInputToken)
	}
	if modelConfig.CostPerOutputToken != nil {
		_ = d.Set(COST_PER_OUTPUT_TOKEN, *modelConfig.CostPerOutputToken)
	}

	if modelConfig.Params != nil {
		_ = d.Set(PARAMS, flattenParams(modelConfig.Params))
	} else {
		_ = d.Set(PARAMS, map[string]string{})
	}
	if modelConfig.CustomParams != nil {
		_ = d.Set(CUSTOM_PARAMS, flattenParams(modelConfig.CustomParams))
	} else {
		_ = d.Set(CUSTOM_PARAMS, map[string]string{})
	}

	d.SetId(projectKey + "/" + key)

	return diags
}

func aiModelConfigIdToKeys(id string) (projectKey string, modelConfigKey string, err error) {
	if strings.Count(id, "/") != 1 {
		return "", "", fmt.Errorf("found unexpected AI model config id format: %q expected format: 'project_key/model_config_key'", id)
	}
	parts := strings.SplitN(id, "/", 2)
	projectKey, modelConfigKey = parts[0], parts[1]
	return projectKey, modelConfigKey, nil
}

func flattenParams(params map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range params {
		if v != nil {
			result[k] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

func expandParams(params map[string]interface{}) map[string]interface{} {
	if params == nil {
		return nil
	}
	return params
}
