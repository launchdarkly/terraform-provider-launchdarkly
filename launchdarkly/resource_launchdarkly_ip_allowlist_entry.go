package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceIpAllowlistEntry() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceIpAllowlistEntryCreate,
		ReadContext:   resourceIpAllowlistEntryRead,
		UpdateContext: resourceIpAllowlistEntryUpdate,
		DeleteContext: resourceIpAllowlistEntryDelete,
		Exists:        resourceIpAllowlistEntryExists,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			IP_ADDRESS: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The IP address or CIDR block for the allowlist entry. Changing this forces a new resource to be created.",
			},
			DESCRIPTION: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A human-readable description of the IP allowlist entry.",
			},
		},

		Description: `Provides a LaunchDarkly IP allowlist entry resource.

This resource allows you to create and manage IP allowlist entries within your LaunchDarkly account.`,
	}
}

func resourceIpAllowlistEntryCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	ipAddress := d.Get(IP_ADDRESS).(string)

	var description *string
	if v, ok := d.GetOk(DESCRIPTION); ok {
		desc := v.(string)
		description = &desc
	}

	entry, err := createIpAllowlistEntry(client, ipAddress, description)
	if err != nil {
		return diag.Errorf("failed to create IP allowlist entry for %q: %s", ipAddress, err)
	}

	d.SetId(entry.ID)

	return resourceIpAllowlistEntryRead(ctx, d, metaRaw)
}

func resourceIpAllowlistEntryRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := metaRaw.(*Client)

	allowlist, err := getIpAllowlist(client)
	if err != nil {
		return diag.Errorf("failed to read IP allowlist: %s", err)
	}

	entry := findIpAllowlistEntryByID(allowlist.Entries, d.Id())
	if entry == nil {
		d.SetId("")
		return diags
	}

	_ = d.Set(IP_ADDRESS, entry.IpAddress)
	if entry.Description != nil {
		_ = d.Set(DESCRIPTION, *entry.Description)
	} else {
		_ = d.Set(DESCRIPTION, "")
	}

	return diags
}

func resourceIpAllowlistEntryUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)

	if d.HasChange(DESCRIPTION) {
		description := d.Get(DESCRIPTION).(string)
		_, err := patchIpAllowlistEntry(client, d.Id(), description)
		if err != nil {
			return diag.Errorf("failed to update IP allowlist entry %q: %s", d.Id(), err)
		}
	}

	return resourceIpAllowlistEntryRead(ctx, d, metaRaw)
}

func resourceIpAllowlistEntryDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := metaRaw.(*Client)

	err := deleteIpAllowlistEntry(client, d.Id())
	if err != nil {
		return diag.Errorf("failed to delete IP allowlist entry %q: %s", d.Id(), err)
	}

	return diags
}

func resourceIpAllowlistEntryExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)

	allowlist, err := getIpAllowlist(client)
	if err != nil {
		return false, fmt.Errorf("failed to check IP allowlist entry existence: %s", err)
	}

	return findIpAllowlistEntryByID(allowlist.Entries, d.Id()) != nil, nil
}
