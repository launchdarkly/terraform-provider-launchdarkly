package launchdarkly

import (
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

func baseWebhookSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		SECRET: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "If sign is true, and the secret attribute is omitted, LaunchDarkly will automatically generate a secret for you",
			Sensitive:   true,
		},
		NAME: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "A human-readable name for your webhook",
		},
		POLICY_STATEMENTS: policyStatementsSchema(
			policyStatementSchemaOptions{
				deprecated:    "'policy_statements' is deprecated in favor of 'statements'",
				conflictsWith: []string{STATEMENTS},
			},
		),
		STATEMENTS: policyStatementsSchema(policyStatementSchemaOptions{
			conflictsWith: []string{POLICY_STATEMENTS},
		}),
		TAGS: tagsSchema(),
	}
}

func webhookRead(d *schema.ResourceData, meta interface{}, isDataSource bool) error {
	client := meta.(*Client)
	var webhookID string
	if isDataSource {
		webhookID = d.Get(ID).(string)
	} else {
		webhookID = d.Id()
	}

	webhookRaw, res, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.WebhooksApi.GetWebhook(client.ctx, webhookID)
	})
	webhook := webhookRaw.(ldapi.Webhook)
	if isStatusNotFound(res) && !isDataSource {
		log.Printf("[WARN] failed to find webhook with id %q, removing from state", webhookID)
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get webhook with id %q: %s", webhookID, handleLdapiErr(err))
	}
	statements := policyStatementsToResourceData(webhook.Statements)

	if isDataSource {
		d.SetId(webhook.Id)
	}
	_ = d.Set(URL, webhook.Url)
	_ = d.Set(SECRET, webhook.Secret)

	// "enabled" is deprecated in favor of "on". For data sources, set both, for resources only set the one being used.
	if isDataSource {
		_ = d.Set(ENABLED, webhook.On)
		_ = d.Set(ON, webhook.On)
	} else {
		if _, ok := d.GetOkExists(ENABLED); ok {
			_ = d.Set(ENABLED, webhook.On)
		} else {
			_ = d.Set(ON, webhook.On)
		}
	}

	_ = d.Set(NAME, webhook.Name)

	// // "policy_statements" is deprecated in favor of "statements". For data sources, set both, for resources only set the one being used.
	if isDataSource {
		err = d.Set(POLICY_STATEMENTS, statements)
		if err != nil {
			return fmt.Errorf("failed to set policy_statements on webhook with id %q: %v", webhookID, err)
		}
		err = d.Set(STATEMENTS, statements)
		if err != nil {
			return fmt.Errorf("failed to set statements on webhook with id %q: %v", webhookID, err)
		}
	} else {
		if _, ok := d.GetOk(POLICY_STATEMENTS); ok {
			err = d.Set(POLICY_STATEMENTS, statements)
			if err != nil {
				return fmt.Errorf("failed to set policy_statements on webhook with id %q: %v", webhookID, err)
			}
		} else {
			err = d.Set(STATEMENTS, statements)
			if err != nil {
				return fmt.Errorf("failed to set statements on webhook with id %q: %v", webhookID, err)
			}
		}
	}

	err = d.Set(TAGS, webhook.Tags)
	if err != nil {
		return fmt.Errorf("failed to set tags on webhook with id %q: %v", webhookID, err)
	}
	return nil
}
