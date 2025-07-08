package launchdarkly

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceViewLinks() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceViewLinksCreate,
		ReadContext:   resourceViewLinksRead,
		UpdateContext: resourceViewLinksUpdate,
		DeleteContext: resourceViewLinksDelete,
		Exists:        resourceViewLinksExists,

		Importer: &schema.ResourceImporter{
			StateContext: resourceViewLinksImport,
		},

		Description: `Provides a LaunchDarkly view links resource for managing bulk resource linkage to views.

This resource allows you to efficiently link multiple flags (and in the future, segments and metrics) to a specific view. This is particularly useful for administrators organizing resources by team or deployment unit.

-> **Note:** This resource manages ALL links for the specified resource types within a view. Adding or removing items from the configuration will link or unlink those resources accordingly.`,

		Schema: map[string]*schema.Schema{
			PROJECT_KEY: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      addForceNewDescription("The project key.", true),
				ValidateDiagFunc: validateKey(),
			},
			VIEW_KEY: {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      addForceNewDescription("The view key to link resources to.", true),
				ValidateDiagFunc: validateKey(),
			},
			FLAGS: {
				Type:        schema.TypeSet,
				Set:         schema.HashString,
				Optional:    true,
				Description: "A set of feature flag keys to link to the view.",
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validateKey(),
				},
			},
			// Future extensibility - commented out for now
			// "segments": {
			// 	Type:        schema.TypeSet,
			// 	Optional:    true,
			// 	Description: "A set of segment keys to link to the view.",
			// 	Elem: &schema.Schema{
			// 		Type:             schema.TypeString,
			// 		ValidateDiagFunc: validateKey(),
			// 	},
			// },
			// "metrics": {
			// 	Type:        schema.TypeSet,
			// 	Optional:    true,
			// 	Description: "A set of metric keys to link to the view.",
			// 	Elem: &schema.Schema{
			// 		Type:             schema.TypeString,
			// 		ValidateDiagFunc: validateKey(),
			// 	},
			// },
			"comment": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Optional comment for the link operations.",
				Default:     "Managed by Terraform",
			},
		},
	}
}

func resourceViewLinksCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S)
	if err != nil {
		return diag.FromErr(err)
	}

	projectKey := d.Get(PROJECT_KEY).(string)
	viewKey := d.Get(VIEW_KEY).(string)
	comment := d.Get("comment").(string)

	// Check if view exists
	if exists, err := viewExists(projectKey, viewKey, betaClient); !exists {
		if err != nil {
			return diag.FromErr(err)
		}
		return diag.Errorf("cannot find view with key %q in project %q", viewKey, projectKey)
	}

	// Link flags if specified
	if flagsRaw, ok := d.GetOk(FLAGS); ok {
		flags := interfaceSliceToStringSlice(flagsRaw.(*schema.Set).List())
		if len(flags) > 0 {
			err = linkResourcesToView(betaClient, projectKey, viewKey, FLAGS, flags, comment)
			if err != nil {
				return diag.Errorf("failed to link flags to view %q in project %q: %s", viewKey, projectKey, err)
			}
		}
	}

	d.SetId(fmt.Sprintf("%s/%s", projectKey, viewKey))
	return resourceViewLinksRead(ctx, d, metaRaw)
}

func resourceViewLinksRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := metaRaw.(*Client)
	betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S)
	if err != nil {
		return diag.FromErr(err)
	}

	projectKey := d.Get(PROJECT_KEY).(string)
	viewKey := d.Get(VIEW_KEY).(string)

	// Check if view still exists
	if exists, err := viewExists(projectKey, viewKey, betaClient); !exists {
		if err != nil {
			return diag.FromErr(err)
		}
		log.Printf("[WARN] view with key %q in project %q not found, removing from state", viewKey, projectKey)
		d.SetId("")
		return diags
	}

	// Get currently linked flags
	linkedFlags, err := getLinkedResources(betaClient, projectKey, viewKey, FLAGS)
	if err != nil {
		return diag.Errorf("failed to get linked flags for view %q in project %q: %s", viewKey, projectKey, err)
	}

	flagKeys := make([]string, len(linkedFlags))
	for i, flag := range linkedFlags {
		flagKeys[i] = flag.ResourceKey
	}

	err = d.Set(FLAGS, flagKeys)
	if err != nil {
		return diag.Errorf("failed to set flags for view %q in project %q: %s", viewKey, projectKey, err)
	}

	return diags
}

func resourceViewLinksUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S)
	if err != nil {
		return diag.FromErr(err)
	}

	projectKey := d.Get(PROJECT_KEY).(string)
	viewKey := d.Get(VIEW_KEY).(string)
	comment := d.Get("comment").(string)

	if d.HasChange(FLAGS) {
		oldFlagsRaw, newFlagsRaw := d.GetChange(FLAGS)
		oldFlags := interfaceSliceToStringSlice(oldFlagsRaw.(*schema.Set).List())
		newFlags := interfaceSliceToStringSlice(newFlagsRaw.(*schema.Set).List())

		// Calculate flags to add and remove
		flagsToAdd := difference(newFlags, oldFlags)
		flagsToRemove := difference(oldFlags, newFlags)

		// Remove flags that are no longer in the list
		if len(flagsToRemove) > 0 {
			err = unlinkResourcesFromView(betaClient, projectKey, viewKey, FLAGS, flagsToRemove, comment)
			if err != nil {
				return diag.Errorf("failed to unlink flags from view %q in project %q: %s", viewKey, projectKey, err)
			}
		}

		// Add new flags
		if len(flagsToAdd) > 0 {
			err = linkResourcesToView(betaClient, projectKey, viewKey, FLAGS, flagsToAdd, comment)
			if err != nil {
				return diag.Errorf("failed to link flags to view %q in project %q: %s", viewKey, projectKey, err)
			}
		}
	}

	return resourceViewLinksRead(ctx, d, metaRaw)
}

func resourceViewLinksDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := metaRaw.(*Client)
	betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S)
	if err != nil {
		return diag.FromErr(err)
	}

	projectKey := d.Get(PROJECT_KEY).(string)
	viewKey := d.Get(VIEW_KEY).(string)
	comment := d.Get("comment").(string)

	// Unlink all flags
	if flagsRaw, ok := d.GetOk(FLAGS); ok {
		flags := interfaceSliceToStringSlice(flagsRaw.(*schema.Set).List())
		if len(flags) > 0 {
			err = unlinkResourcesFromView(betaClient, projectKey, viewKey, FLAGS, flags, comment)
			if err != nil {
				return diag.Errorf("failed to unlink flags from view %q in project %q: %s", viewKey, projectKey, err)
			}
		}
	}

	return diags
}

func resourceViewLinksExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S)
	if err != nil {
		return false, err
	}
	projectKey := d.Get(PROJECT_KEY).(string)
	viewKey := d.Get(VIEW_KEY).(string)
	return viewExists(projectKey, viewKey, betaClient)
}

func resourceViewLinksImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	projectKey, viewKey, err := viewIdToKeys(d.Id())
	if err != nil {
		return nil, err
	}

	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(VIEW_KEY, viewKey)

	return []*schema.ResourceData{d}, nil
}

// Helper function to calculate the difference between two string slices
func difference(slice1, slice2 []string) []string {
	set2 := make(map[string]bool)
	for _, item := range slice2 {
		set2[item] = true
	}

	var diff []string
	for _, item := range slice1 {
		if !set2[item] {
			diff = append(diff, item)
		}
	}
	return diff
}
