package launchdarkly

import (
	"context"
	"fmt"
	"log"

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

		Description: `Provides a LaunchDarkly view filter links resource for linking resources to views using filter expressions.

This resource allows you to link all flags and/or segments matching a filter expression to a specific view. The filter is resolved at apply time — the backend finds all resources matching the filter and links them to the view.

-> **Note:** Filter-based links are "point-in-time". The filter is resolved when ` + "`terraform apply`" + ` runs. Resources created or tagged after the apply will not be automatically linked. Run ` + "`terraform apply`" + ` again to pick up new matches.

## When to use which resource

- **` + "`view_links`" + `**: You know the exact flag/segment keys to link. Terraform tracks the explicit list and detects drift if links are removed externally.
- **` + "`view_filter_links`" + ` (this resource)**: You want to link all resources matching a dynamic query (e.g. all flags tagged "frontend"). No drift detection on resolved keys — only changes to the filter string itself trigger updates.
- **` + "`view_keys`" + ` on individual resources**: Each flag/segment declares its own view membership. Best for modular Terraform structures.

-> **Warning:** Be careful not to use ` + "`view_filter_links`" + ` and ` + "`view_links`" + ` targeting the same view and resource type, as they may conflict.`,

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
				Description:  "A filter expression to match segments for linking to the view. Uses the same filter syntax as the segment list API endpoint (e.g. `tags:backend`). Requires `segment_filter_environment_id` to be set.",
			},
			SEGMENT_FILTER_ENVIRONMENT_ID: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The environment ID to use when resolving segment filters. Required when `segment_filter` is set. This is the environment's opaque ID (e.g. from `launchdarkly_project.environments[*].client_side_id`).",
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
	if exists, err := viewExists(projectKey, viewKey, betaClient); !exists {
		if err != nil {
			return diag.FromErr(err)
		}
		return diag.Errorf("cannot find view with key %q in project %q", viewKey, projectKey)
	}

	// Validate that segment_filter_environment_id is set when segment_filter is used
	segmentFilterEnvId := d.Get(SEGMENT_FILTER_ENVIRONMENT_ID).(string)
	if _, ok := d.GetOk(SEGMENT_FILTER); ok && segmentFilterEnvId == "" {
		return diag.Errorf("%q is required when %q is set", SEGMENT_FILTER_ENVIRONMENT_ID, SEGMENT_FILTER)
	}

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
	if exists, err := viewExists(projectKey, viewKey, betaClient); !exists {
		if err != nil {
			return diag.FromErr(err)
		}
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

	// Validate that segment_filter_environment_id is set when segment_filter is used
	segmentFilterEnvId := d.Get(SEGMENT_FILTER_ENVIRONMENT_ID).(string)
	if _, ok := d.GetOk(SEGMENT_FILTER); ok && segmentFilterEnvId == "" {
		return diag.Errorf("%q is required when %q is set", SEGMENT_FILTER_ENVIRONMENT_ID, SEGMENT_FILTER)
	}

	if d.HasChange(FLAG_FILTER) {
		oldVal, newVal := d.GetChange(FLAG_FILTER)
		oldFilter := oldVal.(string)
		newFilter := newVal.(string)

		// Unlink using old filter if it was set
		if oldFilter != "" {
			err = unlinkResourcesByFilterFromView(betaClient, projectKey, viewKey, FLAGS, oldFilter, "")
			if err != nil {
				return diag.Errorf("failed to unlink flags by old filter from view %q in project %q: %s", viewKey, projectKey, err)
			}
		}

		// Link using new filter if it is set
		if newFilter != "" {
			err = linkResourcesByFilterToView(betaClient, projectKey, viewKey, FLAGS, newFilter, "")
			if err != nil {
				return diag.Errorf("failed to link flags by new filter to view %q in project %q: %s", viewKey, projectKey, err)
			}
		}
	}

	if d.HasChange(SEGMENT_FILTER) || d.HasChange(SEGMENT_FILTER_ENVIRONMENT_ID) {
		oldVal, newVal := d.GetChange(SEGMENT_FILTER)
		oldFilter := oldVal.(string)
		newFilter := newVal.(string)

		oldEnvVal, _ := d.GetChange(SEGMENT_FILTER_ENVIRONMENT_ID)
		oldEnvId := oldEnvVal.(string)

		// Unlink using old filter if it was set
		if oldFilter != "" {
			err = unlinkResourcesByFilterFromView(betaClient, projectKey, viewKey, SEGMENTS, oldFilter, oldEnvId)
			if err != nil {
				return diag.Errorf("failed to unlink segments by old filter from view %q in project %q: %s", viewKey, projectKey, err)
			}
		}

		// Link using new filter if it is set
		if newFilter != "" {
			err = linkResourcesByFilterToView(betaClient, projectKey, viewKey, SEGMENTS, newFilter, segmentFilterEnvId)
			if err != nil {
				return diag.Errorf("failed to link segments by new filter to view %q in project %q: %s", viewKey, projectKey, err)
			}
		}
	}

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

	segmentFilterEnvId := d.Get(SEGMENT_FILTER_ENVIRONMENT_ID).(string)

	// Unlink flags by filter if set
	if flagFilter, ok := d.GetOk(FLAG_FILTER); ok {
		err = unlinkResourcesByFilterFromView(betaClient, projectKey, viewKey, FLAGS, flagFilter.(string), "")
		if err != nil {
			return diag.Errorf("failed to unlink flags by filter from view %q in project %q: %s", viewKey, projectKey, err)
		}
	}

	// Unlink segments by filter if set
	if segmentFilter, ok := d.GetOk(SEGMENT_FILTER); ok {
		err = unlinkResourcesByFilterFromView(betaClient, projectKey, viewKey, SEGMENTS, segmentFilter.(string), segmentFilterEnvId)
		if err != nil {
			return diag.Errorf("failed to unlink segments by filter from view %q in project %q: %s", viewKey, projectKey, err)
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
