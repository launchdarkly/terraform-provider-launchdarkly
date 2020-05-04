package launchdarkly

import (
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

func resourceWebhook() *schema.Resource {
	return &schema.Resource{
		Create: resourceWebhookCreate,
		Read:   resourceWebhookRead,
		Update: resourceWebhookUpdate,
		Delete: resourceWebhookDelete,
		Exists: resourceWebhookExists,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			URL: {
				Type:     schema.TypeString,
				Required: true,
			},
			SECRET: {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},
			ENABLED: {
				Type:     schema.TypeBool,
				Required: true,
			},
			NAME: {
				Type:     schema.TypeString,
				Optional: true,
			},
			POLICY_STATEMENTS: policyStatementsSchema(),
			TAGS:              tagsSchema(),
		},
	}
}

func resourceWebhookCreate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	webhookURL := d.Get(URL).(string)
	webhookSecret := d.Get(SECRET).(string)
	webhookOn := d.Get(ENABLED).(bool)
	webhookName := d.Get(NAME).(string)
	statements, err := policyStatementsFromResourceData(d)
	if err != nil {
		return err
	}

	webhookBody := ldapi.WebhookBody{
		Url:        webhookURL,
		Secret:     webhookSecret,
		On:         webhookOn,
		Name:       webhookName,
		Statements: statements,
	}

	// The sign field isn't returned when GETting a webhook so terraform can't import it properly.
	// We hide the field from terraform to avoid import problems.
	if webhookSecret != "" {
		webhookBody.Sign = true
	}

	webhookRaw, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.WebhooksApi.PostWebhook(client.ctx, webhookBody)
	})
	webhook := webhookRaw.(ldapi.Webhook)
	if err != nil {
		return fmt.Errorf("failed to create webhook with name %q: %s", webhookName, handleLdapiErr(err))
	}

	d.SetId(webhook.Id)

	// ld's api does not allow tags to be passed in during webhook creation so we do an update
	err = resourceWebhookUpdate(d, metaRaw)
	if err != nil {
		return fmt.Errorf("error updating after webhook creation. Webhook name: %q", webhookName)
	}

	return resourceWebhookRead(d, metaRaw)
}

func resourceWebhookRead(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	webhookID := d.Id()

	webhookRaw, res, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.WebhooksApi.GetWebhook(client.ctx, webhookID)
	})
	webhook := webhookRaw.(ldapi.Webhook)
	if isStatusNotFound(res) {
		log.Printf("[WARN] failed to find webhook with id %q, removing from state", webhookID)
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get webhook with id %q: %s", webhookID, handleLdapiErr(err))
	}
	statements := policyStatementsToResourceData(webhook.Statements)

	_ = d.Set(URL, webhook.Url)
	_ = d.Set(SECRET, webhook.Secret)
	_ = d.Set(ENABLED, webhook.On)
	_ = d.Set(NAME, webhook.Name)
	err = d.Set(POLICY_STATEMENTS, statements)
	if err != nil {
		return fmt.Errorf("failed to set policy_statements on webhook with id %q: %v", webhookID, err)
	}

	err = d.Set(TAGS, webhook.Tags)
	if err != nil {
		return fmt.Errorf("failed to set tags on webhook with id %q: %v", webhookID, err)
	}
	return nil
}

func resourceWebhookUpdate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	webhookID := d.Id()
	webhookURL := d.Get(URL).(string)
	webhookSecret := d.Get(SECRET).(string)
	webhookOn := d.Get(ENABLED).(bool)
	webhookName := d.Get(NAME).(string)
	webhookTags := stringsFromResourceData(d, TAGS)

	patch := []ldapi.PatchOperation{
		patchReplace("/url", &webhookURL),
		patchReplace("/secret", &webhookSecret),
		patchReplace("/on", &webhookOn),
		patchReplace("/name", &webhookName),
		patchReplace("/tags", &webhookTags),
	}

	statements, err := policyStatementsFromResourceData(d)
	if err != nil {
		return err
	}
	if len(statements) > 0 {
		patch = append(patch, patchReplace("/statements", &statements))
	}

	_, _, err = handleRateLimit(func() (interface{}, *http.Response, error) {
		return handleNoConflict(func() (interface{}, *http.Response, error) {
			return client.ld.WebhooksApi.PatchWebhook(client.ctx, webhookID, patch)
		})
	})
	if err != nil {
		return fmt.Errorf("failed to update webhook with id %q: %s", webhookID, handleLdapiErr(err))
	}

	return resourceWebhookRead(d, metaRaw)
}

func resourceWebhookDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	webhookID := d.Id()

	_, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		res, err := client.ld.WebhooksApi.DeleteWebhook(client.ctx, webhookID)
		return nil, res, err
	})

	if err != nil {
		return fmt.Errorf("failed to delete webhook with id %q: %s", webhookID, handleLdapiErr(err))
	}

	return nil
}

func resourceWebhookExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return webhookExists(d.Id(), metaRaw.(*Client))
}

func webhookExists(webhookID string, meta *Client) (bool, error) {
	_, res, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return meta.ld.WebhooksApi.GetWebhook(meta.ctx, webhookID)
	})
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get webhook with id %q: %s", webhookID, handleLdapiErr(err))
	}

	return true, nil
}
