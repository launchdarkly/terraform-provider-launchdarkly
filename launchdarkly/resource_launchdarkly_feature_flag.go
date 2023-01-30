package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v10"
)

// We assign a custom diff in cases where the customer has not assigned CSA or IIS in config for a flag in order to respect project level defaults
func customizeFlagDiff(ctx context.Context, diff *schema.ResourceDiff, v interface{}) error {
	config := diff.GetRawConfig()
	client := v.(*Client)
	projectKey := diff.Get(PROJECT_KEY).(string)

	// Below values will exist due to the schema, we need to check if they are all null
	snippetInConfig := config.GetAttr(INCLUDE_IN_SNIPPET)
	csaInConfig := config.GetAttr(CLIENT_SIDE_AVAILABILITY)

	// If we have no keys in the CSA block in the config (length is 0) we know the customer hasn't set any CSA values
	csaKeys := csaInConfig.AsValueSlice()
	if len(csaKeys) == 0 {
		// When we have no values for either clienSideAvailability or includeInSnippet
		// Force an UPDATE call by setting a new value for INCLUDE_IN_SNIPPET in the diff according to project defaults
		if snippetInConfig.IsNull() {
			defaultCSA, includeInSnippetByDefault, err := getProjectDefaultCSAandIncludeInSnippet(client, projectKey)
			// We will fall into this block during the first config read when a user creates a flag at the same time they create the parent project
			// (and during our tests)
			// We can ignore the error here, as it is correctly handled during update/create (and doesn't occur then as the project will have been created)
			if err != nil {
			} else {
				// We set our values to the project defaults in order to guarantee an update call happening
				// If we don't do this, we can run into an edge case described below
				// IF previous value of INCLUDE_IN_SNIPPET was false
				// AND the project default value for INCLUDE_IN_SNIPPET is true
				// AND the customer removes the INCLUDE_IN_SNIPPET key from the config without replacing with defaultCSA
				// The read would assume no changes are needed, HOWEVER we need to jump back to project level set defaults
				// Hence the setting below
				err := diff.SetNew(INCLUDE_IN_SNIPPET, includeInSnippetByDefault)
				if err != nil {
					return err
				}
				err = diff.SetNew(CLIENT_SIDE_AVAILABILITY, []map[string]interface{}{{
					USING_ENVIRONMENT_ID: defaultCSA.UsingEnvironmentId,
					USING_MOBILE_KEY:     defaultCSA.UsingMobileKey,
				}})
				if err != nil {
					return err
				}
			}
		}

	}

	return nil
}

func resourceFeatureFlag() *schema.Resource {
	schemaMap := baseFeatureFlagSchema()
	schemaMap[NAME] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "A human-friendly name for the feature flag",
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
		Schema:        schemaMap,
		CustomizeDiff: customizeFlagDiff,
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
	updateDiags := resourceFeatureFlagUpdate(ctx, d, metaRaw)
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

	if clientSideAvailabilityOk && clientSideHasChange {
		patch.Patch = append(patch.Patch, patchReplace("/clientSideAvailability", clientSideAvailability))
	} else if includeInSnippetOk && snippetHasChange {
		// If includeInSnippet is set, still use clientSideAvailability behind the scenes in order to switch UsingMobileKey to false if needed
		patch.Patch = append(patch.Patch, patchReplace("/clientSideAvailability", &ldapi.ClientSideAvailabilityPost{
			UsingEnvironmentId: includeInSnippet,
			UsingMobileKey:     false,
		}))
	} else {
		// If the user doesn't set either CSA or IIS in config, we pull the defaults from their Project level settings and apply those
		// IncludeInSnippetdefault is the same as defaultCSA.UsingEnvironmentId, so we can _ it
		defaultCSA, _, err := getProjectDefaultCSAandIncludeInSnippet(client, projectKey)
		if err != nil {
			return diag.Errorf("failed to get project level client side availability defaults. %v", err)
		}
		patch.Patch = append(patch.Patch, patchReplace("/clientSideAvailability", &ldapi.ClientSideAvailabilityPost{
			UsingEnvironmentId: *defaultCSA.UsingEnvironmentId,
			UsingMobileKey:     *defaultCSA.UsingMobileKey,
		}))
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

	// Only update the maintainer ID if is specified in the schema
	maintainerID, ok := d.GetOk(MAINTAINER_ID)
	if ok {
		patch.Patch = append(patch.Patch, patchReplace("/maintainerId", maintainerID.(string)))
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
