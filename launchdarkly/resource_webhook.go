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
			sign: &schema.Schema{
				Type:     schema.TypeBool,
				Required: true,
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
	webhookUrl := d.Get(url).(string)
	webhookSecret := d.Get(secret).(string)
	webhookSign := d.Get(sign).(bool)
	webhookOn := d.Get(sign).(bool)
	webhookName := d.Get(name).(string)

	webhookBody := ldapi.WebhookBody{
		Url:    webhookUrl,
		Secret: webhookSecret,
		Sign:   webhookSign,
		On:     webhookOn,
		Name:   webhookName,
	}

	webhook, _, err := client.LaunchDarkly.WebhooksApi.PostWebhook(client.Ctx, webhookBody)
	if err != nil {
		return fmt.Errorf("failed to create webhook with name %q: %s", webhookName, handleLdapiErr(err))
	}

	d.SetId(webhook.Id)

	// LaunchDarkly's api does not allow tags to be passed in during webhook creation so we do an update
	err = resourceWebhookUpdate(d, metaRaw)
	if err != nil {
		return errors.Wrapf(err, "During webhook creation. Webhook name: %q", webhookName)
	}

	return resourceWebhookRead(d, metaRaw)
}

func resourceWebhookRead(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	webhookId := d.Id()

	webhook, _, err := client.LaunchDarkly.WebhooksApi.GetWebhook(client.Ctx, webhookId)
	if err != nil {
		return fmt.Errorf("failed to get webhook with id %q: %s", webhookId, handleLdapiErr(err))
	}

	d.Set(url, webhook.Url)
	d.Set(secret, webhook.Secret)
	d.Set(on, webhook.On)
	d.Set(name, webhook.Name)
	err = d.Set(tags, webhook.Tags)
	if err != nil {
		return fmt.Errorf("failed to set tags on webhook with id %q: %v", webhookId, err)
	}
	return nil
}

func resourceWebhookUpdate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	webhookId := d.Id()
	webhookUrl := d.Get(url).(string)
	webhookSecret := d.Get(secret).(string)
	webhookOn := d.Get(on).(bool)
	webhookName := d.Get(name).(string)
	webhookTags := stringsFromResourceData(d, tags)

	patch := []ldapi.PatchOperation{
		patchReplace("/url", &webhookUrl),
		patchReplace("/secret", &webhookSecret),
		patchReplace("/on", &webhookOn),
		patchReplace("/name", &webhookName),
		patchReplace("/tags", &webhookTags),
	}

	_, _, err := client.LaunchDarkly.WebhooksApi.PatchWebhook(client.Ctx, webhookId, patch)
	if err != nil {
		return fmt.Errorf("failed to update webhook with id %q: %s", webhookId, handleLdapiErr(err))
	}

	return resourceWebhookRead(d, metaRaw)
}

func resourceWebhookDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	webhookId := d.Id()

	_, err := client.LaunchDarkly.WebhooksApi.DeleteWebhook(client.Ctx, webhookId)
	if err != nil {
		return fmt.Errorf("failed to delete webhook with id %q: %s", webhookId, handleLdapiErr(err))
	}

	return nil
}

func resourceWebhookExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return webhookExists(d.Id(), metaRaw.(*Client))
}

func webhookExists(webhookId string, meta *Client) (bool, error) {
	_, httpResponse, err := meta.LaunchDarkly.WebhooksApi.GetWebhook(meta.Ctx, webhookId)
	if httpResponse != nil && httpResponse.StatusCode == 404 {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get webhook with id %q: %s", webhookId, handleLdapiErr(err))
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
