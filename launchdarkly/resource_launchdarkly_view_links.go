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

This resource allows you to efficiently link multiple flags and/or segments to a specific view. This is particularly useful for administrators organizing resources by team or deployment unit.

-> **Note:** This resource manages ALL links for the specified resource types within a view. Adding or removing items from the configuration will link or unlink those resources accordingly.

## Alternative Approach: view_keys on Individual Resources

For modular Terraform configurations where flags and segments are defined in separate files or modules, you can use the ` + "`view_keys`" + ` field directly on the resource instead of using this centralized ` + "`view_links`" + ` resource:

- **Feature Flags**: Use the ` + "`view_keys`" + ` attribute on ` + "`launchdarkly_feature_flag`" + ` resources
- **Segments**: Use the ` + "`view_keys`" + ` attribute on ` + "`launchdarkly_segment`" + ` resources

**When to use ` + "`view_links`" + ` (this resource):**
- Managing many flags/segments for a single view (bulk operations)
- Centralized view management across your infrastructure
- Administrative view organization

**When to use ` + "`view_keys`" + ` on individual resources:**
- Modular Terraform structures with separate files per flag/segment
- Each team/module manages their own resources
- Want view membership defined alongside the resource

-> **Warning:** Avoid using both ` + "`view_links`" + ` and ` + "`view_keys`" + ` to manage the same flag or segment's view associations, as this may cause conflicts.

See the feature flag resource documentation and segment resource documentation for details on the ` + "`view_keys`" + ` attribute.`,

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
			SEGMENTS: {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "A set of segments to link to the view. Each segment is identified by its environment ID and segment key.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						SEGMENT_ENVIRONMENT_ID: {
							Type:             schema.TypeString,
							Required:         true,
							Description:      "The environment ID of the segment.",
							ValidateDiagFunc: validateID(),
						},
						SEGMENT_KEY: {
							Type:             schema.TypeString,
							Required:         true,
							Description:      "The key of the segment.",
							ValidateDiagFunc: validateKey(),
						},
					},
				},
			},
		},
	}
}

func resourceViewLinksCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		return diag.FromErr(err)
	}

	projectKey := d.Get(PROJECT_KEY).(string)
	viewKey := d.Get(VIEW_KEY).(string)

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
		err = linkResourcesToView(betaClient, projectKey, viewKey, FLAGS, flags)
		if err != nil {
			return diag.Errorf("failed to link flags to view %q in project %q: %s", viewKey, projectKey, err)
		}
	}

	// Link segments if specified
	if segmentsRaw, ok := d.GetOk(SEGMENTS); ok {
		segments := segmentsRaw.(*schema.Set).List()
		segmentIdentifiers := make([]ViewSegmentIdentifier, len(segments))
		for i, seg := range segments {
			segMap := seg.(map[string]interface{})
			segmentIdentifiers[i] = ViewSegmentIdentifier{
				EnvironmentId: segMap[SEGMENT_ENVIRONMENT_ID].(string),
				SegmentKey:    segMap[SEGMENT_KEY].(string),
			}
		}
		err = linkSegmentsToView(betaClient, projectKey, viewKey, segmentIdentifiers)
		if err != nil {
			return diag.Errorf("failed to link segments to view %q in project %q: %s", viewKey, projectKey, err)
		}
	}

	d.SetId(fmt.Sprintf("%s/%s", projectKey, viewKey))
	return resourceViewLinksRead(ctx, d, metaRaw)
}

func resourceViewLinksRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := metaRaw.(*Client)
	betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
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

	// Get currently linked segments
	linkedSegments, err := getLinkedResources(betaClient, projectKey, viewKey, SEGMENTS)
	if err != nil {
		return diag.Errorf("failed to get linked segments for view %q in project %q: %s", viewKey, projectKey, err)
	}

	segments := make([]map[string]interface{}, len(linkedSegments))
	for i, segment := range linkedSegments {
		segments[i] = map[string]interface{}{
			SEGMENT_ENVIRONMENT_ID: segment.EnvironmentId,
			SEGMENT_KEY:            segment.ResourceKey,
		}
	}

	err = d.Set(SEGMENTS, segments)
	if err != nil {
		return diag.Errorf("failed to set segments for view %q in project %q: %s", viewKey, projectKey, err)
	}

	return diags
}

func resourceViewLinksUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		return diag.FromErr(err)
	}

	projectKey := d.Get(PROJECT_KEY).(string)
	viewKey := d.Get(VIEW_KEY).(string)

	if d.HasChange(FLAGS) {
		oldFlagsRaw, newFlagsRaw := d.GetChange(FLAGS)
		oldFlags := interfaceSliceToStringSlice(oldFlagsRaw.(*schema.Set).List())
		newFlags := interfaceSliceToStringSlice(newFlagsRaw.(*schema.Set).List())

		// Calculate flags to add and remove
		flagsToAdd := difference(newFlags, oldFlags)
		flagsToRemove := difference(oldFlags, newFlags)

		// Remove flags that are no longer in the list
		if len(flagsToRemove) > 0 {
			err = unlinkResourcesFromView(betaClient, projectKey, viewKey, FLAGS, flagsToRemove)
			if err != nil {
				return diag.Errorf("failed to unlink flags from view %q in project %q: %s", viewKey, projectKey, err)
			}
		}

		// Add new flags
		if len(flagsToAdd) > 0 {
			err = linkResourcesToView(betaClient, projectKey, viewKey, FLAGS, flagsToAdd)
			if err != nil {
				return diag.Errorf("failed to link flags to view %q in project %q: %s", viewKey, projectKey, err)
			}
		}
	}

	if d.HasChange(SEGMENTS) {
		oldSegmentsRaw, newSegmentsRaw := d.GetChange(SEGMENTS)
		oldSegments := oldSegmentsRaw.(*schema.Set).List()
		newSegments := newSegmentsRaw.(*schema.Set).List()

		// Convert to segment identifiers
		oldSegmentIdentifiers := make([]ViewSegmentIdentifier, len(oldSegments))
		for i, seg := range oldSegments {
			segMap := seg.(map[string]interface{})
			oldSegmentIdentifiers[i] = ViewSegmentIdentifier{
				EnvironmentId: segMap[SEGMENT_ENVIRONMENT_ID].(string),
				SegmentKey:    segMap[SEGMENT_KEY].(string),
			}
		}

		newSegmentIdentifiers := make([]ViewSegmentIdentifier, len(newSegments))
		for i, seg := range newSegments {
			segMap := seg.(map[string]interface{})
			newSegmentIdentifiers[i] = ViewSegmentIdentifier{
				EnvironmentId: segMap[SEGMENT_ENVIRONMENT_ID].(string),
				SegmentKey:    segMap[SEGMENT_KEY].(string),
			}
		}

		// Calculate segments to add and remove
		segmentsToAdd := differenceSegmentIdentifiers(newSegmentIdentifiers, oldSegmentIdentifiers)
		segmentsToRemove := differenceSegmentIdentifiers(oldSegmentIdentifiers, newSegmentIdentifiers)

		// Remove segments that are no longer in the list
		if len(segmentsToRemove) > 0 {
			err = unlinkSegmentsFromView(betaClient, projectKey, viewKey, segmentsToRemove)
			if err != nil {
				return diag.Errorf("failed to unlink segments from view %q in project %q: %s", viewKey, projectKey, err)
			}
		}

		// Add new segments
		if len(segmentsToAdd) > 0 {
			err = linkSegmentsToView(betaClient, projectKey, viewKey, segmentsToAdd)
			if err != nil {
				return diag.Errorf("failed to link segments to view %q in project %q: %s", viewKey, projectKey, err)
			}
		}
	}

	return resourceViewLinksRead(ctx, d, metaRaw)
}

func resourceViewLinksDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := metaRaw.(*Client)
	betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		return diag.FromErr(err)
	}

	projectKey := d.Get(PROJECT_KEY).(string)
	viewKey := d.Get(VIEW_KEY).(string)

	// Unlink all flags
	if flagsRaw, ok := d.GetOk(FLAGS); ok {
		flags := interfaceSliceToStringSlice(flagsRaw.(*schema.Set).List())
		if len(flags) > 0 {
			err = unlinkResourcesFromView(betaClient, projectKey, viewKey, FLAGS, flags)
			if err != nil {
				return diag.Errorf("failed to unlink flags from view %q in project %q: %s", viewKey, projectKey, err)
			}
		}
	}

	// Unlink all segments
	if segmentsRaw, ok := d.GetOk(SEGMENTS); ok {
		segments := segmentsRaw.(*schema.Set).List()
		if len(segments) > 0 {
			segmentIdentifiers := make([]ViewSegmentIdentifier, len(segments))
			for i, seg := range segments {
				segMap := seg.(map[string]interface{})
				segmentIdentifiers[i] = ViewSegmentIdentifier{
					EnvironmentId: segMap[SEGMENT_ENVIRONMENT_ID].(string),
					SegmentKey:    segMap[SEGMENT_KEY].(string),
				}
			}
			err = unlinkSegmentsFromView(betaClient, projectKey, viewKey, segmentIdentifiers)
			if err != nil {
				return diag.Errorf("failed to unlink segments from view %q in project %q: %s", viewKey, projectKey, err)
			}
		}
	}

	return diags
}

func resourceViewLinksExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
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

// Helper function to calculate the difference between two segment identifier slices
func differenceSegmentIdentifiers(slice1, slice2 []ViewSegmentIdentifier) []ViewSegmentIdentifier {
	set2 := make(map[string]bool)
	for _, item := range slice2 {
		key := item.EnvironmentId + ":" + item.SegmentKey
		set2[key] = true
	}

	var diff []ViewSegmentIdentifier
	for _, item := range slice1 {
		key := item.EnvironmentId + ":" + item.SegmentKey
		if !set2[key] {
			diff = append(diff, item)
		}
	}
	return diff
}
