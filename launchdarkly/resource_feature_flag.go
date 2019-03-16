package launchdarkly

import (
	"fmt"
	"strings"

	"github.com/launchdarkly/api-client-go"

	"github.com/hashicorp/terraform/helper/schema"
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
			project_key: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			key: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			name: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			description: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			variations: variationsSchema(),
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

	flag := ldapi.FeatureFlagBody{
		Name:             flagName,
		Key:              key,
		Description:      description,
		Variations:       variationsFromResourceData(d),
		Temporary:        temporary,
		Tags:             tags,
		IncludeInSnippet: includeInSnippet,
	}

	_, _, err := client.LaunchDarkly.FeatureFlagsApi.PostFeatureFlag(client.Ctx, projectKey, flag, nil)

	if err != nil {
		return fmt.Errorf("failed to create flag %q in project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	// LaunchDarkly's api does not allow some fields to be passed in during flag creation so we do an update:
	// https://apidocs.launchdarkly.com/docs/create-feature-flag
	err = resourceFeatureFlagUpdate(d, metaRaw)
	if err != nil {
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

	flag, _, err := client.LaunchDarkly.FeatureFlagsApi.GetFeatureFlag(client.Ctx, projectKey, key, nil)

	if err != nil {
		return fmt.Errorf("failed to get flag %q of project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	transformedCustomProperties := customPropertiesToResourceData(flag.CustomProperties)

	d.Set(key, flag.Key)
	d.Set(name, flag.Name)
	d.Set(description, flag.Description)
	d.Set(include_in_snippet, flag.IncludeInSnippet)
	d.Set(temporary, flag.Temporary)

	err = d.Set(variations, variationsToResourceData(flag.Variations))
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

	_, _, err := client.LaunchDarkly.FeatureFlagsApi.PatchFeatureFlag(client.Ctx, projectKey, key, patch)
	if err != nil {
		return fmt.Errorf("failed to update flag %q in project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	return resourceFeatureFlagRead(d, metaRaw)
}

func resourceFeatureFlagDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(project_key).(string)
	key := d.Get(key).(string)

	_, err := client.LaunchDarkly.FeatureFlagsApi.DeleteFeatureFlag(client.Ctx, projectKey, key)
	if err != nil {
		return fmt.Errorf("failed to delete flag %q from project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	return nil
}

func resourceFeatureFlagExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	client := metaRaw.(*Client)
	projectKey := d.Get(project_key).(string)
	key := d.Get(key).(string)

	_, httpResponse, err := client.LaunchDarkly.FeatureFlagsApi.GetFeatureFlag(client.Ctx, projectKey, key, nil)
	if httpResponse != nil && httpResponse.StatusCode == 404 {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check if flag %q exists in project %q: %s", key, projectKey, handleLdapiErr(err))
	}
	return true, nil
}

func resourceFeatureFlagImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()

	if strings.Count(id, "/") != 1 {
		return nil, fmt.Errorf("found unexpected flag id format: %q expected format: 'project_key/flag_key'", id)
	}
	parts := strings.SplitN(d.Id(), "/", 2)
	projectKey, flagKey := parts[0], parts[1]
	d.Set(project_key, projectKey)
	d.Set(key, flagKey)

	if err := resourceFeatureFlagRead(d, meta); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}
