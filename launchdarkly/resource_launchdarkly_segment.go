package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// customizeSegmentDiff validates that view_keys is set when the project requires view association for new segments
func customizeSegmentDiff(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
	// Only validate on create (when there's no ID yet)
	if diff.Id() != "" {
		return nil
	}

	client := meta.(*Client)
	projectKey := diff.Get(PROJECT_KEY).(string)

	// Fetch project view settings
	viewSettings, err := getProjectViewSettings(ctx, client, projectKey)
	if err != nil {
		// Log warning but don't fail - the setting might not be available
		log.Printf("[WARN] could not fetch project view settings for %q during plan: %v", projectKey, err)
		return nil
	}

	if viewSettings.RequireViewAssociationForNewSegments {
		viewKeys := optionalSchemaSetFromInterface(diff.Get(VIEW_KEYS))
		if viewKeys == nil || viewKeys.Len() == 0 {
			return fmt.Errorf("project %q requires new segments to be associated with at least one view. Please set the 'view_keys' attribute", projectKey)
		}
	}

	return nil
}

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

		CustomizeDiff: customizeSegmentDiff,

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
	envKey := effectiveEnvKeyFromIDOrAttr(d)
	if envKey == "" {
		return diag.Errorf(
			"%s is required (LaunchDarkly environment key). If the embedded schema omits it, set resource id to project_key/env_key/segment_key before create.",
			ENV_KEY)
	}

	if exists, err := projectExists(projectKey, client); !exists {
		if err != nil {
			return diag.FromErr(err)
		}
		return diag.Errorf("cannot find project with key %q", projectKey)
	}
	if exists, err := environmentExists(projectKey, envKey, client); !exists {
		if err != nil {
			return diag.FromErr(err)
		}
		return diag.Errorf(
			"environment %q not found in project %q — env_key must match the LaunchDarkly environment **key**. Create nested `environments` or a `launchdarkly_environment` first.",
			envKey, projectKey)
	}

	key := d.Get(KEY).(string)
	description := optionalStringAttr(d, DESCRIPTION)
	segmentName := d.Get(NAME).(string)
	tags := stringsFromResourceData(d, TAGS)
	unbounded := optionalBoolFromResourceData(d, UNBOUNDED, false)
	unboundedContextKind := optionalStringAttr(d, UNBOUNDED_CONTEXT_KIND)

	// Check if view_keys is specified - if so, we need to use raw HTTP to include it in creation
	var viewKeys []string
	if viewKeysRaw, ok := d.GetOk(VIEW_KEYS); ok {
		if viewKeysSet := optionalSchemaSetFromInterface(viewKeysRaw); viewKeysSet != nil {
			for _, v := range viewKeysSet.List() {
				viewKeys = append(viewKeys, v.(string))
			}
		}
	}

	var err error
	if len(viewKeys) > 0 {
		// Use raw HTTP call to include viewKeys in the creation request
		segmentBody := SegmentBodyWithViewKeys{
			Name:                 segmentName,
			Key:                  key,
			Description:          description,
			Tags:                 tags,
			Unbounded:            unbounded,
			UnboundedContextKind: unboundedContextKind,
			ViewKeys:             viewKeys,
		}
		err = client.withConcurrency(ctx, func() error {
			return createSegmentWithViewKeys(ctx, client, projectKey, envKey, segmentBody)
		})
	} else {
		// Use the standard API client when no view_keys are specified
		segment := ldapi.SegmentBody{
			Name:                 segmentName,
			Key:                  key,
			Description:          &description,
			Tags:                 tags,
			Unbounded:            &unbounded,
			UnboundedContextKind: &unboundedContextKind,
		}
		err = client.withConcurrency(ctx, func() error {
			_, _, err = client.ld.SegmentsApi.PostSegment(client.ctx, projectKey, envKey).SegmentBody(segment).Execute()
			return err
		})
	}
	if err != nil {
		return diag.Errorf("failed to create segment %q in project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	// Set the ID immediately after POST so that any subsequent error during
	// PATCH still leaves the created segment tracked in Terraform state.
	// Without this, a PATCH that hits the segment-approval gate would leave
	// an orphan LD-side resource that Terraform doesn't know about. See #370.
	d.SetId(projectKey + "/" + envKey + "/" + key)

	// LD's POST /api/v2/segments body only carries name/key/description/tags/
	// unbounded/unboundedContextKind. Rule and target fields must be applied
	// via a follow-up PATCH. We emit that PATCH only when the user actually
	// configured one of those fields, so minimal configs don't trip the
	// segment-approval gate. See #370.
	patchOps, patchFields, err := segmentPostCreatePatchOps(d)
	if err != nil {
		return diag.Errorf("failed to build patch ops for segment %q in project %q: %s", key, projectKey, err)
	}
	if len(patchOps) > 0 {
		comment := "Terraform"
		var patchRes *http.Response
		err = client.withConcurrency(client.ctx, func() error {
			_, patchRes, err = client.ld.SegmentsApi.PatchSegment(client.ctx, projectKey, envKey, key).PatchWithComment(ldapi.PatchWithComment{
				Comment: &comment,
				Patch:   patchOps,
			}).Execute()
			return err
		})
		if err != nil {
			if is403ApprovalRequired(patchRes, err) {
				// Populate state from the partially-created segment so the
				// user can inspect it via `terraform state show` and recover
				// without re-importing. Read failures are appended but not
				// allowed to mask the primary approval-required diagnostic.
				// SDKv2 will mark the resource tainted because Create
				// returned an error — the diagnostic below tells the user
				// how to break that loop.
				diags := resourceSegmentRead(ctx, d, metaRaw)
				diags = append(diags, diag.Errorf(
					"segment %q was created in project %q environment %q, but the follow-up PATCH to set %v was rejected because segment approvals are required on this environment. "+
						"The segment exists on the LaunchDarkly side with the POST-carried fields applied (name, description, tags, unbounded settings); only the rule/target fields are unset. "+
						"Terraform has marked the resource tainted (id %q) — re-applying without further action will destroy and recreate it and hit the same gate. "+
						"To recover: either (a) have an approver disable segment approvals on this environment, then `terraform untaint <resource_address>` and `terraform apply` to apply the remaining PATCH in place; or (b) leave the partial segment as-is, `terraform untaint <resource_address>`, and submit the rule/target changes via the LaunchDarkly UI approval workflow.",
					key, projectKey, envKey, patchFields, d.Id())...)
				return diags
			}
			return diag.Errorf("failed to apply post-create patch to segment %q in project %q: %s", key, projectKey, handleLdapiErr(err))
		}
	}

	return resourceSegmentRead(ctx, d, metaRaw)
}

func resourceSegmentRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return segmentRead(ctx, d, metaRaw, false)
}

func resourceSegmentUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	key := d.Get(KEY).(string)
	projectKey := d.Get(PROJECT_KEY).(string)
	envKey := effectiveEnvKeyFromIDOrAttr(d)
	if envKey == "" {
		return diag.Errorf("%s is empty and resource id %q is not project_key/env_key/segment_key", ENV_KEY, d.Id())
	}
	description := optionalStringAttr(d, DESCRIPTION)
	name := d.Get(NAME).(string)
	tags := stringsFromResourceData(d, TAGS)
	included := getOptionalInterfaceSlice(d, INCLUDED)
	excluded := getOptionalInterfaceSlice(d, EXCLUDED)
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

			desiredViewKeys := stringListFromOptionalSetValue(viewKeysRaw)

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

			// Detect potential mixed-management and warn, but still reconcile to configured view_keys.
			// This preserves convergence when out-of-band changes happen between plan and apply.
			if len(viewsToRemove) > 0 {
				oldViewKeysRaw, _ := d.GetChange(VIEW_KEYS)
				if oldViewKeysRaw != nil {
					oldViewKeys := stringListFromOptionalSetValue(oldViewKeysRaw)
					unexpectedViews := difference(viewsToRemove, oldViewKeys)
					if len(unexpectedViews) > 0 {
						log.Printf(
							"[WARN] Segment %q has view associations %v that were not previously tracked by view_keys; "+
								"proceeding to reconcile to configured view_keys. Avoid mixing view_keys and launchdarkly_view_links "+
								"for the same segment to prevent ownership conflicts.",
							key, unexpectedViews,
						)
					}
				}
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
			if err != nil {
				return diag.Errorf("failed to create beta client for view unlinking: %v", err)
			}

			oldViewKeysRaw, _ := d.GetChange(VIEW_KEYS)
			if oldViewKeysRaw != nil {
				// Get the environment ID
				var env *ldapi.Environment
				err = client.withConcurrency(client.ctx, func() error {
					env, _, err = client.ld.EnvironmentsApi.GetEnvironment(client.ctx, projectKey, envKey).Execute()
					return err
				})
				if err != nil {
					return diag.Errorf("failed to get environment %q in project %q: %s", envKey, projectKey, handleLdapiErr(err))
				}

				oldViewKeys := stringListFromOptionalSetValue(oldViewKeysRaw)
				for _, viewKey := range oldViewKeys {
					segmentIdentifiers := []ViewSegmentIdentifier{{
						EnvironmentId: env.Id,
						SegmentKey:    key,
					}}
					err = unlinkSegmentsFromView(betaClient, projectKey, viewKey, segmentIdentifiers)
					if err != nil {
						return diag.Errorf("failed to unlink segment %q from view %q: %v", key, viewKey, err)
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
	envKey := effectiveEnvKeyFromIDOrAttr(d)
	if envKey == "" {
		return diag.Errorf("%s is empty and resource id %q is not project_key/env_key/segment_key", ENV_KEY, d.Id())
	}
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
	envKey := effectiveEnvKeyFromIDOrAttr(d)
	if envKey == "" {
		return false, fmt.Errorf("%s is required, or resource id must be project_key/env_key/segment_key", ENV_KEY)
	}
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

	projectKey := strings.TrimSpace(parts[0])
	envKey := strings.TrimSpace(parts[1])
	segmentKey := strings.TrimSpace(parts[2])

	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(ENV_KEY, envKey)
	_ = d.Set(KEY, segmentKey)

	return []*schema.ResourceData{d}, nil
}
