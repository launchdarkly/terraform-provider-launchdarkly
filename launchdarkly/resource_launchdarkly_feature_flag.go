package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v14"
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

This resource allows you to create and manage feature flags within your LaunchDarkly organization.`,
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
	_, _, err = client.ld.FeatureFlagsApi.PostFeatureFlag(client.ctx, projectKey).FeatureFlagBody(flag).Execute()
	if err != nil {
		return diag.Errorf("failed to create flag %q in project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	// ld's api does not allow some fields to be passed in during flag creation so we do an update:
	// https://apidocs.launchdarkly.com/tag/Feature-flags#operation/postFeatureFlag
	updateDiags := featureFlagUpdate(ctx, d, metaRaw, true)
	if updateDiags.HasError() {
		// if there was a problem in the update state, we need to clean up completely by deleting the flag
		_, deleteErr := client.ld.FeatureFlagsApi.DeleteFeatureFlag(client.ctx, projectKey, key).Execute()
		if deleteErr != nil {
			return diag.Errorf("failed to delete flag %q from project %q: %s", key, projectKey, handleLdapiErr(deleteErr))
		}
		// TODO: Figure out if we can get the err out of updateDiag (not looking likely) to use in hanldeLdapiErr
		return updateDiags
		// return diag.Errorf("failed to update flag with name %q key %q for projectKey %q: %s",
		// 	flagName, key, projectKey, handleLdapiErr(errs))
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
		flag, res, err := client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projectKey, key).Execute()
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

	_, _, err = client.ld.FeatureFlagsApi.PatchFeatureFlag(client.ctx, projectKey, key).PatchWithComment(patch).Execute()
	if err != nil {
		return diag.Errorf("failed to update flag %q in project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	return resourceFeatureFlagRead(ctx, d, metaRaw)
}

func resourceFeatureFlagDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)

	_, err := client.ld.FeatureFlagsApi.DeleteFeatureFlag(client.ctx, projectKey, key).Execute()
	if err != nil {
		return diag.Errorf("failed to delete flag %q from project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	return diags
}

func resourceFeatureFlagExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)

	_, res, err := client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projectKey, key).Execute()
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
