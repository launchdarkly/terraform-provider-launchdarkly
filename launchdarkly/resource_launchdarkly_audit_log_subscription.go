package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v12"
)

func resourceAuditLogSubscription() *schema.Resource {
	return &schema.Resource{
		Description: `Provides a LaunchDarkly audit log subscription resource.

This resource allows you to create and manage LaunchDarkly audit log subscriptions.`,
		CreateContext: resourceAuditLogSubscriptionCreate,
		UpdateContext: resourceAuditLogSubscriptionUpdate,
		DeleteContext: resourceAuditLogSubscriptionDelete,
		ReadContext:   resourceAuditLogSubscriptionRead,
		Exists:        resourceAuditLogSubscriptionExists,

		Importer: &schema.ResourceImporter{
			State: resourceAuditLogSubscriptionImport,
		},

		Schema: auditLogSubscriptionSchema(false),
	}
}

func resourceAuditLogSubscriptionCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	integrationKey := d.Get(INTEGRATION_KEY).(string)
	name := d.Get(NAME).(string)
	on := d.Get(ON).(bool)
	tags := stringsFromSchemaSet(d.Get(TAGS).(*schema.Set))
	config, err := configFromResourceData(d)
	if err != nil {
		return diag.Errorf("failed to create %s integration with name %s: %v", integrationKey, name, err.Error())
	}

	statements, err := policyStatementsFromResourceData(d.Get(STATEMENTS).([]interface{}))
	if err != nil {
		return diag.Errorf("failed to create %s integration with name %s: %v", integrationKey, name, err.Error())
	}

	subscriptionBody := ldapi.SubscriptionPost{
		Name:       name,
		On:         &on,
		Tags:       tags,
		Config:     config,
		Statements: statements,
	}

	sub, _, err := client.ld.IntegrationAuditLogSubscriptionsApi.CreateSubscription(client.ctx, integrationKey).SubscriptionPost(subscriptionBody).Execute()

	if err != nil {
		return diag.Errorf("failed to create %s integration with name %s: %v", integrationKey, name, handleLdapiErr(err))
	}
	d.SetId(*sub.Id)
	return resourceAuditLogSubscriptionRead(ctx, d, metaRaw)
}

func resourceAuditLogSubscriptionUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	integrationKey := d.Get(INTEGRATION_KEY).(string)
	name := d.Get(NAME).(string)
	tags := stringsFromResourceData(d, TAGS)
	on := d.Get(ON).(bool)
	config, err := configFromResourceData(d)
	if err != nil {
		return diag.FromErr(err)
	}
	id := d.Id()

	statements, err := policyStatementsFromResourceData(d.Get(STATEMENTS).([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}

	patch := []ldapi.PatchOperation{
		patchReplace("/name", &name),
		patchReplace("/tags", &tags),
		patchReplace("/config", &config),
		patchReplace("/on", &on),
		patchReplace("/statements", &statements),
	}

	_, _, err = client.ld.IntegrationAuditLogSubscriptionsApi.UpdateSubscription(client.ctx, integrationKey, id).PatchOperation(patch).Execute()
	if err != nil {
		return diag.Errorf("failed to update %q integration with name %q and ID %q: %s", integrationKey, name, id, handleLdapiErr(err))
	}
	return resourceAuditLogSubscriptionRead(ctx, d, metaRaw)
}

func resourceAuditLogSubscriptionDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	id := d.Id()
	integrationKey := d.Get(INTEGRATION_KEY).(string)

	_, err := client.ld.IntegrationAuditLogSubscriptionsApi.DeleteSubscription(client.ctx, integrationKey, id).Execute()

	if err != nil {
		return diag.Errorf("failed to delete integration with ID %q: %s", id, handleLdapiErr(err))
	}
	return diag.Diagnostics{}
}

func resourceAuditLogSubscriptionRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return auditLogSubscriptionRead(ctx, d, metaRaw, false)
}

func resourceAuditLogSubscriptionExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	id := d.Id()
	integrationKey := d.Get(INTEGRATION_KEY).(string)

	_, res, err := client.ld.IntegrationAuditLogSubscriptionsApi.GetSubscriptionByID(client.ctx, integrationKey, id).Execute()
	if isStatusNotFound(res) {
		log.Println("got 404 when getting integration. returning false.")
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get integration with ID %q: %v", id, handleLdapiErr(err))
	}
	return true, nil
}

func resourceAuditLogSubscriptionImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()
	parts := strings.Split(d.Id(), "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("found unexpected id format for import: %q. expected format: 'integrationKey/integration_id'", id)
	}

	integrationKey, integrationID := parts[0], parts[1]
	_ = d.Set(INTEGRATION_KEY, integrationKey)
	d.SetId(integrationID)
	return []*schema.ResourceData{d}, nil
}
