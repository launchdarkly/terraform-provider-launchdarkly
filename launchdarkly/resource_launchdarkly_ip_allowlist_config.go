package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const ipAllowlistConfigID = "ip-allowlist-config"

func resourceIpAllowlistConfig() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceIpAllowlistConfigCreate,
		ReadContext:   resourceIpAllowlistConfigRead,
		UpdateContext: resourceIpAllowlistConfigUpdate,
		DeleteContext: resourceIpAllowlistConfigDelete,
		Exists:        resourceIpAllowlistConfigExists,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			SESSION_ALLOWLIST_ENABLED: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether the session IP allowlist is enabled.",
			},
			SCOPED_ALLOWLIST_ENABLED: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether the scoped (API token) IP allowlist is enabled.",
			},
		},

		Description: `Provides a LaunchDarkly IP allowlist configuration resource.

This resource allows you to manage the IP allowlist configuration for your LaunchDarkly account. There is only one configuration per account, so only a single instance of this resource should be defined.`,
	}
}

func resourceIpAllowlistConfigCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)

	sessionEnabled := d.Get(SESSION_ALLOWLIST_ENABLED).(bool)
	scopedEnabled := d.Get(SCOPED_ALLOWLIST_ENABLED).(bool)

	_, err := patchIpAllowlistConfig(client, &sessionEnabled, &scopedEnabled)
	if err != nil {
		return diag.Errorf("failed to create IP allowlist config: %s", err)
	}

	d.SetId(ipAllowlistConfigID)

	return resourceIpAllowlistConfigRead(ctx, d, metaRaw)
}

func resourceIpAllowlistConfigRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := metaRaw.(*Client)

	allowlist, err := getIpAllowlist(client)
	if err != nil {
		return diag.Errorf("failed to read IP allowlist config: %s", err)
	}

	_ = d.Set(SESSION_ALLOWLIST_ENABLED, allowlist.SessionAllowlistEnabled)
	_ = d.Set(SCOPED_ALLOWLIST_ENABLED, allowlist.ApiTokenAllowlistEnabled)

	return diags
}

func resourceIpAllowlistConfigUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)

	sessionEnabled := d.Get(SESSION_ALLOWLIST_ENABLED).(bool)
	scopedEnabled := d.Get(SCOPED_ALLOWLIST_ENABLED).(bool)

	_, err := patchIpAllowlistConfig(client, &sessionEnabled, &scopedEnabled)
	if err != nil {
		return diag.Errorf("failed to update IP allowlist config: %s", err)
	}

	return resourceIpAllowlistConfigRead(ctx, d, metaRaw)
}

func resourceIpAllowlistConfigDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := metaRaw.(*Client)

	falseVal := false
	_, err := patchIpAllowlistConfig(client, &falseVal, &falseVal)
	if err != nil {
		return diag.Errorf("failed to reset IP allowlist config: %s", err)
	}

	return diags
}

func resourceIpAllowlistConfigExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)

	_, err := getIpAllowlist(client)
	if err != nil {
		return false, nil
	}

	return true, nil
}
