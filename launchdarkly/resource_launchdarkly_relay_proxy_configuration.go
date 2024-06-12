package launchdarkly

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v16"
)

func resourceRelayProxyConfig() *schema.Resource {
	return &schema.Resource{
		CreateContext: relayProxyConfigCreate,
		ReadContext:   resourceRelayProxyConfigRead,
		UpdateContext: relayProxyConfigUpdate,
		DeleteContext: relayProxyConfigDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			NAME: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The human-readable name for your Relay Proxy configuration.",
			},
			POLICY: policyStatementsSchema(policyStatementSchemaOptions{
				required:    true,
				description: "The Relay Proxy configuration's rule policy block. This determines what content the Relay Proxy receives. To learn more, read [Understanding policies](https://docs.launchdarkly.com/home/members/role-policies#understanding-policies).",
			}),
			FULL_KEY: {
				Type:        schema.TypeString,
				Sensitive:   true,
				Computed:    true,
				Description: "The Relay Proxy configuration's unique key. Because the `full_key` is only exposed upon creation, it will not be available if the resource is imported.",
			},
			DISPLAY_KEY: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The last 4 characters of the Relay Proxy configuration's unique key.",
			},
		},
		Description: `Provides a LaunchDarkly Relay Proxy configuration resource for use with the Relay Proxy's [automatic configuration feature](https://docs.launchdarkly.com/home/relay-proxy/automatic-configuration).

-> **Note:** Relay Proxy automatic configuration is available to customers on an Enterprise LaunchDarkly plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).

This resource allows you to create and manage Relay Proxy configurations within your LaunchDarkly organization.

-> **Note:** This resource will store the full plaintext secret for your Relay Proxy configuration's unique key in Terraform state. Be sure your state is configured securely before using this resource. See https://www.terraform.io/docs/state/sensitive-data.html for more details.`,
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

	return resourceRelayProxyConfigRead(ctx, d, m)
}

func resourceRelayProxyConfigRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return relayProxyConfigRead(ctx, d, m, false)
}

func relayProxyConfigRead(ctx context.Context, d *schema.ResourceData, m interface{}, isDataSource bool) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*Client)

	id := d.Id()
	proxyConfig, res, err := client.ld.RelayProxyConfigurationsApi.GetRelayProxyConfig(client.ctx, id).Execute()
	if isStatusNotFound(res) {
		if isDataSource {
			return diag.Errorf("Relay Proxy configuration with id %q not found.", id)
		}
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
