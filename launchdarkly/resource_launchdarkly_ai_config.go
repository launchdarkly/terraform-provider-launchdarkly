package launchdarkly

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

func resourceAIConfig() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAIConfigCreate,
		ReadContext:   resourceAIConfigRead,
		UpdateContext: resourceAIConfigUpdate,
		DeleteContext: resourceAIConfigDelete,
		Exists:        resourceAIConfigExists,

		Importer: &schema.ResourceImporter{
			StateContext: resourceAIConfigImport,
		},

		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
			isInverted := diff.Get(IS_INVERTED).(bool)
			metricKey := diff.Get(EVALUATION_METRIC_KEY).(string)
			if isInverted && metricKey == "" {
				return fmt.Errorf("is_inverted requires evaluation_metric_key to be set")
			}
			return nil
		},

		Description: `Provides a LaunchDarkly AI Config resource.

This resource allows you to create and manage AI Configurations within your LaunchDarkly project.`,

		Schema: baseAIConfigSchema(false),
	}
}

func resourceAIConfigCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)

	projectKey := d.Get(PROJECT_KEY).(string)
	configKey := d.Get(KEY).(string)
	name := d.Get(NAME).(string)

	post := *ldapi.NewAIConfigPost(configKey, name)
	// The constructor sets Description to "" by default, which the API rejects.
	// Clear it so it's omitted from the JSON unless the user explicitly sets it.
	post.Description = nil

	if v, ok := d.GetOk(DESCRIPTION); ok {
		description := v.(string)
		post.Description = &description
	}

	if v, ok := d.GetOk(MODE); ok {
		mode := v.(string)
		post.Mode = &mode
	}

	if v, ok := d.GetOk(MAINTAINER_ID); ok {
		maintainerId := v.(string)
		post.MaintainerId = &maintainerId
	}

	if v, ok := d.GetOk(MAINTAINER_TEAM_KEY); ok {
		maintainerTeamKey := v.(string)
		post.MaintainerTeamKey = &maintainerTeamKey
	}

	if v, ok := d.GetOk(EVALUATION_METRIC_KEY); ok {
		evaluationMetricKey := v.(string)
		post.EvaluationMetricKey = &evaluationMetricKey

		// Only send is_inverted when evaluation_metric_key is set, since it's
		// meaningless without a metric. Use d.Get() instead of d.GetOk() because
		// GetOk returns ok=false for bool(false), which would silently drop an
		// explicit is_inverted = false.
		isInverted := d.Get(IS_INVERTED).(bool)
		post.IsInverted = &isInverted
	}

	if v, ok := d.GetOk(TAGS); ok {
		post.Tags = interfaceSliceToStringSlice(v.(*schema.Set).List())
	}

	err := retryOnTransient400(client, 3, func() error {
		_, _, err := client.ld.AIConfigsApi.PostAIConfig(client.ctx, projectKey).AIConfigPost(post).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to create AI config with key %q in project %q: %s", configKey, projectKey, handleLdapiErr(err))
	}

	d.SetId(fmt.Sprintf("%s/%s", projectKey, configKey))

	return resourceAIConfigRead(ctx, d, metaRaw)
}

func resourceAIConfigRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return aiConfigRead(ctx, d, metaRaw, false)
}

func resourceAIConfigUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)

	projectKey := d.Get(PROJECT_KEY).(string)
	configKey := d.Get(KEY).(string)

	patch := *ldapi.NewAIConfigPatch()
	hasChanges := false

	if d.HasChange(NAME) {
		name := d.Get(NAME).(string)
		patch.Name = &name
		hasChanges = true
	}

	if d.HasChange(DESCRIPTION) {
		description := d.Get(DESCRIPTION).(string)
		patch.Description = &description
		hasChanges = true
	}

	if d.HasChange(MAINTAINER_ID) {
		if v, ok := d.GetOk(MAINTAINER_ID); ok {
			maintainerId := v.(string)
			patch.MaintainerId = &maintainerId
		}
		hasChanges = true
	}

	if d.HasChange(MAINTAINER_TEAM_KEY) {
		if v, ok := d.GetOk(MAINTAINER_TEAM_KEY); ok {
			maintainerTeamKey := v.(string)
			patch.MaintainerTeamKey = &maintainerTeamKey
		}
		hasChanges = true
	}

	if d.HasChange(TAGS) {
		tags := stringsFromResourceData(d, TAGS)
		patch.Tags = tags
		hasChanges = true
	}

	if d.HasChange(EVALUATION_METRIC_KEY) {
		// Use d.Get() instead of d.GetOk() so users can unset to empty string.
		evaluationMetricKey := d.Get(EVALUATION_METRIC_KEY).(string)
		patch.EvaluationMetricKey = &evaluationMetricKey
		hasChanges = true
	}

	if d.HasChange(IS_INVERTED) {
		// Use d.Get() instead of d.GetOk() because GetOk returns ok=false
		// for bool(false), silently dropping true→false changes.
		isInverted := d.Get(IS_INVERTED).(bool)
		patch.IsInverted = &isInverted
		hasChanges = true
	}

	if hasChanges {
		var err error
		err = client.withConcurrency(client.ctx, func() error {
			_, _, err = client.ld.AIConfigsApi.PatchAIConfig(client.ctx, projectKey, configKey).AIConfigPatch(patch).Execute()
			return err
		})
		if err != nil {
			return diag.Errorf("failed to update AI config with key %q in project %q: %s", configKey, projectKey, handleLdapiErr(err))
		}
	}

	return resourceAIConfigRead(ctx, d, metaRaw)
}

func resourceAIConfigDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	configKey := d.Get(KEY).(string)

	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, err = client.ld.AIConfigsApi.DeleteAIConfig(client.ctx, projectKey, configKey).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to delete AI config with key %q in project %q: %s", configKey, projectKey, handleLdapiErr(err))
	}

	return diags
}

func resourceAIConfigExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	configKey := d.Get(KEY).(string)

	var res *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, res, err = client.ld.AIConfigsApi.GetAIConfig(client.ctx, projectKey, configKey).Execute()
		return err
	})
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get AI config with key %q in project %q: %s", configKey, projectKey, handleLdapiErr(err))
	}

	return true, nil
}

func resourceAIConfigImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()
	if id == "" {
		return nil, fmt.Errorf("import ID cannot be empty")
	}

	projectKey, configKey, err := aiConfigIdToKeys(id)
	if err != nil {
		return nil, err
	}

	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(KEY, configKey)

	return []*schema.ResourceData{d}, nil
}
