package launchdarkly

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

func resourceModelConfig() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceModelConfigCreate,
		ReadContext:   resourceModelConfigRead,
		DeleteContext: resourceModelConfigDelete,
		Exists:        resourceModelConfigExists,

		Importer: &schema.ResourceImporter{
			StateContext: resourceModelConfigImport,
		},

		Description: `Provides a LaunchDarkly model config resource.

This resource allows you to create and manage AI model configurations within your LaunchDarkly project. Since the API does not support updates, all mutable fields will force recreation of the resource when changed.`,

		Schema: baseModelConfigSchema(false),
	}
}

func resourceModelConfigCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)

	projectKey := d.Get(PROJECT_KEY).(string)
	modelConfigKey := d.Get(KEY).(string)
	name := d.Get(NAME).(string)
	modelID := d.Get(MODEL_ID).(string)

	post := *ldapi.NewModelConfigPost(name, modelConfigKey, modelID)

	if v, ok := d.GetOk(ICON); ok {
		icon := v.(string)
		post.Icon = &icon
	}

	if v, ok := d.GetOk(PROVIDER_NAME); ok {
		provider := v.(string)
		post.Provider = &provider
	}

	if v, ok := d.GetOk(PARAMS); ok {
		params, err := jsonStringToMap(v.(string))
		if err != nil {
			return diag.Errorf("failed to parse params JSON: %s", err)
		}
		if params != nil {
			post.Params = params
		}
	}

	if v, ok := d.GetOk(CUSTOM_PARAMETERS); ok {
		customParams, err := jsonStringToMap(v.(string))
		if err != nil {
			return diag.Errorf("failed to parse custom_parameters JSON: %s", err)
		}
		if customParams != nil {
			post.CustomParams = customParams
		}
	}

	tags := stringsFromResourceData(d, TAGS)
	post.Tags = tags

	if v, ok := d.GetOk(COST_PER_INPUT_TOKEN); ok {
		costPerInputToken := v.(float64)
		post.CostPerInputToken = &costPerInputToken
	}

	if v, ok := d.GetOk(COST_PER_OUTPUT_TOKEN); ok {
		costPerOutputToken := v.(float64)
		post.CostPerOutputToken = &costPerOutputToken
	}

	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, _, err = client.ld.AIConfigsApi.PostModelConfig(client.ctx, projectKey).ModelConfigPost(post).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to create model config with key %q in project %q: %s", modelConfigKey, projectKey, handleLdapiErr(err))
	}

	d.SetId(fmt.Sprintf("%s/%s", projectKey, modelConfigKey))

	return resourceModelConfigRead(ctx, d, metaRaw)
}

func resourceModelConfigRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return modelConfigRead(ctx, d, metaRaw, false)
}

func resourceModelConfigDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	modelConfigKey := d.Get(KEY).(string)

	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, err = client.ld.AIConfigsApi.DeleteModelConfig(client.ctx, projectKey, modelConfigKey).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to delete model config with key %q in project %q: %s", modelConfigKey, projectKey, handleLdapiErr(err))
	}

	return diags
}

func resourceModelConfigExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	modelConfigKey := d.Get(KEY).(string)

	var res *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, res, err = client.ld.AIConfigsApi.GetModelConfig(client.ctx, projectKey, modelConfigKey).Execute()
		return err
	})
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get model config with key %q in project %q: %s", modelConfigKey, projectKey, handleLdapiErr(err))
	}

	return true, nil
}

func resourceModelConfigImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()
	if id == "" {
		return nil, fmt.Errorf("import ID cannot be empty")
	}

	projectKey, modelConfigKey, err := modelConfigIdToKeys(id)
	if err != nil {
		return nil, err
	}

	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(KEY, modelConfigKey)

	return []*schema.ResourceData{d}, nil
}
