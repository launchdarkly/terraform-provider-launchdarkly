package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceViewFilterLinks() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceViewFilterLinksCreate,
		ReadContext:   resourceViewFilterLinksRead,
		UpdateContext: resourceViewFilterLinksUpdate,
		DeleteContext: resourceViewFilterLinksDelete,
		Exists:        resourceViewFilterLinksExists,

		Importer: &schema.ResourceImporter{
			StateContext: resourceViewFilterLinksImport,
		},

		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
			// Always re-resolve filters on every apply to catch changes
			// in the matching resource set (e.g. tag additions/removals).
			if diff.Id() != "" {
				if err := diff.SetNewComputed(RESOLVED_AT); err != nil {
					return err
				}
			}
			return nil
		},

		Description: `Provides a LaunchDarkly view filter links resource for linking resources to views using filter expressions.

This resource allows you to link all flags and/or segments matching a filter expression to a specific view. The filter is resolved at apply time — the backend finds all resources matching the filter and links them to the view.

-> **Note:** Filter-based links are point-in-time. The filter is resolved when ` + "`terraform apply`" + ` runs. Resources created or tagged after the apply will not be automatically linked. Run ` + "`terraform apply`" + ` again to pick up new matches.

## When to use which resource

- **` + "`view_links`" + `**: You know the exact flag/segment keys to link. Terraform tracks the explicit list and detects drift if links are removed externally.
- **` + "`view_filter_links`" + ` (this resource)**: You want to link all resources matching a dynamic query (e.g. all flags tagged "frontend"). No drift detection on resolved keys — only changes to the filter string itself trigger updates.
- **` + "`view_keys`" + ` on individual resources**: Each flag/segment declares its own view membership. Best for modular Terraform structures.

-> **Warning:** Do not use ` + "`view_filter_links`" + ` and ` + "`view_links`" + ` targeting the same view and resource type, as conflicts may cause unexpected behavior.`,

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
			FLAG_FILTER: {
				Type:         schema.TypeString,
				Optional:     true,
				AtLeastOneOf: []string{FLAG_FILTER, SEGMENT_FILTER},
				Description:  "A filter expression to match feature flags for linking to the view. Uses the same filter syntax as the flag list API endpoint (e.g. `tags:frontend`, `status:active`).",
			},
			SEGMENT_FILTER: {
				Type:         schema.TypeString,
				Optional:     true,
				AtLeastOneOf: []string{FLAG_FILTER, SEGMENT_FILTER},
				RequiredWith: []string{SEGMENT_FILTER_ENVIRONMENT_ID},
				Description:  "A filter expression to match segments for linking to the view. Uses the segment query filter syntax (e.g. `tags anyOf [\"backend\"]`, `query = \"my-segment\"`, `unbounded = true`). Requires `segment_filter_environment_id` to be set.",
			},
			SEGMENT_FILTER_ENVIRONMENT_ID: {
				Type:         schema.TypeString,
				Optional:     true,
				RequiredWith: []string{SEGMENT_FILTER},
				Description:  "The environment ID to use when resolving segment filters. Required when `segment_filter` is set. This is the environment's opaque ID (e.g. from `launchdarkly_project.environments[*].client_side_id`).",
			},
			RESOLVED_AT: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp of the last filter resolution. This value changes on every apply, ensuring linked resources stay in sync with the filter.",
			},
		},
	}
}

func resourceViewFilterLinksCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		return diag.FromErr(err)
	}

	projectKey := d.Get(PROJECT_KEY).(string)
	viewKey := d.Get(VIEW_KEY).(string)

	// Check if view exists
	exists, err := viewExists(projectKey, viewKey, betaClient)
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		return diag.Errorf("cannot find view with key %q in project %q", viewKey, projectKey)
	}

	segmentFilterEnvId := d.Get(SEGMENT_FILTER_ENVIRONMENT_ID).(string)

	// Link flags by filter if specified
	if flagFilter, ok := d.GetOk(FLAG_FILTER); ok {
		err = linkResourcesByFilterToView(betaClient, projectKey, viewKey, FLAGS, flagFilter.(string), "")
		if err != nil {
			return diag.Errorf("failed to link flags by filter to view %q in project %q: %s", viewKey, projectKey, err)
		}
	}

	// Link segments by filter if specified
	if segmentFilter, ok := d.GetOk(SEGMENT_FILTER); ok {
		err = linkResourcesByFilterToView(betaClient, projectKey, viewKey, SEGMENTS, segmentFilter.(string), segmentFilterEnvId)
		if err != nil {
			return diag.Errorf("failed to link segments by filter to view %q in project %q: %s", viewKey, projectKey, err)
		}
	}

	d.SetId(fmt.Sprintf("%s/%s", projectKey, viewKey))
	_ = d.Set(RESOLVED_AT, time.Now().UTC().Format(time.RFC3339))
	return nil
}

func resourceViewFilterLinksRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := metaRaw.(*Client)
	betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		return diag.FromErr(err)
	}

	projectKey := d.Get(PROJECT_KEY).(string)
	viewKey := d.Get(VIEW_KEY).(string)

	// Check if view still exists — if not, remove from state
	exists, err := viewExists(projectKey, viewKey, betaClient)
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		log.Printf("[WARN] view with key %q in project %q not found, removing from state", viewKey, projectKey)
		d.SetId("")
		return diags
	}

	// Filter strings are stored in state as-is — no API resolution needed.
	// We intentionally do NOT read back the resolved keys, because the filter
	// result set is dynamic and would cause phantom diffs on every plan.
	return diags
}

func resourceViewFilterLinksUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		return diag.FromErr(err)
	}

	projectKey := d.Get(PROJECT_KEY).(string)
	viewKey := d.Get(VIEW_KEY).(string)

	segmentFilterEnvId := d.Get(SEGMENT_FILTER_ENVIRONMENT_ID).(string)

	// Flags: full re-sync
	flagFilter := d.Get(FLAG_FILTER).(string)
	if flagFilter != "" {
		// Unlink all currently linked flags (clean slate)
		linkedFlags, err := getLinkedResources(betaClient, projectKey, viewKey, FLAGS)
		if err != nil {
			return diag.Errorf("failed to get linked flags for view %q in project %q: %s", viewKey, projectKey, err)
		}
		if len(linkedFlags) > 0 {
			flagKeys := make([]string, len(linkedFlags))
			for i, f := range linkedFlags {
				flagKeys[i] = f.ResourceKey
			}
			err = unlinkResourcesFromView(betaClient, projectKey, viewKey, FLAGS, flagKeys)
			if err != nil {
				return diag.Errorf("failed to unlink flags from view %q in project %q: %s", viewKey, projectKey, err)
			}
		}
		// Re-link by current filter
		err = linkResourcesByFilterToView(betaClient, projectKey, viewKey, FLAGS, flagFilter, "")
		if err != nil {
			return diag.Errorf("failed to link flags by filter to view %q in project %q: %s", viewKey, projectKey, err)
		}
	} else {
		// Flag filter was removed — unlink all flags
		oldVal, _ := d.GetChange(FLAG_FILTER)
		if oldVal.(string) != "" {
			linkedFlags, err := getLinkedResources(betaClient, projectKey, viewKey, FLAGS)
			if err != nil {
				return diag.Errorf("failed to get linked flags for view %q in project %q: %s", viewKey, projectKey, err)
			}
			if len(linkedFlags) > 0 {
				flagKeys := make([]string, len(linkedFlags))
				for i, f := range linkedFlags {
					flagKeys[i] = f.ResourceKey
				}
				err = unlinkResourcesFromView(betaClient, projectKey, viewKey, FLAGS, flagKeys)
				if err != nil {
					return diag.Errorf("failed to unlink flags from view %q in project %q: %s", viewKey, projectKey, err)
				}
			}
		}
	}

	// Segments: full re-sync
	segmentFilter := d.Get(SEGMENT_FILTER).(string)
	if segmentFilter != "" {
		// Unlink all currently linked segments (clean slate)
		linkedSegments, err := getLinkedResources(betaClient, projectKey, viewKey, SEGMENTS)
		if err != nil {
			return diag.Errorf("failed to get linked segments for view %q in project %q: %s", viewKey, projectKey, err)
		}
		if len(linkedSegments) > 0 {
			segmentIds := make([]ViewSegmentIdentifier, len(linkedSegments))
			for i, s := range linkedSegments {
				segmentIds[i] = ViewSegmentIdentifier{
					EnvironmentId: s.EnvironmentId,
					SegmentKey:    s.ResourceKey,
				}
			}
			err = unlinkSegmentsFromView(betaClient, projectKey, viewKey, segmentIds)
			if err != nil {
				return diag.Errorf("failed to unlink segments from view %q in project %q: %s", viewKey, projectKey, err)
			}
		}
		// Re-link by current filter
		err = linkResourcesByFilterToView(betaClient, projectKey, viewKey, SEGMENTS, segmentFilter, segmentFilterEnvId)
		if err != nil {
			return diag.Errorf("failed to link segments by filter to view %q in project %q: %s", viewKey, projectKey, err)
		}
	} else {
		// Segment filter was removed — unlink all segments
		oldVal, _ := d.GetChange(SEGMENT_FILTER)
		if oldVal.(string) != "" {
			linkedSegments, err := getLinkedResources(betaClient, projectKey, viewKey, SEGMENTS)
			if err != nil {
				return diag.Errorf("failed to get linked segments for view %q in project %q: %s", viewKey, projectKey, err)
			}
			if len(linkedSegments) > 0 {
				segmentIds := make([]ViewSegmentIdentifier, len(linkedSegments))
				for i, s := range linkedSegments {
					segmentIds[i] = ViewSegmentIdentifier{
						EnvironmentId: s.EnvironmentId,
						SegmentKey:    s.ResourceKey,
					}
				}
				err = unlinkSegmentsFromView(betaClient, projectKey, viewKey, segmentIds)
				if err != nil {
					return diag.Errorf("failed to unlink segments from view %q in project %q: %s", viewKey, projectKey, err)
				}
			}
		}
	}

	_ = d.Set(RESOLVED_AT, time.Now().UTC().Format(time.RFC3339))
	return nil
}

func resourceViewFilterLinksDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := metaRaw.(*Client)
	betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		return diag.FromErr(err)
	}

	projectKey := d.Get(PROJECT_KEY).(string)
	viewKey := d.Get(VIEW_KEY).(string)

	// Unlink all linked flags if flag_filter is set
	if _, ok := d.GetOk(FLAG_FILTER); ok {
		linkedFlags, err := getLinkedResources(betaClient, projectKey, viewKey, FLAGS)
		if err != nil {
			return diag.Errorf("failed to get linked flags for view %q in project %q: %s", viewKey, projectKey, err)
		}
		if len(linkedFlags) > 0 {
			flagKeys := make([]string, len(linkedFlags))
			for i, f := range linkedFlags {
				flagKeys[i] = f.ResourceKey
			}
			err = unlinkResourcesFromView(betaClient, projectKey, viewKey, FLAGS, flagKeys)
			if err != nil {
				return diag.Errorf("failed to unlink flags from view %q in project %q: %s", viewKey, projectKey, err)
			}
		}
	}

	// Unlink all linked segments if segment_filter is set
	if _, ok := d.GetOk(SEGMENT_FILTER); ok {
		linkedSegments, err := getLinkedResources(betaClient, projectKey, viewKey, SEGMENTS)
		if err != nil {
			return diag.Errorf("failed to get linked segments for view %q in project %q: %s", viewKey, projectKey, err)
		}
		if len(linkedSegments) > 0 {
			segmentIds := make([]ViewSegmentIdentifier, len(linkedSegments))
			for i, s := range linkedSegments {
				segmentIds[i] = ViewSegmentIdentifier{
					EnvironmentId: s.EnvironmentId,
					SegmentKey:    s.ResourceKey,
				}
			}
			err = unlinkSegmentsFromView(betaClient, projectKey, viewKey, segmentIds)
			if err != nil {
				return diag.Errorf("failed to unlink segments from view %q in project %q: %s", viewKey, projectKey, err)
			}
		}
	}

	return diags
}

func resourceViewFilterLinksExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		return false, err
	}
	projectKey := d.Get(PROJECT_KEY).(string)
	viewKey := d.Get(VIEW_KEY).(string)
	return viewExists(projectKey, viewKey, betaClient)
}

func resourceViewFilterLinksImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	projectKey, viewKey, err := viewIdToKeys(d.Id())
	if err != nil {
		return nil, err
	}

	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(VIEW_KEY, viewKey)

	return []*schema.ResourceData{d}, nil
}
