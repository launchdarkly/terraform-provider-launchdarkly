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
			tags: tagsSchema(),
			//TODO waiting on https://github.com/launchdarkly/api-client-go/issues/1
			//custom_properties: {
			//	Type:     schema.TypeList,
			//	Optional: true,
			//	Elem: &schema.Resource{
			//		Schema: map[string]*schema.Schema{
			//			key: {
			//				Type:     schema.TypeString,
			//				Required: true,
			//			},
			//			name: {
			//				Type:     schema.TypeString,
			//				Required: true,
			//			},
			//			value: {
			//				Type:     schema.TypeList,
			//				Required: true,
			//				//TODO: can these be other types?
			//				Elem:     &schema.Schema{Type: schema.TypeString},
			//			},
			//		},
			//	},
			//},
		},
	}
}

type JsonCustomProperty struct {
	Name  string   `json:name`
	Value []string `json:value`
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

	variationType := d.Get(variation_type).(string)

	variations, err := variationsFromResourceData(d, variationType)
	if err != nil {
		return fmt.Errorf("Failed to create flag %q in project %q: %s", key, projectKey, err)
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

	_, _, err = client.LaunchDarkly.FeatureFlagsApi.PostFeatureFlag(client.Ctx, projectKey, flag, nil)

	if err != nil {
		return fmt.Errorf("Failed to create flag %q in project %q: %s", key, projectKey, err)
	}

	// LaunchDarkly's api does not allow some fields to be passed in during flag creation so we do an update:
	// https://apidocs.launchdarkly.com/docs/create-feature-flag
	err = resourceFeatureFlagUpdate(d, metaRaw)
	if err != nil {
		return fmt.Errorf("failed to update flag with name %q key %q for projectKey %q: %v", flagName, key, projectKey, err)
	}

	d.SetId(key)
	return resourceFeatureFlagRead(d, metaRaw)
}

func resourceFeatureFlagRead(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(project_key).(string)
	key := d.Get(key).(string)

	flag, _, err := client.LaunchDarkly.FeatureFlagsApi.GetFeatureFlag(client.Ctx, projectKey, key, nil)

	if err != nil {
		return fmt.Errorf("Failed to get flag %q of project %q: %s", key, projectKey, err)
	}

	//TODO waiting on https://github.com/launchdarkly/api-client-go/issues/1
	// transformedCustomProperties := transformCustomPropertiesFromLaunchDarklyFormat(flag.CustomProperties)

	d.Set(key, flag.Key)
	d.Set(name, flag.Name)
	d.Set(description, flag.Description)
	d.Set(variations, variationsToResourceData(flag.Variations))
	d.Set(tags, flag.Tags)
	d.Set(include_in_snippet, flag.IncludeInSnippet)
	d.Set(temporary, flag.Temporary)
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

	//TODO waiting on https://github.com/launchdarkly/api-client-go/issues/1
	//customProperties := d.Get(custom_properties).([]interface{})

	patch := ldapi.PatchComment{
		Comment: "Terraform",
		Patch: []ldapi.PatchOperation{
			patchReplace("/name", name),
			patchReplace("/description", description),
			patchReplace("/tags", tags),
			patchReplace("/includeInSnippet", includeInSnippet),
			patchReplace("/temporary", temporary),
			//TODO waiting on https://github.com/launchdarkly/api-client-go/issues/1
			// patchReplace("/customProperties", transformCustomPropertiesFromTerraformFormat(customProperties)),
		}}

	_, _, err := client.LaunchDarkly.FeatureFlagsApi.PatchFeatureFlag(client.Ctx, projectKey, key, patch)
	if err != nil {
		return fmt.Errorf("Failed to update flag %q in project %q: %s", key, projectKey, err)
	}

	return resourceFeatureFlagRead(d, metaRaw)
}

func resourceFeatureFlagDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(project_key).(string)
	key := d.Get(key).(string)

	_, err := client.LaunchDarkly.FeatureFlagsApi.DeleteFeatureFlag(client.Ctx, projectKey, key)
	if err != nil {
		return fmt.Errorf("Failed to delete flag %q from project %q: %s", key, projectKey, err)
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
		return false, fmt.Errorf("Failed to check if flag %q exists in project %q: %s", key, projectKey, err)
	}
	return true, nil
}

func resourceFeatureFlagImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	projectKey := defaultProjectKey
	key := d.Id()

	if strings.Contains(d.Id(), "/") {
		parts := strings.SplitN(d.Id(), "/", 2)

		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("ID must have format <key> or <project>/<key>")
		}

		projectKey, key = parts[0], parts[1]
	}

	d.Set(project_key, projectKey)
	d.Set(key, key)
	d.SetId(key)

	if err := resourceFeatureFlagRead(d, meta); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

func transformCustomPropertiesFromTerraformFormat(properties []interface{}) map[string]JsonCustomProperty {
	transformed := make(map[string]JsonCustomProperty)

	for _, raw := range properties {
		val := raw.(map[string]interface{})
		key := val[key].(string)
		name := val[name].(string)

		var values []string
		for _, v := range val[value].([]interface{}) {
			values = append(values, v.(string))
		}

		transformed[key] = JsonCustomProperty{
			Name:  name,
			Value: values,
		}
	}

	return transformed
}

func transformCustomPropertiesFromLaunchDarklyFormat(properties map[string]JsonCustomProperty) interface{} {
	transformed := make([]map[string]interface{}, 0)

	for key, body := range properties {
		sub := make(map[string]interface{})
		sub[key] = key
		sub[name] = body.Name
		sub[value] = body.Value

		transformed = append(transformed, sub)
	}

	return transformed
}
