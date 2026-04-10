package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

func resourceAIConfigVariation() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAIConfigVariationCreate,
		ReadContext:   resourceAIConfigVariationRead,
		UpdateContext: resourceAIConfigVariationUpdate,
		DeleteContext: resourceAIConfigVariationDelete,
		Exists:        resourceAIConfigVariationExists,

		Importer: &schema.ResourceImporter{
			StateContext: resourceAIConfigVariationImport,
		},

		Description: `Provides a LaunchDarkly AI Config variation resource.

This resource allows you to create and manage AI Config variations within your LaunchDarkly project.`,

		Schema: aiConfigVariationSchema(false),
	}
}

func resourceAIConfigVariationCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)

	projectKey := d.Get(PROJECT_KEY).(string)
	configKey := d.Get(AI_CONFIG_KEY).(string)
	variationKey := d.Get(KEY).(string)
	name := d.Get(NAME).(string)

	// Parse the model JSON string to a map; use empty map if not provided
	// because NewAIConfigVariationPost requires a model parameter.
	model := map[string]interface{}{}
	if v, ok := d.GetOk(MODEL); ok {
		modelMap, err := jsonStringToMap(v.(string))
		if err != nil {
			return diag.Errorf("failed to parse model JSON: %s", err)
		}
		if modelMap != nil {
			model = modelMap
		}
	}

	post := ldapi.NewAIConfigVariationPost(variationKey, name)
	post.Model = model

	if v, ok := d.GetOk(DESCRIPTION); ok {
		description := v.(string)
		post.Description = &description
	}

	if v, ok := d.GetOk(INSTRUCTIONS); ok {
		instructions := v.(string)
		post.Instructions = &instructions
	}

	if v, ok := d.GetOk(MODEL_CONFIG_KEY); ok {
		modelConfigKey := v.(string)
		post.ModelConfigKey = &modelConfigKey
	}

	if v, ok := d.GetOk(MESSAGES); ok {
		raw := v.([]interface{})
		if len(raw) > 0 {
			post.Messages = messagesFromResourceData(d)
		}
	}

	if v, ok := d.GetOk(TOOL_KEYS); ok {
		toolKeys := stringsFromSchemaSet(v.(*schema.Set))
		post.ToolKeys = toolKeys
	}

	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, _, err = client.ld.AIConfigsApi.PostAIConfigVariation(client.ctx, projectKey, configKey).AIConfigVariationPost(*post).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to create AI config variation with key %q in config %q project %q: %s", variationKey, configKey, projectKey, handleLdapiErr(err))
	}

	d.SetId(fmt.Sprintf("%s/%s/%s", projectKey, configKey, variationKey))

	return resourceAIConfigVariationReadWithRetry(ctx, d, metaRaw)
}

func resourceAIConfigVariationRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return aiConfigVariationRead(ctx, d, metaRaw, false)
}

func resourceAIConfigVariationUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)

	projectKey := d.Get(PROJECT_KEY).(string)
	configKey := d.Get(AI_CONFIG_KEY).(string)
	variationKey := d.Get(KEY).(string)

	patch := ldapi.NewAIConfigVariationPatch()

	if d.HasChange(NAME) {
		name := d.Get(NAME).(string)
		patch.Name = &name
	}

	if d.HasChange(DESCRIPTION) {
		description := d.Get(DESCRIPTION).(string)
		patch.Description = &description
	}

	if d.HasChange(INSTRUCTIONS) {
		instructions := d.Get(INSTRUCTIONS).(string)
		patch.Instructions = &instructions
	}

	if d.HasChange(MODEL) {
		modelMap, err := jsonStringToMap(d.Get(MODEL).(string))
		if err != nil {
			return diag.Errorf("failed to parse model JSON: %s", err)
		}
		patch.Model = modelMap
	}

	if d.HasChange(MODEL_CONFIG_KEY) {
		modelConfigKey := d.Get(MODEL_CONFIG_KEY).(string)
		patch.ModelConfigKey = &modelConfigKey
	}

	if d.HasChange(MESSAGES) {
		patch.Messages = messagesFromResourceData(d)
	}

	if d.HasChange(STATE) {
		state := d.Get(STATE).(string)
		patch.State = &state
	}

	if d.HasChange(TOOL_KEYS) {
		toolKeys := stringsFromSchemaSet(d.Get(TOOL_KEYS).(*schema.Set))
		patch.ToolKeys = toolKeys
	}

	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, _, err = client.ld.AIConfigsApi.PatchAIConfigVariation(client.ctx, projectKey, configKey, variationKey).AIConfigVariationPatch(*patch).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to update AI config variation with key %q in config %q project %q: %s", variationKey, configKey, projectKey, handleLdapiErr(err))
	}

	return resourceAIConfigVariationReadWithRetry(ctx, d, metaRaw)
}

func resourceAIConfigVariationDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)

	projectKey := d.Get(PROJECT_KEY).(string)
	configKey := d.Get(AI_CONFIG_KEY).(string)
	variationKey := d.Get(KEY).(string)

	var res *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		res, err = client.ld.AIConfigsApi.DeleteAIConfigVariation(client.ctx, projectKey, configKey, variationKey).Execute()
		return err
	})
	if err != nil {
		// The API returns 404 if the parent AI config was already deleted (cascading delete).
		// The API returns 400 "Cannot delete the last variation" if this is the only variation —
		// in that case the parent AI config delete will cascade, so we can safely ignore it.
		if isStatusNotFound(res) {
			return diags
		}
		errMsg := handleLdapiErr(err).Error()
		if strings.Contains(errMsg, "Cannot delete the last variation") {
			log.Printf("[WARN] cannot delete last variation %q in config %q project %q — will be removed when parent AI config is deleted", variationKey, configKey, projectKey)
			return diags
		}
		return diag.Errorf("failed to delete AI config variation with key %q in config %q project %q: %s", variationKey, configKey, projectKey, handleLdapiErr(err))
	}

	return diags
}

func resourceAIConfigVariationExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)

	projectKey := d.Get(PROJECT_KEY).(string)
	configKey := d.Get(AI_CONFIG_KEY).(string)
	variationKey := d.Get(KEY).(string)

	var res *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, res, err = client.ld.AIConfigsApi.GetAIConfigVariation(client.ctx, projectKey, configKey, variationKey).Execute()
		return err
	})
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get AI config variation with key %q in config %q project %q: %s", variationKey, configKey, projectKey, handleLdapiErr(err))
	}

	return true, nil
}

func resourceAIConfigVariationImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()
	if id == "" {
		return nil, fmt.Errorf("import ID cannot be empty")
	}

	projectKey, configKey, variationKey, err := variationIdToKeys(id)
	if err != nil {
		return nil, err
	}

	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(AI_CONFIG_KEY, configKey)
	_ = d.Set(KEY, variationKey)

	return []*schema.ResourceData{d}, nil
}
