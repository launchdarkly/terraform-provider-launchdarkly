package launchdarkly

import (
	"context"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v17"
)

type webhookSchemaOptions struct {
	isDataSource bool
}

func baseWebhookSchema(options webhookSchemaOptions) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		SECRET: {
			Type:        schema.TypeString,
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Description: "The secret used to sign the webhook.",
			Sensitive:   true,
		},
		NAME: {
			Type:        schema.TypeString,
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Description: "The webhook's human-readable name.",
		},
		STATEMENTS: policyStatementsSchema(policyStatementSchemaOptions{
			optional:    !options.isDataSource,
			computed:    options.isDataSource,
			description: `List of policy statement blocks used to filter webhook events. For more information on webhook policy filters read [Adding a policy filter](https://docs.launchdarkly.com/integrations/webhooks#adding-a-policy-filter).`,
		}),
		TAGS: tagsSchema(tagsSchemaOptions(options)),
	}
}

func webhookRead(ctx context.Context, d *schema.ResourceData, meta interface{}, isDataSource bool) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*Client)
	var webhookID string
	if isDataSource {
		webhookID = d.Get(ID).(string)
	} else {
		webhookID = d.Id()
	}

	var webhook *ldapi.Webhook
	var res *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		webhook, res, err = client.ld.WebhooksApi.GetWebhook(client.ctx, webhookID).Execute()
		return err
	})
	if isStatusNotFound(res) && !isDataSource {
		log.Printf("[WARN] failed to find webhook with id %q, removing from state", webhookID)
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.Errorf("failed to get webhook with id %q: %s", webhookID, handleLdapiErr(err))
	}
	if webhook.Statements != nil {
		statements := policyStatementsToResourceData(webhook.Statements)
		err = d.Set(STATEMENTS, statements)
		if err != nil {
			return diag.Errorf("failed to set statements on webhook with id %q: %v", webhookID, err)
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
		return diag.Errorf("failed to set tags on webhook with id %q: %v", webhookID, err)
	}
	return diags
}
