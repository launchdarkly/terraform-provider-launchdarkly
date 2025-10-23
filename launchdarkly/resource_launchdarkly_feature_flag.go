package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v17"
)

func resourceFeatureFlag() *schema.Resource {
	schemaMap := baseFeatureFlagSchema(featureFlagSchemaOptions{isDataSource: false})
	schemaMap[NAME] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The human-readable name of the feature flag.",
	}
	schemaMap[VARIATION_TYPE] = variationTypeSchema()
	return &schema.Resource{
		CreateContext: resourceFeatureFlagCreate,
		ReadContext:   resourceFeatureFlagRead,
		UpdateContext: resourceFeatureFlagUpdate,
		DeleteContext: resourceFeatureFlagDelete,
		Exists:        resourceFeatureFlagExists,

		Importer: &schema.ResourceImporter{
			State: resourceFeatureFlagImport,
		},
		Schema: schemaMap,

		Description: `Provides a LaunchDarkly feature flag resource.

This resource allows you to create and manage feature flags within your LaunchDarkly organization.

-> **Note:** This resource is for global-level feature flag configuration. Unexpected behavior may result if your environment-level configurations are not also managed from Terraform.`,
	}
}

func resourceFeatureFlagCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)

	if exists, err := projectExists(projectKey, client); !exists {
		if err != nil {
			return diag.FromErr(err)
		}
		return diag.Errorf("cannot find project with key %q", projectKey)
	}

	key := d.Get(KEY).(string)
	description := d.Get(DESCRIPTION).(string)
	flagName := d.Get(NAME).(string)
	tags := stringsFromResourceData(d, TAGS)
	includeInSnippet := d.Get(INCLUDE_IN_SNIPPET).(bool)
	// GetOkExists is 'deprecated', but needed as optional booleans set to false return a 'false' ok value from GetOk
	// Also not really deprecated as they are keeping it around pending a replacement https://github.com/hashicorp/terraform-plugin-sdk/pull/350#issuecomment-597888969
	//nolint:staticcheck // SA1019
	_, includeInSnippetOk := d.GetOkExists(INCLUDE_IN_SNIPPET)
	_, clientSideAvailabilityOk := d.GetOk(CLIENT_SIDE_AVAILABILITY)
	clientSideAvailability := &ldapi.ClientSideAvailabilityPost{
		UsingEnvironmentId: d.Get("client_side_availability.0.using_environment_id").(bool),
		UsingMobileKey:     d.Get("client_side_availability.0.using_mobile_key").(bool),
	}
	temporary := d.Get(TEMPORARY).(bool)

	variations, err := variationsFromResourceData(d)
	if err != nil {
		return diag.Errorf("invalid variations: %v", err)
	}

	defaults, err := defaultVariationsFromResourceData(d)
	if err != nil {
		return diag.Errorf("invalid default variations: %v", err)
	}
	variationType := d.Get(VARIATION_TYPE).(string)
	if variationType == BOOL_VARIATION && len(variations) == 0 {
		// explicitly define default boolean variations.
		// this prevents the "Default off variation must be a valid index in the variations list"
		// error that we see when we define defaults but no variations
		variations = []ldapi.Variation{{Value: true}, {Value: false}}
	}

	flag := ldapi.FeatureFlagBody{
		Name:        flagName,
		Key:         key,
		Description: &description,
		Variations:  variations,
		Temporary:   &temporary,
		Tags:        tags,
		Defaults:    defaults,
	}

	if clientSideAvailabilityOk {
		flag.ClientSideAvailability = clientSideAvailability
	} else if includeInSnippetOk {
		// If includeInSnippet is set, still use clientSideAvailability behind the scenes in order to switch UsingMobileKey to false if needed
		flag.ClientSideAvailability = &ldapi.ClientSideAvailabilityPost{
			UsingEnvironmentId: includeInSnippet,
			UsingMobileKey:     false,
		}
	} else {
		// If neither value is set, we should get the default from the project level and apply that
		// IncludeInSnippetdefault is the same as defaultCSA.UsingEnvironmentId, so we can _ it
		defaultCSA, _, err := getProjectDefaultCSAandIncludeInSnippet(client, projectKey)
		if err != nil {
			return diag.Errorf("failed to get project level client side availability defaults. %v", err)
		}
		flag.ClientSideAvailability = &ldapi.ClientSideAvailabilityPost{
			UsingEnvironmentId: *defaultCSA.UsingEnvironmentId,
			UsingMobileKey:     *defaultCSA.UsingMobileKey,
		}
	}
	err = client.withConcurrency(ctx, func() error {
		_, _, err = client.ld.FeatureFlagsApi.PostFeatureFlag(client.ctx, projectKey).FeatureFlagBody(flag).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to create flag %q in project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	// ld's api does not allow some fields to be passed in during flag creation so we do an update:
	// https://apidocs.launchdarkly.com/tag/Feature-flags#operation/postFeatureFlag
	updateDiags := featureFlagUpdate(ctx, d, metaRaw, true)
	if updateDiags.HasError() {
		// if there was a problem in the update state, we need to clean up completely by deleting the flag
		err := client.withConcurrency(ctx, func() error {
			_, deleteErr := client.ld.FeatureFlagsApi.DeleteFeatureFlag(client.ctx, projectKey, key).Execute()
			return deleteErr
		})
		if err != nil {
			return diag.Errorf("failed to delete flag %q from project %q: %s", key, projectKey, handleLdapiErr(err))
		}
		return diag.Errorf("failed to update flag with name %q key %q for projectKey %q: %s",
			flagName, key, projectKey, updateDiags[0].Summary)
	}

	d.SetId(projectKey + "/" + key)
	return resourceFeatureFlagRead(ctx, d, metaRaw)
}

func resourceFeatureFlagRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return featureFlagRead(ctx, d, metaRaw, false)
}

func resourceFeatureFlagUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return featureFlagUpdate(ctx, d, metaRaw, false)
}

func featureFlagUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}, isCreate bool) diag.Diagnostics {
	client := metaRaw.(*Client)
	key := d.Get(KEY).(string)
	projectKey := d.Get(PROJECT_KEY).(string)
	description := d.Get(DESCRIPTION).(string)
	name := d.Get(NAME).(string)
	tags := stringsFromResourceData(d, TAGS)
	includeInSnippet := d.Get(INCLUDE_IN_SNIPPET).(bool)

	snippetHasChange := d.HasChange(INCLUDE_IN_SNIPPET)
	clientSideHasChange := d.HasChange(CLIENT_SIDE_AVAILABILITY)
	// GetOkExists is 'deprecated', but needed as optional booleans set to false return a 'false' ok value from GetOk
	// Also not really deprecated as they are keeping it around pending a replacement https://github.com/hashicorp/terraform-plugin-sdk/pull/350#issuecomment-597888969
	//nolint:staticcheck // SA1019
	_, includeInSnippetOk := d.GetOkExists(INCLUDE_IN_SNIPPET)
	_, clientSideAvailabilityOk := d.GetOk(CLIENT_SIDE_AVAILABILITY)
	temporary := d.Get(TEMPORARY).(bool)
	customProperties := customPropertiesFromResourceData(d)
	archived := d.Get(ARCHIVED).(bool)
	clientSideAvailability := &ldapi.ClientSideAvailabilityPost{
		UsingEnvironmentId: d.Get("client_side_availability.0.using_environment_id").(bool),
		UsingMobileKey:     d.Get("client_side_availability.0.using_mobile_key").(bool),
	}

	comment := "Terraform"
	patch := ldapi.PatchWithComment{
		Comment: &comment,
		Patch: []ldapi.PatchOperation{
			patchReplace("/name", name),
			patchReplace("/description", description),
			patchReplace("/tags", tags),
			patchReplace("/temporary", temporary),
			patchReplace("/customProperties", customProperties),
			patchReplace("/archived", archived),
		}}

	// if it was previously set and then removed, we don't want to update it or revert to project defaults in this case
	// because the LD API only sets it to project defaults upon flag creation and those defaults may have changed since
	if clientSideAvailabilityOk && clientSideHasChange {
		patch.Patch = append(patch.Patch, patchReplace("/clientSideAvailability", clientSideAvailability))
	} else if includeInSnippetOk && snippetHasChange {
		// If includeInSnippet is set, still use clientSideAvailability behind the scenes in order to switch UsingMobileKey to false if needed
		clientSideAvailability.UsingEnvironmentId = includeInSnippet // overwrite with user-set value
		patch.Patch = append(patch.Patch, patchReplace("/clientSideAvailability", clientSideAvailability))
	}

	variationPatches, err := variationPatchesFromResourceData(d)
	if err != nil {
		return diag.Errorf("failed to build variation patches. %v", err)
	}
	patch.Patch = append(patch.Patch, variationPatches...)

	// Only update the defaults if they are specified in the schema
	defaults, err := defaultVariationsFromResourceData(d)
	if err != nil {
		return diag.Errorf("invalid default variations: %v", err)
	}
	if defaults != nil {
		patch.Patch = append(patch.Patch, patchReplace("/defaults", defaults))
	}

	// Only update the maintainer fields if is specified in the schema
	if d.HasChange(MAINTAINER_ID) || d.HasChange(MAINTAINER_TEAM_KEY) {
		var flag *ldapi.FeatureFlag
		var res *http.Response
		var err error
		err = client.withConcurrency(ctx, func() error {
			flag, res, err = client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projectKey, key).Execute()
			return err
		})
		if isStatusNotFound(res) || err != nil {
			return diag.Errorf("error getting flag %q in project %q for update: %s", key, projectKey, handleLdapiErr(err))
		}
		maintainerId, maintainerIdOk := d.GetOk(MAINTAINER_ID)
		maintainerTeamKey, maintainerTeamKeyOk := d.GetOk(MAINTAINER_TEAM_KEY)

		if maintainerIdOk && maintainerTeamKeyOk {
			if d.HasChange(MAINTAINER_ID) {
				if maintainerId != "" {
					patch.Patch = append(patch.Patch, patchReplace("/maintainerId", maintainerId.(string)))
					if flag.MaintainerTeamKey != nil {
						patch.Patch = append(patch.Patch, patchRemove("/maintainerTeamKey"))
					}
				} else if flag.MaintainerId != nil {
					patch.Patch = append(patch.Patch, patchRemove("/maintainerId"))
				}
			}
			if d.HasChange(MAINTAINER_TEAM_KEY) {
				if maintainerTeamKey != "" {
					patch.Patch = append(patch.Patch, patchReplace("/maintainerTeamKey", maintainerTeamKey.(string)))
					if flag.MaintainerId != nil {
						patch.Patch = append(patch.Patch, patchRemove("/maintainerId"))
					}
				} else if flag.MaintainerTeamKey != nil {
					patch.Patch = append(patch.Patch, patchRemove("/maintainerTeamKey"))
				}
			}
			if d.HasChange(MAINTAINER_ID) && d.HasChange(MAINTAINER_TEAM_KEY) {
				fmt.Println("BOTH HAVE CHANGE SOMETHING IS WRONG")
			}
		} else if maintainerIdOk {
			patch.Patch = append(patch.Patch, patchReplace("/maintainerId", maintainerId.(string)))
			if flag.MaintainerTeamKey != nil {
				patch.Patch = append(patch.Patch, patchRemove("/maintainerTeamKey"))
			}
		} else if maintainerTeamKeyOk {
			patch.Patch = append(patch.Patch, patchReplace("/maintainerTeamKey", maintainerTeamKey.(string)))
			if flag.MaintainerId != nil {
				patch.Patch = append(patch.Patch, patchRemove("/maintainerId"))
			}
		}
	}

	err = client.withConcurrency(ctx, func() error {
		_, _, err = client.ld.FeatureFlagsApi.PatchFeatureFlag(client.ctx, projectKey, key).PatchWithComment(patch).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to update flag %q in project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	// Handle view associations if view_keys field is managed
	if d.HasChange(VIEW_KEYS) || isCreate {
		if viewKeysRaw, ok := d.GetOk(VIEW_KEYS); ok {
			betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
			if err != nil {
				return diag.Errorf("failed to create beta client for view linking: %v", err)
			}

			desiredViewKeys := interfaceSliceToStringSlice(viewKeysRaw.(*schema.Set).List())

			// Validate that all specified views exist
			for _, viewKey := range desiredViewKeys {
				exists, err := viewExists(projectKey, viewKey, betaClient)
				if err != nil {
					return diag.Errorf("failed to check if view %q exists: %v", viewKey, err)
				}
				if !exists {
					return diag.Errorf("cannot link flag to view %q in project %q: view does not exist", viewKey, projectKey)
				}
			}

			// Get currently linked views
			currentViewKeys, err := getViewsContainingFlag(betaClient, projectKey, key)
			if err != nil {
				log.Printf("[WARN] failed to get current views for flag %q: %v", key, err)
				currentViewKeys = []string{}
			}

			// Calculate views to add and remove
			viewsToAdd := difference(desiredViewKeys, currentViewKeys)
			viewsToRemove := difference(currentViewKeys, desiredViewKeys)

			// Check for potential conflicts with view_links resource
			// If we're using view_keys and there are views to remove, check if they were previously managed by view_keys
			if len(viewsToRemove) > 0 && !isCreate {
				oldViewKeysRaw, _ := d.GetChange(VIEW_KEYS)
				if oldViewKeysRaw != nil {
					oldViewKeys := interfaceSliceToStringSlice(oldViewKeysRaw.(*schema.Set).List())
					// Check if any views we're removing were NOT in our previous view_keys
					// This indicates they might be managed by view_links
					unexpectedViews := difference(viewsToRemove, oldViewKeys)
					if len(unexpectedViews) > 0 {
						return diag.Errorf(
							"Conflict detected: Flag %q is linked to views %v which are not managed by this resource's view_keys field. "+
								"This typically means these views are managed by a launchdarkly_view_links resource. "+
								"You cannot use both view_keys and view_links to manage the same flag. "+
								"Please either: (1) Remove view_keys from this flag and use view_links only, or "+
								"(2) Remove this flag from any view_links resources and use view_keys only.",
							key, unexpectedViews)
					}
				}
			}

			// Remove views that are no longer in the list
			for _, viewKey := range viewsToRemove {
				err = unlinkResourcesFromView(betaClient, projectKey, viewKey, FLAGS, []string{key})
				if err != nil {
					return diag.Errorf("failed to unlink flag %q from view %q: %v", key, viewKey, err)
				}
			}

			// Add new views
			for _, viewKey := range viewsToAdd {
				err = linkResourcesToView(betaClient, projectKey, viewKey, FLAGS, []string{key})
				if err != nil {
					return diag.Errorf("failed to link flag %q to view %q: %v", key, viewKey, err)
				}
			}
		} else if !isCreate {
			// If view_keys field was removed from config (HasChange but not present),
			// unlink from ALL current views to ensure clean removal
			betaClient, err := newBetaClient(client.apiKey, client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
			if err != nil {
				return diag.Errorf("failed to create beta client for view unlinking: %v", err)
			}

			// Get ALL currently linked views from the API
			currentViewKeys, err := getViewsContainingFlag(betaClient, projectKey, key)
			if err != nil {
				log.Printf("[WARN] failed to get current views for flag %q during removal: %v", key, err)
			} else {
				// Unlink from all current views
				for _, viewKey := range currentViewKeys {
					err = unlinkResourcesFromView(betaClient, projectKey, viewKey, FLAGS, []string{key})
					if err != nil {
						log.Printf("[WARN] failed to unlink flag %q from view %q: %v", key, viewKey, err)
					}
				}
			}
		}
	}

	return resourceFeatureFlagRead(ctx, d, metaRaw)
}

func resourceFeatureFlagDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)

	err := client.withConcurrency(ctx, func() error {
		_, err := client.ld.FeatureFlagsApi.DeleteFeatureFlag(client.ctx, projectKey, key).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to delete flag %q from project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	return diags
}

func resourceFeatureFlagExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)

	var res *http.Response
	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, res, err = client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projectKey, key).Execute()
		return err
	})
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check if flag %q exists in project %q: %s", key, projectKey, handleLdapiErr(err))
	}
	return true, nil
}

func resourceFeatureFlagImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()

	projectKey, flagKey, err := flagIdToKeys(id)
	if err != nil {
		return nil, err
	}
	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(KEY, flagKey)

	return []*schema.ResourceData{d}, nil
}
