package launchdarkly

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v7"
)

func resourceRelayProxyConfig() *schema.Resource {
	return &schema.Resource{
		CreateContext: relayProxyConfigCreate,
		ReadContext:   relayProxyConfigRead,
		UpdateContext: relayProxyConfigUpdate,
		DeleteContext: relayProxyConfigDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			NAME: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "A human-friendly name for the Relay Proxy configuration",
			},
			POLICY: policyStatementsSchema(policyStatementSchemaOptions{required: true}),
			FULL_KEY: {
				Type:        schema.TypeString,
				Sensitive:   true,
				Computed:    true,
				Description: "The unique key assigned to the Relay Proxy configuration during creation.",
			},
			DISPLAY_KEY: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The last four characters of the full_key.",
			},
		},
	}
}

func relayProxyConfigCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*Client)

	name := d.Get(NAME).(string)
	policy, err := policyStatementsFromResourceData(d.Get(POLICY).([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}
	post := ldapi.RelayAutoConfigPost{
		Name:   name,
		Policy: statementPostsToStatementReps(policy),
	}

	proxyConfig, _, err := client.ld.RelayProxyConfigurationsApi.PostRelayAutoConfig(client.ctx).RelayAutoConfigPost(post).Execute()
	if err != nil {
		return diag.Errorf("failed to create Relay Proxy configuration with name %q: %s", name, handleLdapiErr(err))
	}

	d.SetId(proxyConfig.Id)

	// We only have the valid FULL_KEY immediately after creation.
	err = d.Set(FULL_KEY, proxyConfig.FullKey)
	if err != nil {
		return diag.FromErr(err)
	}

	return relayProxyConfigRead(ctx, d, m)
}

func relayProxyConfigRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Client)

	id := d.Id()
	proxyConfig, res, err := client.ld.RelayProxyConfigurationsApi.GetRelayProxyConfig(client.ctx, id).Execute()
	if isStatusNotFound(res) {
		log.Printf("[DEBUG] Relay Proxy configuration with id %q not found on LaunchDarkly. Removing from state", id)
		d.SetId("")
		return diags
	}
	if err != nil {
		return diag.Errorf("failed to get Relay Proxy configuration with id %q", id)
	}
	d.SetId(proxyConfig.Id)

	err = d.Set(NAME, proxyConfig.Name)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set(POLICY, policyStatementsToResourceData(proxyConfig.Policy))
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set(DISPLAY_KEY, proxyConfig.DisplayKey)
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func relayProxyConfigUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Client)

	id := d.Id()
	name := d.Get(NAME).(string)
	policy, err := policyStatementsFromResourceData(d.Get(POLICY).([]interface{}))
	if err != nil {
		return diag.FromErr(err)
	}

	patch := []ldapi.PatchOperation{
		patchReplace("/name", &name),
		patchReplace("/policy", &policy),
	}

	patchWithComment := ldapi.PatchWithComment{
		Patch:   patch,
		Comment: ldapi.PtrString("Terraform"),
	}

	_, _, err = client.ld.RelayProxyConfigurationsApi.PatchRelayAutoConfig(client.ctx, id).PatchWithComment(patchWithComment).Execute()
	if err != nil {
		return diag.Errorf("failed to update relay proxy configuration with id: %q: %s", id, handleLdapiErr(err))
	}

	return diags
}

func relayProxyConfigDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Client)

	id := d.Id()
	_, err := client.ld.RelayProxyConfigurationsApi.DeleteRelayAutoConfig(client.ctx, id).Execute()
	if err != nil {
		return diag.Errorf("failed to delete relay proxy configuration with id: %q: %s", id, handleLdapiErr(err))
	}

	return diags
}
