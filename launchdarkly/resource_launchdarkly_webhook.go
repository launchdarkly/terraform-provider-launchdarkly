package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v12"
)

func resourceWebhook() *schema.Resource {
	schemaMap := baseWebhookSchema(webhookSchemaOptions{isDataSource: false})
	schemaMap[URL] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The URL of the remote webhook",
	}
	schemaMap[ON] = &schema.Schema{
		Type:        schema.TypeBool,
		Description: "Whether this webhook is enabled or not",
		Optional:    true,
		Default:     false,
	}
	return &schema.Resource{
		CreateContext: resourceWebhookCreate,
		ReadContext:   resourceWebhookRead,
		UpdateContext: resourceWebhookUpdate,
		DeleteContext: resourceWebhookDelete,
		Exists:        resourceWebhookExists,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: schemaMap,
	}
}

func resourceWebhookCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	webhookURL := d.Get(URL).(string)
	webhookSecret := d.Get(SECRET).(string)
	webhookName := d.Get(NAME).(string)

	webhookOn := d.Get(ON).(bool)

	webhookBody := ldapi.WebhookPost{
		Url:  webhookURL,
		On:   webhookOn,
		Name: &webhookName,
	}

	if rawStatements, ok := d.GetOk(STATEMENTS); ok {
		statements, err := policyStatementsFromResourceData(rawStatements.([]interface{}))
		if err != nil {
			return diag.FromErr(err)
		}
		webhookBody.Statements = statements
	}

	// The sign field isn't returned when GETting a webhook so terraform can't import it properly.
	// We hide the field from terraform to avoid import problems.
	if webhookSecret != "" {
		webhookBody.Secret = &webhookSecret
		webhookBody.Sign = true
	}

	webhook, _, err := client.ld.WebhooksApi.PostWebhook(client.ctx).WebhookPost(webhookBody).Execute()
	if err != nil {
		return diag.Errorf("failed to create webhook with name %q: %s", webhookName, handleLdapiErr(err))
	}

	d.SetId(webhook.Id)

	// ld's api does not allow tags to be passed in during webhook creation so we do an update
	updateDiags := resourceWebhookUpdate(ctx, d, metaRaw)
	if updateDiags.HasError() {
		updateDiags = append(updateDiags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("error updating after webhook creation. Webhook name: %q", webhookName),
		})
		return updateDiags
	}

	return resourceWebhookRead(ctx, d, metaRaw)
}

func resourceWebhookRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return webhookRead(ctx, d, metaRaw, false)
}

func resourceWebhookUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	webhookID := d.Id()
	webhookURL := d.Get(URL).(string)
	webhookSecret := d.Get(SECRET).(string)
	webhookName := d.Get(NAME).(string)
	webhookTags := stringsFromResourceData(d, TAGS)
	webhookOn := d.Get(ON).(bool)

	patch := []ldapi.PatchOperation{
		patchReplace("/url", &webhookURL),
		patchReplace("/secret", &webhookSecret),
		patchReplace("/on", &webhookOn),
		patchReplace("/name", &webhookName),
		patchReplace("/tags", &webhookTags),
	}

	statements, err := policyStatementsFromResourceData(d.Get(STATEMENTS).([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange(STATEMENTS) {
		if len(statements) > 0 {
			patch = append(patch, patchReplace("/statements", &statements))
		} else {
			patch = append(patch, patchRemove("/statements"))
		}
	}

	_, _, err = client.ld.WebhooksApi.PatchWebhook(client.ctx, webhookID).PatchOperation(patch).Execute()
	if err != nil {
		return diag.Errorf("failed to update webhook with id %q: %s", webhookID, handleLdapiErr(err))
	}

	return resourceWebhookRead(ctx, d, metaRaw)
}

func resourceWebhookDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)
	webhookID := d.Id()

	_, err := client.ld.WebhooksApi.DeleteWebhook(client.ctx, webhookID).Execute()
	if err != nil {
		return diag.Errorf("failed to delete webhook with id %q: %s", webhookID, handleLdapiErr(err))
	}

	return diags
}

func resourceWebhookExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return webhookExists(d.Id(), metaRaw.(*Client))
}

func webhookExists(webhookID string, meta *Client) (bool, error) {
	_, res, err := meta.ld.WebhooksApi.GetWebhook(meta.ctx, webhookID).Execute()
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get webhook with id %q: %s", webhookID, handleLdapiErr(err))
	}

	return true, nil
}
