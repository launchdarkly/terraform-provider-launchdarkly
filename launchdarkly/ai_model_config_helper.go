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
			ForceNew:         true,
			Description:      addForceNewDescription("The AI model config's project key. A change in this field will force the destruction of the existing resource and the creation of a new one.", !isDataSource),
			ValidateDiagFunc: validateKey(),
		},
		KEY: {
			Type:             schema.TypeString,
			Required:         true,
			ForceNew:         true,
			ValidateDiagFunc: validateKey(),
			Description:      addForceNewDescription("The unique key that references the AI model config. A change in this field will force the destruction of the existing resource and the creation of a new one.", !isDataSource),
		},
		NAME: {
			Type:        schema.TypeString,
			Required:    !isDataSource,
			Computed:    isDataSource,
			ForceNew:    !isDataSource,
			Description: addForceNewDescription("The human-friendly name for the AI model config.", !isDataSource),
		},
		ID: {
			Type:        schema.TypeString,
			Required:    !isDataSource,
			Computed:    isDataSource,
			ForceNew:    !isDataSource,
			Description: addForceNewDescription("Identifier for the model, for use with third party providers.", !isDataSource),
		},
		PROVIDER: {
			Type:        schema.TypeString,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			ForceNew:    !isDataSource,
			Description: addForceNewDescription("Provider for the model.", !isDataSource),
		},
		ICON: {
			Type:        schema.TypeString,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			ForceNew:    !isDataSource,
			Description: addForceNewDescription("Icon for the model.", !isDataSource),
		},
		PARAMS: {
			Type:        schema.TypeString,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			ForceNew:    !isDataSource,
			Description: addForceNewDescription("Model parameters as a JSON string.", !isDataSource),
			DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
				return jsonEqual(old, new)
			},
		},
		CUSTOM_PARAMS: {
			Type:        schema.TypeString,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			ForceNew:    !isDataSource,
			Description: addForceNewDescription("Custom model parameters as a JSON string.", !isDataSource),
			DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
				return jsonEqual(old, new)
			},
		},
		TAGS: tagsSchema(tagsSchemaOptions{isDataSource: isDataSource}),
		COST_PER_INPUT_TOKEN: {
			Type:        schema.TypeFloat,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			ForceNew:    !isDataSource,
			Description: addForceNewDescription("Cost per input token in USD.", !isDataSource),
		},
		COST_PER_OUTPUT_TOKEN: {
			Type:        schema.TypeFloat,
			Optional:    !isDataSource,
			Computed:    isDataSource,
			ForceNew:    !isDataSource,
			Description: addForceNewDescription("Cost per output token in USD.", !isDataSource),
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
	_ = d.Set(ID, modelConfig.Id)
	_ = d.Set(PROVIDER, modelConfig.Provider)
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
		paramsJSON, err := jsonMarshal(modelConfig.Params)
		if err != nil {
			return diag.FromErr(err)
		}
		_ = d.Set(PARAMS, paramsJSON)
	}
	if modelConfig.CustomParams != nil {
		customParamsJSON, err := jsonMarshal(modelConfig.CustomParams)
		if err != nil {
			return diag.FromErr(err)
		}
		_ = d.Set(CUSTOM_PARAMS, customParamsJSON)
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
