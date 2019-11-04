package launchdarkly

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

func resourceFeatureFlag() *schema.Resource {
	return &schema.Resource{
		Create: resourceFeatureFlagCreate,
		Read:   resourceFeatureFlagRead,
		Update: resourceFeatureFlagUpdate,
		Delete: resourceFeatureFlagDelete,
		Exists: resourceFeatureFlagExists,

		Importer: &schema.ResourceImporter{
			State: resourceFeatureFlagImport,
		},

		Schema: map[string]*schema.Schema{
			project_key: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The feature flag's project key",
				ValidateFunc: validateKey(),
			},
			key: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateKey(),
				Description:  "The human-readable name of the feature flag",
			},
			name: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The feature flag's description",
			},
			maintainer_id: {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				ValidateFunc: validateID(),
			},
			description: {
				Type:     schema.TypeString,
				Optional: true,
			},
			variation_type: variationTypeSchema(),
			variations:     variationsSchema(),
			temporary: {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			include_in_snippet: {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			tags:              tagsSchema(),
			custom_properties: customPropertiesSchema(),
		},
	}
}

func resourceFeatureFlagCreate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(project_key).(string)

	if exists, err := projectExists(projectKey, client); !exists {
		if err != nil {
			return err
		}
		return fmt.Errorf("Cannot find project with key %q", projectKey)
	}

	key := d.Get(key).(string)
	description := d.Get(description).(string)
	flagName := d.Get(name).(string)
	tags := stringsFromResourceData(d, tags)
	includeInSnippet := d.Get(include_in_snippet).(bool)
	temporary := d.Get(temporary).(bool)

	variations, err := variationsFromResourceData(d)
	if err != nil {
		return fmt.Errorf("invalid variations: %v", err)
	}
	flag := ldapi.FeatureFlagBody{
		Name:             flagName,
		Key:              key,
		Description:      description,
		Variations:       variations,
		Temporary:        temporary,
		Tags:             tags,
		IncludeInSnippet: includeInSnippet,
	}

	_, _, err = client.ld.FeatureFlagsApi.PostFeatureFlag(client.ctx, projectKey, flag, nil)

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
	client := metaRaw.(*Client)
	projectKey := d.Get(project_key).(string)
	key := d.Get(key).(string)

	flag, res, err := client.ld.FeatureFlagsApi.GetFeatureFlag(client.ctx, projectKey, key, nil)
	if isStatusNotFound(res) {
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to get flag %q of project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	transformedCustomProperties := customPropertiesToResourceData(flag.CustomProperties)
	_ = d.Set(key, flag.Key)
	_ = d.Set(name, flag.Name)
	_ = d.Set(maintainer_id, flag.MaintainerId)
	_ = d.Set(description, flag.Description)
	_ = d.Set(include_in_snippet, flag.IncludeInSnippet)
	_ = d.Set(temporary, flag.Temporary)

	variationType, err := variationsToVariationType(flag.Variations)
	if err != nil {
		return fmt.Errorf("failed to determine variation type on flag with key %q: %v", flag.Key, err)
	}
	err = d.Set(variation_type, variationType)
	if err != nil {
		return fmt.Errorf("failed to set variation type on flag with key %q: %v", flag.Key, err)
	}

	parsedVariations, err := variationsToResourceData(flag.Variations, variationType)
	if err != nil {
		return fmt.Errorf("failed to parse variations on flag with key %q: %v", flag.Key, err)
	}
	err = d.Set(variations, parsedVariations)
	if err != nil {
		return fmt.Errorf("failed to set variations on flag with key %q: %v", flag.Key, err)
	}

	err = d.Set(tags, flag.Tags)
	if err != nil {
		return fmt.Errorf("failed to set tags on flag with key %q: %v", flag.Key, err)
	}

	err = d.Set(custom_properties, transformedCustomProperties)
	if err != nil {
		return fmt.Errorf("failed to set custom properties on flag with key %q: %v", flag.Key, err)
	}
	d.SetId(projectKey + "/" + key)
	return nil
}

func resourceFeatureFlagUpdate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	key := d.Get(key).(string)
	projectKey := d.Get(project_key).(string)
	description := d.Get(description).(string)
	name := d.Get(name).(string)
	tags := stringsFromResourceData(d, tags)
	includeInSnippet := d.Get(include_in_snippet).(bool)
	temporary := d.Get(temporary).(bool)
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

	// Only update the maintainer ID if is specified in the schema
	maintainerID, ok := d.GetOk(maintainer_id)
	if ok {
		patch.Patch = append(patch.Patch, patchReplace("/maintainerId", maintainerID.(string)))
	}

	_, _, err = repeatUntilNoConflict(func() (interface{}, *http.Response, error) {
		return client.ld.FeatureFlagsApi.PatchFeatureFlag(client.ctx, projectKey, key, patch)
	})
	if err != nil {
		return fmt.Errorf("failed to update flag %q in project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	return resourceFeatureFlagRead(d, metaRaw)
}

func resourceFeatureFlagDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(project_key).(string)
	key := d.Get(key).(string)

	_, err := client.ld.FeatureFlagsApi.DeleteFeatureFlag(client.ctx, projectKey, key)
	if err != nil {
		return fmt.Errorf("failed to delete flag %q from project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	return nil
}

func resourceFeatureFlagExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	projectKey := d.Get(project_key).(string)
	key := d.Get(key).(string)

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
	_ = d.Set(project_key, projectKey)
	_ = d.Set(key, flagKey)

	if err := resourceFeatureFlagRead(d, meta); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

func flagIdToKeys(id string) (projectKey string, flagKey string, err error) {
	if strings.Count(id, "/") != 1 {
		return "", "", fmt.Errorf("found unexpected flag id format: %q expected format: 'project_key/flag_key'", id)
	}
	parts := strings.SplitN(id, "/", 2)
	projectKey, flagKey = parts[0], parts[1]
	return projectKey, flagKey, nil
}
