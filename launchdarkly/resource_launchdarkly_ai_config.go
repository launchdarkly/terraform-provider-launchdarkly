package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceAiConfig() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAiConfigCreate,
		ReadContext:   resourceAiConfigRead,
		UpdateContext: resourceAiConfigUpdate,
		DeleteContext: resourceAiConfigDelete,
		Schema:        baseAiConfigSchema(false),
		Importer: &schema.ResourceImporter{
			StateContext: resourceAiConfigImport,
		},

		Description: `Provides a LaunchDarkly AI Config resource.

This resource allows you to create and manage AI Configs within your LaunchDarkly organization.

To learn more about AI Configs, read the [AI Configs documentation](https://docs.launchdarkly.com/home/ai-configs).`,
	}
}

func resourceAiConfigCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)

	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)
	name := d.Get(NAME).(string)

	post := AiConfigPost{
		Key:  key,
		Name: name,
	}

	if description, ok := d.GetOk(DESCRIPTION); ok {
		post.Description = description.(string)
	}

	if mode, ok := d.GetOk(MODE); ok {
		post.Mode = mode.(string)
	}

	if maintainerId, ok := d.GetOk(MAINTAINER_ID); ok {
		post.MaintainerId = maintainerId.(string)
	}

	if maintainerTeamKey, ok := d.GetOk(MAINTAINER_TEAM_KEY); ok {
		post.MaintainerTeamKey = maintainerTeamKey.(string)
	}

	if tags, ok := d.GetOk(TAGS); ok {
		post.Tags = interfaceSliceToStringSlice(tags.(*schema.Set).List())
	}

	if evaluationMetricKey, ok := d.GetOk(EVALUATION_METRIC_KEY); ok {
		post.EvaluationMetricKey = evaluationMetricKey.(string)
	}

	if isInverted, ok := d.GetOk(IS_INVERTED); ok {
		v := isInverted.(bool)
		post.IsInverted = &v
	}

	_, err := createAiConfig(client, projectKey, post)
	if err != nil {
		return diag.Errorf("failed to create AI Config %q in project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	d.SetId(fmt.Sprintf("%s/%s", projectKey, key))

	return resourceAiConfigRead(ctx, d, metaRaw)
}

func resourceAiConfigRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return aiConfigRead(ctx, d, metaRaw, false)
}

func resourceAiConfigUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)

	projectKey := d.Get(PROJECT_KEY).(string)
	configKey := d.Get(KEY).(string)

	patch := AiConfigPatch{}

	if d.HasChange(NAME) {
		name := d.Get(NAME).(string)
		patch.Name = &name
	}

	if d.HasChange(DESCRIPTION) {
		description := d.Get(DESCRIPTION).(string)
		patch.Description = &description
	}

	if d.HasChange(MAINTAINER_ID) {
		if maintainerId, ok := d.GetOk(MAINTAINER_ID); ok {
			id := maintainerId.(string)
			patch.MaintainerId = &id
		}
	}

	if d.HasChange(MAINTAINER_TEAM_KEY) {
		if maintainerTeamKey, ok := d.GetOk(MAINTAINER_TEAM_KEY); ok {
			key := maintainerTeamKey.(string)
			patch.MaintainerTeamKey = &key
		}
	}

	if d.HasChange(TAGS) {
		patch.Tags = interfaceSliceToStringSlice(d.Get(TAGS).(*schema.Set).List())
	}

	if d.HasChange(EVALUATION_METRIC_KEY) {
		evaluationMetricKey := d.Get(EVALUATION_METRIC_KEY).(string)
		patch.EvaluationMetricKey = &evaluationMetricKey
	}

	if d.HasChange(IS_INVERTED) {
		isInverted := d.Get(IS_INVERTED).(bool)
		patch.IsInverted = &isInverted
	}

	_, err := patchAiConfig(client, projectKey, configKey, patch)
	if err != nil {
		return diag.Errorf("failed to update AI Config %q in project %q: %s", configKey, projectKey, handleLdapiErr(err))
	}

	return resourceAiConfigRead(ctx, d, metaRaw)
}

func resourceAiConfigDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)

	projectKey := d.Get(PROJECT_KEY).(string)
	configKey := d.Get(KEY).(string)

	err := deleteAiConfig(client, projectKey, configKey)
	if err != nil {
		return diag.Errorf("failed to delete AI Config %q in project %q: %s", configKey, projectKey, handleLdapiErr(err))
	}

	return diags
}

func resourceAiConfigImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()

	projectKey, configKey, err := aiConfigIdToKeys(id)
	if err != nil {
		return nil, err
	}
	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(KEY, configKey)

	return []*schema.ResourceData{d}, nil
}
