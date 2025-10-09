package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceReleasePolicyOrder() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceReleasePolicyOrderCreate,
		ReadContext:   resourceReleasePolicyOrderRead,
		UpdateContext: resourceReleasePolicyOrderUpdate,
		DeleteContext: resourceReleasePolicyOrderDelete,
		Exists:        resourceReleasePolicyOrderExists,

		Importer: &schema.ResourceImporter{
			StateContext: resourceReleasePolicyOrderImport,
		},

		Description: `Provides a LaunchDarkly release policy order resource. This resource is still in beta.

This resource allows you to manage the priority of release policies within LaunchDarkly projects.

Learn more about [release policies here](https://launchdarkly.com/docs/home/releases/release-policies), and read our [API docs here](https://launchdarkly.com/docs/api/release-policies-beta/).`,

		Schema: map[string]*schema.Schema{
			PROJECT_KEY: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      addForceNewDescription("The project key.", true),
				ForceNew:         true,
				ValidateDiagFunc: validateKeyAndLength(1, 140),
			},
			RELEASE_POLICY_KEYS: {
				Type:        schema.TypeList,
				Required:    true,
				Description: "An ordered list of release policy keys that defines the order of release policies within the project.",
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validateKeyAndLength(1, 256),
				},
			},
		},
	}
}

func resourceReleasePolicyOrderCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	policyKeys := d.Get(RELEASE_POLICY_KEYS).([]interface{})

	// Convert interface slice to string slice
	releasePolicyKeys := make([]string, len(policyKeys))
	for i, key := range policyKeys {
		releasePolicyKeys[i] = key.(string)
	}

	err := updateReleasePolicyOrder(client, projectKey, releasePolicyKeys)
	if err != nil {
		return diag.Errorf("failed to create release policy order for project %q: %s", projectKey, handleLdapiErr(err))
	}

	d.SetId(projectKey)

	return resourceReleasePolicyOrderRead(ctx, d, metaRaw)
}

func resourceReleasePolicyOrderRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := metaRaw.(*Client)

	projectKey := d.Get(PROJECT_KEY).(string)

	order, err := getReleasePolicyOrder(client, projectKey)
	if err != nil {
		return diag.Errorf("failed to get release policy order for project %q: %s", projectKey, handleLdapiErr(err))
	}

	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(RELEASE_POLICY_KEYS, order)

	return diags
}

func resourceReleasePolicyOrderUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)

	// If the project key changes, treat it like a new resource
	if d.HasChange(PROJECT_KEY) {
		return resourceReleasePolicyOrderCreate(ctx, d, metaRaw)
	}

	if d.HasChange(RELEASE_POLICY_KEYS) {
		policyKeys := d.Get(RELEASE_POLICY_KEYS).([]interface{})

		// Convert interface slice to string slice
		releasePolicyKeys := make([]string, len(policyKeys))
		for i, key := range policyKeys {
			releasePolicyKeys[i] = key.(string)
		}

		err := updateReleasePolicyOrder(client, projectKey, releasePolicyKeys)
		if err != nil {
			return diag.Errorf("failed to update release policy order for project %q: %s", projectKey, handleLdapiErr(err))
		}
	}

	return resourceReleasePolicyOrderRead(ctx, d, metaRaw)
}

func resourceReleasePolicyOrderDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	// Just remove from state, don't actually delete anything
	d.SetId("")
	return diags
}

func resourceReleasePolicyOrderExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)

	_, err := getReleasePolicyOrder(client, projectKey)
	if err != nil {
		return false, fmt.Errorf("failed to get release policy order for project %q: %s", projectKey, handleLdapiErr(err))
	}

	return true, nil
}

func resourceReleasePolicyOrderImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	projectKey := d.Id()
	if projectKey == "" {
		return nil, fmt.Errorf("import ID cannot be empty")
	}
	_ = d.Set(PROJECT_KEY, projectKey)
	d.SetId(projectKey)

	return []*schema.ResourceData{d}, nil
}
