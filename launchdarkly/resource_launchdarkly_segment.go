package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v17"
)

func resourceSegment() *schema.Resource {
	schemaMap := baseSegmentSchema(segmentSchemaOptions{isDataSource: false})
	schemaMap[PROJECT_KEY] = &schema.Schema{
		Type:             schema.TypeString,
		Required:         true,
		ForceNew:         true,
		ValidateDiagFunc: validateKey(),
		Description:      addForceNewDescription("The segment's project key.", true),
	}
	schemaMap[ENV_KEY] = &schema.Schema{
		Type:             schema.TypeString,
		Required:         true,
		ForceNew:         true,
		ValidateDiagFunc: validateKey(),
		Description:      addForceNewDescription("The segment's environment key.", true),
	}
	schemaMap[KEY] = &schema.Schema{
		Type:             schema.TypeString,
		Required:         true,
		ForceNew:         true,
		ValidateDiagFunc: validateKey(),
		Description:      addForceNewDescription("The unique key that references the segment.", true),
	}
	schemaMap[NAME] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The human-friendly name for the segment.",
	}
	return &schema.Resource{
		CreateContext: resourceSegmentCreate,
		ReadContext:   resourceSegmentRead,
		UpdateContext: resourceSegmentUpdate,
		DeleteContext: resourceSegmentDelete,
		Exists:        resourceSegmentExists,

		Importer: &schema.ResourceImporter{
			State: resourceSegmentImport,
		},

		Schema: schemaMap,
		Description: `Provides a LaunchDarkly segment resource.

This resource allows you to create and manage segments within your LaunchDarkly organization.`,
	}
}

func resourceSegmentCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	envKey := d.Get(ENV_KEY).(string)

	key := d.Get(KEY).(string)
	description := d.Get(DESCRIPTION).(string)
	segmentName := d.Get(NAME).(string)
	tags := stringsFromResourceData(d, TAGS)
	unbounded := d.Get(UNBOUNDED).(bool)
	unboundedContextKind := d.Get(UNBOUNDED_CONTEXT_KIND).(string)

	segment := ldapi.SegmentBody{
		Name:                 segmentName,
		Key:                  key,
		Description:          &description,
		Tags:                 tags,
		Unbounded:            &unbounded,
		UnboundedContextKind: &unboundedContextKind,
	}

	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, _, err = client.ld.SegmentsApi.PostSegment(client.ctx, projectKey, envKey).SegmentBody(segment).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to create segment %q in project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	// ld's api does not allow some fields to be passed in during segment creation so we do an update:
	// https://apidocs.launchdarkly.com/reference#create-segment
	updateDiags := resourceSegmentUpdate(ctx, d, metaRaw)
	if updateDiags.HasError() {
		// TODO: Figure out if we can get the err out of updateDiag (not looking likely) to use in handleLdapiErr
		return updateDiags
		// return diag.Errorf("failed to update segment with name %q key %q for projectKey %q: %s",
		// 	segmentName, key, projectKey, handleLdapiErr(errs))
	}

	d.SetId(projectKey + "/" + envKey + "/" + key)
	return resourceSegmentRead(ctx, d, metaRaw)
}

func resourceSegmentRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return segmentRead(ctx, d, metaRaw, false)
}

func resourceSegmentUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	key := d.Get(KEY).(string)
	projectKey := d.Get(PROJECT_KEY).(string)
	envKey := d.Get(ENV_KEY).(string)
	description := d.Get(DESCRIPTION).(string)
	name := d.Get(NAME).(string)
	tags := stringsFromResourceData(d, TAGS)
	included := d.Get(INCLUDED).([]interface{})
	excluded := d.Get(EXCLUDED).([]interface{})
	includedContexts := segmentTargetsFromResourceData(d, segmentTargetOptions{Included: true})
	excludedContexts := segmentTargetsFromResourceData(d, segmentTargetOptions{Excluded: true})
	rules, err := segmentRulesFromResourceData(d, RULES)
	if err != nil {
		return diag.FromErr(err)
	}
	comment := "Terraform"
	patchOps := []ldapi.PatchOperation{
		patchReplace("/name", name),
		patchReplace("/description", description),
		patchReplace("/temporary", TEMPORARY),
		patchReplace("/included", included),
		patchReplace("/excluded", excluded),
		patchReplace("/rules", rules),
		patchReplace("/includedContexts", includedContexts),
		patchReplace("/excludedContexts", excludedContexts),
	}

	tagPatch := patchReplace("/tags", tags)
	if d.HasChange(TAGS) && len(tags) == 0 {
		tagPatch = patchRemove("/tags")
	}
	patchOps = append(patchOps, tagPatch)

	err = client.withConcurrency(client.ctx, func() error {
		_, _, err = client.ld.SegmentsApi.PatchSegment(client.ctx, projectKey, envKey, key).PatchWithComment(ldapi.PatchWithComment{
			Comment: &comment,
			Patch:   patchOps}).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to update segment %q in project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	// Handle view associations if view_keys field is managed
	if d.HasChange(VIEW_KEYS) {
		if viewKeysRaw, ok := d.GetOk(VIEW_KEYS); ok {
			betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
			if err != nil {
				return diag.Errorf("failed to create beta client for view linking: %v", err)
			}

			desiredViewKeys := interfaceSliceToStringSlice(viewKeysRaw.(*schema.Set).List())

			// Get the environment ID
			var env *ldapi.Environment
			err = client.withConcurrency(client.ctx, func() error {
				env, _, err = client.ld.EnvironmentsApi.GetEnvironment(client.ctx, projectKey, envKey).Execute()
				return err
			})
			if err != nil {
				return diag.Errorf("failed to get environment %q in project %q: %s", envKey, projectKey, handleLdapiErr(err))
			}

			// Validate that all specified views exist
			for _, viewKey := range desiredViewKeys {
				exists, err := viewExists(projectKey, viewKey, betaClient)
				if err != nil {
					return diag.Errorf("failed to check if view %q exists: %v", viewKey, err)
				}
				if !exists {
					return diag.Errorf("cannot link segment to view %q in project %q: view does not exist", viewKey, projectKey)
				}
			}

			// Get currently linked views
			currentViewKeys, err := getViewsContainingSegment(betaClient, projectKey, env.Id, key)
			if err != nil {
				log.Printf("[WARN] failed to get current views for segment %q: %v", key, err)
				currentViewKeys = []string{}
			}

			// Calculate views to add and remove
			viewsToAdd := difference(desiredViewKeys, currentViewKeys)
			viewsToRemove := difference(currentViewKeys, desiredViewKeys)

			// Warn if there might be conflicts with view_links resource
			if len(viewsToRemove) > 0 {
				log.Printf("[INFO] Segment %q: Unlinking from views %v. If you're also using launchdarkly_view_links to manage this segment, this may cause conflicts.", key, viewsToRemove)
			}

			// Remove views that are no longer in the list
			for _, viewKey := range viewsToRemove {
				segmentIdentifiers := []ViewSegmentIdentifier{{
					EnvironmentId: env.Id,
					SegmentKey:    key,
				}}
				err = unlinkSegmentsFromView(betaClient, projectKey, viewKey, segmentIdentifiers)
				if err != nil {
					return diag.Errorf("failed to unlink segment %q from view %q: %v", key, viewKey, err)
				}
			}

			// Add new views
			for _, viewKey := range viewsToAdd {
				segmentIdentifiers := []ViewSegmentIdentifier{{
					EnvironmentId: env.Id,
					SegmentKey:    key,
				}}
				err = linkSegmentsToView(betaClient, projectKey, viewKey, segmentIdentifiers)
				if err != nil {
					return diag.Errorf("failed to link segment %q to view %q: %v", key, viewKey, err)
				}
			}
		} else {
			// If view_keys was explicitly removed (set to null), unlink from all views
			// that were previously managed by this resource
			betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
			if err == nil {
				oldViewKeysRaw, _ := d.GetChange(VIEW_KEYS)
				if oldViewKeysRaw != nil {
					// Get the environment ID
					var env *ldapi.Environment
					err = client.withConcurrency(client.ctx, func() error {
						env, _, err = client.ld.EnvironmentsApi.GetEnvironment(client.ctx, projectKey, envKey).Execute()
						return err
					})
					if err == nil {
						oldViewKeys := interfaceSliceToStringSlice(oldViewKeysRaw.(*schema.Set).List())
						for _, viewKey := range oldViewKeys {
							segmentIdentifiers := []ViewSegmentIdentifier{{
								EnvironmentId: env.Id,
								SegmentKey:    key,
							}}
							err = unlinkSegmentsFromView(betaClient, projectKey, viewKey, segmentIdentifiers)
							if err != nil {
								log.Printf("[WARN] failed to unlink segment %q from view %q: %v", key, viewKey, err)
							}
						}
					}
				}
			}
		}
	}

	return resourceSegmentRead(ctx, d, metaRaw)
}

func resourceSegmentDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	envKey := d.Get(ENV_KEY).(string)
	key := d.Get(KEY).(string)

	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, err = client.ld.SegmentsApi.DeleteSegment(client.ctx, projectKey, envKey, key).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to delete segment %q from project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	return diags
}

func resourceSegmentExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	envKey := d.Get(ENV_KEY).(string)
	key := d.Get(KEY).(string)

	var res *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, res, err = client.ld.SegmentsApi.GetSegment(client.ctx, projectKey, envKey, key).Execute()
		return err
	})
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check if segment %q exists in project %q: %s",
			key, projectKey, handleLdapiErr(err))
	}
	return true, nil
}

func resourceSegmentImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()

	if strings.Count(id, "/") != 2 {
		return nil, fmt.Errorf("found unexpected segment id format: %q expected format: 'project_key/env_key/segment_key'", id)
	}

	parts := strings.SplitN(d.Id(), "/", 3)

	projectKey, envKey, segmentKey := parts[0], parts[1], parts[2]

	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(ENV_KEY, envKey)
	_ = d.Set(KEY, segmentKey)

	return []*schema.ResourceData{d}, nil
}
