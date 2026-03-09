package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	ldapi "github.com/launchdarkly/api-client-go/v22"
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

	post := *ldapi.NewAIConfigPost(key, name)

	if description, ok := d.GetOk(DESCRIPTION); ok {
		post.SetDescription(description.(string))
	}

	if mode, ok := d.GetOk(MODE); ok {
		post.SetMode(mode.(string))
	}

	if maintainerId, ok := d.GetOk(MAINTAINER_ID); ok {
		post.SetMaintainerId(maintainerId.(string))
	}

	if maintainerTeamKey, ok := d.GetOk(MAINTAINER_TEAM_KEY); ok {
		post.SetMaintainerTeamKey(maintainerTeamKey.(string))
	}

	if tags, ok := d.GetOk(TAGS); ok {
		post.SetTags(interfaceSliceToStringSlice(tags.(*schema.Set).List()))
	}

	if evaluationMetricKey, ok := d.GetOk(EVALUATION_METRIC_KEY); ok {
		post.SetEvaluationMetricKey(evaluationMetricKey.(string))
	}

	if isInverted, ok := d.GetOk(IS_INVERTED); ok {
		post.SetIsInverted(isInverted.(bool))
	}

	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, _, err = client.ld.AIConfigsApi.PostAIConfig(client.ctx, projectKey).AIConfigPost(post).Execute()
		return err
	})
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

	patch := *ldapi.NewAIConfigPatch()

	if d.HasChange(NAME) {
		patch.SetName(d.Get(NAME).(string))
	}

	if d.HasChange(DESCRIPTION) {
		patch.SetDescription(d.Get(DESCRIPTION).(string))
	}

	if d.HasChange(MAINTAINER_ID) {
		if maintainerId, ok := d.GetOk(MAINTAINER_ID); ok {
			patch.SetMaintainerId(maintainerId.(string))
		}
	}

	if d.HasChange(MAINTAINER_TEAM_KEY) {
		if maintainerTeamKey, ok := d.GetOk(MAINTAINER_TEAM_KEY); ok {
			patch.SetMaintainerTeamKey(maintainerTeamKey.(string))
		}
	}

	if d.HasChange(TAGS) {
		patch.SetTags(interfaceSliceToStringSlice(d.Get(TAGS).(*schema.Set).List()))
	}

	if d.HasChange(EVALUATION_METRIC_KEY) {
		patch.SetEvaluationMetricKey(d.Get(EVALUATION_METRIC_KEY).(string))
	}

	if d.HasChange(IS_INVERTED) {
		patch.SetIsInverted(d.Get(IS_INVERTED).(bool))
	}

	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, _, err = client.ld.AIConfigsApi.PatchAIConfig(client.ctx, projectKey, configKey).AIConfigPatch(patch).Execute()
		return err
	})
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

	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, err = client.ld.AIConfigsApi.DeleteAIConfig(client.ctx, projectKey, configKey).Execute()
		return err
	})
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
