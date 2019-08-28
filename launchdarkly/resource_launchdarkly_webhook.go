package launchdarkly

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
	"github.com/pkg/errors"
)

func resourceWebhook() *schema.Resource {
	return &schema.Resource{
		Create: resourceWebhookCreate,
		Read:   resourceWebhookRead,
		Update: resourceWebhookUpdate,
		Delete: resourceWebhookDelete,
		Exists: resourceWebhookExists,

		Importer: &schema.ResourceImporter{
			State: resourceWebhookImport,
		},

		Schema: map[string]*schema.Schema{
			url: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			secret: &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},
			on: &schema.Schema{
				Type:     schema.TypeBool,
				Required: true,
			},
			name: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			tags: tagsSchema(),
		},
	}
}

func resourceWebhookCreate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	webhookURL := d.Get(url).(string)
	webhookSecret := d.Get(secret).(string)
	webhookOn := d.Get(on).(bool)
	webhookName := d.Get(name).(string)

	webhookBody := ldapi.WebhookBody{
		Url:    webhookURL,
		Secret: webhookSecret,
		On:     webhookOn,
		Name:   webhookName,
	}

	// The sign field isn't returned when GETting a webhook so terraform can't import it properly.
	// We hide the field from terraform to avoid import problems.
	if webhookSecret != "" {
		webhookBody.Sign = true
	}

	webhook, _, err := client.ld.WebhooksApi.PostWebhook(client.ctx, webhookBody)
	if err != nil {
		return fmt.Errorf("failed to create webhook with name %q: %s", webhookName, handleLdapiErr(err))
	}

	d.SetId(webhook.Id)

	// ld's api does not allow tags to be passed in during webhook creation so we do an update
	err = resourceWebhookUpdate(d, metaRaw)
	if err != nil {
		return errors.Wrapf(err, "During webhook creation. Webhook name: %q", webhookName)
	}

	return resourceWebhookRead(d, metaRaw)
}

func resourceWebhookRead(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	webhookID := d.Id()

	webhook, _, err := client.ld.WebhooksApi.GetWebhook(client.ctx, webhookID)
	if err != nil {
		return fmt.Errorf("failed to get webhook with id %q: %s", webhookID, handleLdapiErr(err))
	}

	_ = d.Set(url, webhook.Url)
	_ = d.Set(secret, webhook.Secret)
	_ = d.Set(on, webhook.On)
	_ = d.Set(name, webhook.Name)
	err = d.Set(tags, webhook.Tags)
	if err != nil {
		return fmt.Errorf("failed to set tags on webhook with id %q: %v", webhookID, err)
	}
	return nil
}

func resourceWebhookUpdate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	webhookID := d.Id()
	webhookURL := d.Get(url).(string)
	webhookSecret := d.Get(secret).(string)
	webhookOn := d.Get(on).(bool)
	webhookName := d.Get(name).(string)
	webhookTags := stringsFromResourceData(d, tags)

	patch := []ldapi.PatchOperation{
		patchReplace("/url", &webhookURL),
		patchReplace("/secret", &webhookSecret),
		patchReplace("/on", &webhookOn),
		patchReplace("/name", &webhookName),
		patchReplace("/tags", &webhookTags),
	}

	_, _, err := client.ld.WebhooksApi.PatchWebhook(client.ctx, webhookID, patch)
	if err != nil {
		return fmt.Errorf("failed to update webhook with id %q: %s", webhookID, handleLdapiErr(err))
	}

	return resourceWebhookRead(d, metaRaw)
}

func resourceWebhookDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	webhookID := d.Id()

	_, err := client.ld.WebhooksApi.DeleteWebhook(client.ctx, webhookID)
	if err != nil {
		return fmt.Errorf("failed to delete webhook with id %q: %s", webhookID, handleLdapiErr(err))
	}

	return nil
}

func resourceWebhookExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return webhookExists(d.Id(), metaRaw.(*Client))
}

func webhookExists(webhookID string, meta *Client) (bool, error) {
	_, httpResponse, err := meta.ld.WebhooksApi.GetWebhook(meta.ctx, webhookID)
	if httpResponse != nil && httpResponse.StatusCode == 404 {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get webhook with id %q: %s", webhookID, handleLdapiErr(err))
	}

	return true, nil
}

func resourceWebhookImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	d.SetId(d.Id())

	if err := resourceWebhookRead(d, meta); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}
