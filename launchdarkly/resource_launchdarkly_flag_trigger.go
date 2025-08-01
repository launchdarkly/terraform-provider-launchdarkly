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

func resourceFlagTrigger() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceFlagTriggerCreate,
		ReadContext:   resourceFlagTriggerRead,
		UpdateContext: resourceFlagTriggerUpdate,
		DeleteContext: resourceFlagTriggerDelete,
		Exists:        resourceFlagTriggerExists,

		Importer: &schema.ResourceImporter{
			StateContext: resourceFlagTriggerImport,
		},
		Schema: baseFlagTriggerSchema(false),

		Description: `Provides a LaunchDarkly flag trigger resource.

-> **Note:** Flag triggers are available to customers on an Enterprise LaunchDarkly plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).

This resource allows you to create and manage flag triggers within your LaunchDarkly organization.

-> **Note:** This resource will store sensitive unique trigger URL value in plaintext in your Terraform state. Be sure your state is configured securely before using this resource. See https://www.terraform.io/docs/state/sensitive-data.html for more details.`,
	}
}

func resourceFlagTriggerCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	envKey := d.Get(ENV_KEY).(string)
	flagKey := d.Get(FLAG_KEY).(string)
	integrationKey := d.Get(INTEGRATION_KEY).(string)
	instructions := instructionsFromResourceData(d, "POST")

	enabled := d.Get(ENABLED).(bool)

	triggerBody := ldapi.NewTriggerPost(integrationKey)
	triggerBody.Instructions = instructions

	var createdTrigger *ldapi.TriggerWorkflowRep
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		createdTrigger, _, err = client.ld.FlagTriggersApi.CreateTriggerWorkflow(client.ctx, projectKey, envKey, flagKey).TriggerPost(*triggerBody).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to create %s trigger for proj/env/flag %s/%s/%s: %s", integrationKey, projectKey, envKey, flagKey, err.Error())
	}
	_ = d.Set(TRIGGER_URL, createdTrigger.TriggerURL)

	if createdTrigger.Id == nil {
		return diag.Errorf("received a nil trigger ID when creating %s trigger for proj/env/flag %s/%s/%s: %s", integrationKey, projectKey, envKey, flagKey, err.Error())
	}
	d.SetId(*createdTrigger.Id)

	// if enabled is false upon creation, we need to do a patch since the create endpoint
	// does not accept multiple instructions
	if !enabled {
		instructions = []map[string]interface{}{{
			KIND: "disableTrigger",
		}}
		input := ldapi.FlagTriggerInput{
			Instructions: instructions,
		}

		err = client.withConcurrency(client.ctx, func() error {
			_, _, err = client.ld.FlagTriggersApi.PatchTriggerWorkflow(client.ctx, projectKey, envKey, flagKey, *createdTrigger.Id).FlagTriggerInput(input).Execute()
			return err
		})
		if err != nil {
			return diag.Errorf("failed to update %s trigger for proj/env/flag %s/%s/%s: %s", integrationKey, projectKey, envKey, flagKey, err.Error())
		}
	}
	return resourceFlagTriggerRead(ctx, d, metaRaw)
}

func resourceFlagTriggerRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return flagTriggerRead(ctx, d, metaRaw, false)
}

func resourceFlagTriggerUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	envKey := d.Get(ENV_KEY).(string)
	flagKey := d.Get(FLAG_KEY).(string)
	integrationKey := d.Get(INTEGRATION_KEY).(string)
	instructions := instructionsFromResourceData(d, "PATCH")

	triggerId := d.Id()

	oldEnabled, newEnabled := d.GetChange(ENABLED)
	if oldEnabled.(bool) != newEnabled.(bool) {
		if newEnabled.(bool) {
			instructions = append(instructions, map[string]interface{}{
				KIND: "enableTrigger",
			})
		} else {
			instructions = append(instructions, map[string]interface{}{
				KIND: "disableTrigger",
			})
		}
	}
	input := ldapi.FlagTriggerInput{
		Instructions: instructions,
	}

	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, _, err = client.ld.FlagTriggersApi.PatchTriggerWorkflow(client.ctx, projectKey, envKey, flagKey, triggerId).FlagTriggerInput(input).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to update %s trigger for proj/env/flag %s/%s/%s", integrationKey, projectKey, envKey, flagKey)
	}
	return resourceFlagTriggerRead(ctx, d, metaRaw)
}

func resourceFlagTriggerDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	integrationKey := d.Get(INTEGRATION_KEY).(string)
	projectKey := d.Get(PROJECT_KEY).(string)
	envKey := d.Get(ENV_KEY).(string)
	flagKey := d.Get(FLAG_KEY).(string)

	triggerId := d.Id()

	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, err = client.ld.FlagTriggersApi.DeleteTriggerWorkflow(client.ctx, projectKey, envKey, flagKey, triggerId).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to delete %s trigger with ID %s for proj/env/flag %s/%s/%s", integrationKey, triggerId, projectKey, envKey, flagKey)
	}
	return diag.Diagnostics{}
}

func resourceFlagTriggerExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	integrationKey := d.Get(INTEGRATION_KEY).(string)
	projectKey := d.Get(PROJECT_KEY).(string)
	envKey := d.Get(ENV_KEY).(string)
	flagKey := d.Get(FLAG_KEY).(string)

	triggerId := d.Id()

	var res *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, res, err = client.ld.FlagTriggersApi.GetTriggerWorkflowById(client.ctx, projectKey, flagKey, envKey, triggerId).Execute()
		return err
	})
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check if %s trigger with ID %s exists in proj/env/flag %s/%s/%s: %s", integrationKey, triggerId, projectKey, envKey, flagKey, handleLdapiErr(err))
	}
	return true, nil
}

func resourceFlagTriggerImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	projectKey, envKey, flagKey, triggerId, err := triggerImportIdToKeys(d.Id())
	if err != nil {
		return nil, err
	}
	d.SetId(triggerId)

	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(ENV_KEY, envKey)
	_ = d.Set(FLAG_KEY, flagKey)
	return []*schema.ResourceData{d}, nil
}

func triggerImportIdToKeys(id string) (projectKey string, envKey string, flagKey string, triggerId string, err error) {
	if strings.Count(id, "/") != 3 {
		return "", "", "", "", fmt.Errorf("found unexpected trigger id format: %q expected format: 'project_key/env_key/flag_key/trigger_id'", triggerId)
	}
	parts := strings.SplitN(id, "/", 4)
	projectKey, envKey, flagKey, triggerId = parts[0], parts[1], parts[2], parts[3]
	return projectKey, envKey, flagKey, triggerId, nil
}
