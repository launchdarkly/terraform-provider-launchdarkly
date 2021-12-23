package launchdarkly

import (
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v7"
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
		STATEMENTS: policyStatementsSchema(policyStatementSchemaOptions{}),
		TAGS:       tagsSchema(),
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
		return client.ld.WebhooksApi.GetWebhook(client.ctx, webhookID).Execute()
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
	if webhook.Statements != nil {
		statements := policyStatementsToResourceData(*webhook.Statements)
		err = d.Set(STATEMENTS, statements)
		if err != nil {
			return fmt.Errorf("failed to set statements on webhook with id %q: %v", webhookID, err)
		}
	}

	if isDataSource {
		d.SetId(webhook.Id)
	}
	_ = d.Set(URL, webhook.Url)
	_ = d.Set(SECRET, webhook.Secret)
	_ = d.Set(ON, webhook.On)
	_ = d.Set(NAME, webhook.Name)

	err = d.Set(TAGS, webhook.Tags)
	if err != nil {
		return fmt.Errorf("failed to set tags on webhook with id %q: %v", webhookID, err)
	}
	return nil
}
