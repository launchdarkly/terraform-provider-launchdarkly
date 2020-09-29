package launchdarkly

import (
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

func resourceFeatureFlag() *schema.Resource {
	schemaMap := baseFeatureFlagSchema()
	schemaMap[NAME] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The feature flag's description",
	}
	schemaMap[VARIATION_TYPE] = variationTypeSchema()
	return &schema.Resource{
		Create: resourceFeatureFlagCreate,
		Read:   resourceFeatureFlagRead,
		Update: resourceFeatureFlagUpdate,
		Delete: resourceFeatureFlagDelete,
		Exists: resourceFeatureFlagExists,

		Importer: &schema.ResourceImporter{
			State: resourceFeatureFlagImport,
		},
		Schema: schemaMap,
	}
}

func resourceFeatureFlagCreate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)

	if exists, err := projectExists(projectKey, client); !exists {
		if err != nil {
			return err
		}
		return fmt.Errorf("Cannot find project with key %q", projectKey)
	}

	key := d.Get(KEY).(string)
	description := d.Get(DESCRIPTION).(string)
	flagName := d.Get(NAME).(string)
	tags := stringsFromResourceData(d, TAGS)
	includeInSnippet := d.Get(INCLUDE_IN_SNIPPET).(bool)
	temporary := d.Get(TEMPORARY).(bool)

	variations, err := variationsFromResourceData(d)
	if err != nil {
		return fmt.Errorf("invalid variations: %v", err)
	}

	defaults, err := defaultVariationsFromResourceData(d)
	if err != nil {
		return fmt.Errorf("invalid default variations: %v", err)
	}

	flag := ldapi.FeatureFlagBody{
		Name:             flagName,
		Key:              key,
		Description:      description,
		Variations:       variations,
		Temporary:        temporary,
		Tags:             tags,
		IncludeInSnippet: includeInSnippet,
		Defaults:         defaults,
	}

	_, _, err = handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.FeatureFlagsApi.PostFeatureFlag(client.ctx, projectKey, flag, nil)
	})

	if err != nil {
		return fmt.Errorf("failed to create flag %q in project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	// ld's api does not allow some fields to be passed in during flag creation so we do an update:
	// https://apidocs.launchdarkly.com/docs/create-feature-flag
	err = resourceFeatureFlagUpdate(d, metaRaw)
	if err != nil {
		// if there was a problem in the update state, we need to clean up completely by deleting the flag
		_, deleteErr := client.ld.FeatureFlagsApi.DeleteFeatureFlag(client.ctx, projectKey, key)
		if deleteErr != nil {
			return fmt.Errorf("failed to delete flag %q from project %q: %s", key, projectKey, handleLdapiErr(err))
		}
		return fmt.Errorf("failed to update flag with name %q key %q for projectKey %q: %s",
			flagName, key, projectKey, handleLdapiErr(err))
	}

	d.SetId(projectKey + "/" + key)
	return resourceFeatureFlagRead(d, metaRaw)
}

func resourceFeatureFlagRead(d *schema.ResourceData, metaRaw interface{}) error {
	return featureFlagRead(d, metaRaw, false)
}

func resourceFeatureFlagUpdate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	key := d.Get(KEY).(string)
	projectKey := d.Get(PROJECT_KEY).(string)
	description := d.Get(DESCRIPTION).(string)
	name := d.Get(NAME).(string)
	tags := stringsFromResourceData(d, TAGS)
	includeInSnippet := d.Get(INCLUDE_IN_SNIPPET).(bool)
	temporary := d.Get(TEMPORARY).(bool)
	customProperties := customPropertiesFromResourceData(d)

	patch := ldapi.PatchComment{
		Comment: "Terraform",
		Patch: []ldapi.PatchOperation{
			patchReplace("/name", name),
			patchReplace("/description", description),
			patchReplace("/tags", tags),
			patchReplace("/includeInSnippet", includeInSnippet),
			patchReplace("/temporary", temporary),
			patchReplace("/customProperties", customProperties),
		}}

	variationPatches, err := variationPatchesFromResourceData(d)
	if err != nil {
		return fmt.Errorf("failed to build variation patches. %v", err)
	}
	patch.Patch = append(patch.Patch, variationPatches...)

	// Only update the defaults if they are specified in the schema
	defaults, err := defaultVariationsFromResourceData(d)
	if err != nil {
		return fmt.Errorf("invalid default variations: %v", err)
	}
	if defaults != nil {
		patch.Patch = append(patch.Patch, patchReplace("/defaults", defaults))
	}

	// Only update the maintainer ID if is specified in the schema
	maintainerID, ok := d.GetOk(MAINTAINER_ID)
	if ok {
		patch.Patch = append(patch.Patch, patchReplace("/maintainerId", maintainerID.(string)))
	}

	_, _, err = handleRateLimit(func() (interface{}, *http.Response, error) {
		return handleNoConflict(func() (interface{}, *http.Response, error) {
			return client.ld.FeatureFlagsApi.PatchFeatureFlag(client.ctx, projectKey, key, patch)
		})
	})

	if err != nil {
		return fmt.Errorf("failed to update flag %q in project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	return resourceFeatureFlagRead(d, metaRaw)
}

func resourceFeatureFlagDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)

	_, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		res, err := client.ld.FeatureFlagsApi.DeleteFeatureFlag(client.ctx, projectKey, key)
		return nil, res, err
	})
	if err != nil {
		return fmt.Errorf("failed to delete flag %q from project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	return nil
}

func resourceFeatureFlagExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)

	_, res, err := client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projectKey, key, nil)
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
