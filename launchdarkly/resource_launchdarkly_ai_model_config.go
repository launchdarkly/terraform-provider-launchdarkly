package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	ldapi "github.com/launchdarkly/api-client-go/v17"
)

func resourceAIModelConfig() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAIModelConfigCreate,
		ReadContext:   resourceAIModelConfigRead,
		UpdateContext: resourceAIModelConfigUpdate,
		DeleteContext: resourceAIModelConfigDelete,
		Schema:        baseAIModelConfigSchema(false),
		Importer: &schema.ResourceImporter{
			State: resourceAIModelConfigImport,
		},
		CustomizeDiff: validateAIModelConfigImmutable,

		Description: `Provides a LaunchDarkly AI model config resource.

This resource allows you to create and manage AI model configs within your LaunchDarkly organization.

~> **Note:** AI model configs are immutable. Any changes to the configuration will result in an error. To modify an AI model config, you must delete and recreate the resource.`,
	}
}

func validateAIModelConfigImmutable(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	// Only validate on updates, not on initial creation
	if d.Id() == "" {
		return nil
	}

	immutableFields := []string{
		NAME,
		MODEL_ID,
		MODEL_PROVIDER,
		ICON,
		PARAMS,
		CUSTOM_PARAMS,
		TAGS,
		COST_PER_INPUT_TOKEN,
		COST_PER_OUTPUT_TOKEN,
	}

	for _, field := range immutableFields {
		if d.HasChange(field) {
			return fmt.Errorf("AI model config resources are immutable. The field %q cannot be changed after creation. To modify this value, delete and recreate the resource", field)
		}
	}

	return nil
}

func resourceAIModelConfigCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)

	var diags diag.Diagnostics

	projectKey := d.Get(PROJECT_KEY).(string)

	if exists, err := projectExists(projectKey, client); !exists {
		if err != nil {
			return diag.FromErr(err)
		}
		return diag.Errorf("cannot find project with key %q", projectKey)
	}

	key := d.Get(KEY).(string)
	name := d.Get(NAME).(string)
	modelId := d.Get(MODEL_ID).(string)
	provider := d.Get(MODEL_PROVIDER).(string)
	icon := d.Get(ICON).(string)
	tags := stringsFromResourceData(d, TAGS)

	modelConfig := ldapi.ModelConfigPost{
		Key:      key,
		Name:     name,
		Id:       modelId,
		Provider: &provider,
		Icon:     &icon,
		Tags:     tags,
	}

	if params, ok := d.GetOk(PARAMS); ok {
		modelConfig.Params = expandParams(params.(map[string]interface{}))
	}

	if customParams, ok := d.GetOk(CUSTOM_PARAMS); ok {
		modelConfig.CustomParams = expandParams(customParams.(map[string]interface{}))
	}

	if costPerInputToken, ok := d.GetOk(COST_PER_INPUT_TOKEN); ok {
		cost := float64(costPerInputToken.(float64))
		modelConfig.CostPerInputToken = &cost
	}

	if costPerOutputToken, ok := d.GetOk(COST_PER_OUTPUT_TOKEN); ok {
		cost := float64(costPerOutputToken.(float64))
		modelConfig.CostPerOutputToken = &cost
	}

	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, _, err = client.ld.AIConfigsBetaApi.PostModelConfig(client.ctx, projectKey).ModelConfigPost(modelConfig).Execute()
		return err
	})

	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Error creating AI model config resource: %q", key),
			Detail:   fmt.Sprintf("Details: \n %q", handleLdapiErr(err)),
		})
		return diags
	}

	d.SetId(projectKey + "/" + key)

	return resourceAIModelConfigRead(ctx, d, metaRaw)
}

func resourceAIModelConfigRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return aiModelConfigRead(ctx, d, metaRaw, false)
}

// resourceAIModelConfigUpdate always returns an error because AI model configs are immutable.
// This function exists to satisfy Terraform SDK's internal validation requirements.
// The CustomizeDiff should catch any changes during the plan phase before this is ever called.
func resourceAIModelConfigUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return diag.Errorf("AI model config resources are immutable and cannot be updated. To modify an AI model config, delete and recreate the resource")
}

func resourceAIModelConfigDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	var diags diag.Diagnostics

	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)

	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, err = client.ld.AIConfigsBetaApi.DeleteModelConfig(client.ctx, projectKey, key).Execute()
		return err
	})
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Error deleting AI model config resource %q from project %q", key, projectKey),
			Detail:   fmt.Sprintf("Details: \n %q", handleLdapiErr(err)),
		})
		return diags
	}

	return resourceAIModelConfigRead(ctx, d, metaRaw)
}

func resourceAIModelConfigImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()

	projectKey, modelConfigKey, err := aiModelConfigIdToKeys(id)
	if err != nil {
		return nil, err
	}
	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(KEY, modelConfigKey)

	return []*schema.ResourceData{d}, nil
}
