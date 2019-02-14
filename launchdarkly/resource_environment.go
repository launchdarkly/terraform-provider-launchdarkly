package launchdarkly

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

func resourceEnvironment() *schema.Resource {
	return &schema.Resource{
		Create: resourceEnvironmentCreate,
		Read:   resourceEnvironmentRead,
		Update: resourceEnvironmentUpdate,
		Delete: resourceEnvironmentDelete,
		Exists: resourceEnvironmentExists,

		Importer: &schema.ResourceImporter{
			State: resourceEnvironmentImport,
		},

		Schema: map[string]*schema.Schema{
			projKey: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "default",
				ForceNew: true,
			},
			"key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"api_key": &schema.Schema{
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"mobile_key": &schema.Schema{
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"color": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"default_ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"secure_mode": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"default_track_events": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceEnvironmentCreate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(projKey).(string)
	key := d.Get("key").(string)
	name := d.Get("name").(string)
	color := d.Get("color").(string)
	defaultTtl := float32(d.Get("default_ttl").(int))

	envPost := ldapi.EnvironmentPost{
		Name:       name,
		Key:        key,
		Color:      color,
		DefaultTtl: defaultTtl,
	}

	_, err := client.LaunchDarkly.EnvironmentsApi.PostEnvironment(client.Ctx, projectKey, envPost)
	if err != nil {
		return fmt.Errorf("failed to create environment: [%+v] for project key: %s", envPost, projectKey)
	}

	// LaunchDarkly's api does not allow some fields to be passed in during env creation so we do an update:
	// https://apidocs.launchdarkly.com/docs/create-environment
	err = resourceEnvironmentUpdate(d, metaRaw)
	if err != nil {
		return fmt.Errorf("failed to update environment with name %q key %q for projectKey %q: %v", name, key, projectKey, err)
	}

	d.SetId(key)
	return resourceEnvironmentRead(d, metaRaw)
}

func resourceEnvironmentRead(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(projKey).(string)
	key := d.Get("key").(string)

	env, _, err := client.LaunchDarkly.EnvironmentsApi.GetEnvironment(client.Ctx, projectKey, key)
	if err != nil {
		return fmt.Errorf("failed to get environment with key %q for project key: %q: %v", key, projectKey, err)
	}

	d.Set("key", env.Key)
	d.Set("name", env.Name)
	d.Set("api_key", env.ApiKey)
	d.Set("mobile_key", env.MobileKey)
	d.Set("color", env.Color)
	d.Set("default_ttl", env.DefaultTtl)
	d.Set("secure_mode", env.SecureMode)
	d.Set("default_track_events", env.DefaultTrackEvents)
	d.Set("tags", env.Tags)
	return nil
}

func resourceEnvironmentUpdate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(projKey).(string)
	key := d.Get("key").(string)
	name := d.Get("name")
	color := d.Get("color")
	defaultTtl := d.Get("default_ttl")
	secureMode := d.Get("secure_mode")
	defaultTrackEvents := d.Get("default_track_events")
	tags := stringSetFromResourceData(d, "tags")

	patch := []ldapi.PatchOperation{
		patchReplace("/name", &name),
		patchReplace("/color", &color),
		patchReplace("/defaultTtl", &defaultTtl),
		patchReplace("/secureMode", &secureMode),
		patchReplace("/defaultTrackEvents", &defaultTrackEvents),
		patchReplace("/tags", &tags),
	}

	_, _, err := client.LaunchDarkly.EnvironmentsApi.PatchEnvironment(client.Ctx, projectKey, key, patch)
	if err != nil {
		return fmt.Errorf("failed to update environment with key %q for project: %q: %s", key, projectKey, err)
	}

	return resourceEnvironmentRead(d, metaRaw)
}

func resourceEnvironmentDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(projKey).(string)
	key := d.Get("key").(string)

	_, err := client.LaunchDarkly.EnvironmentsApi.DeleteEnvironment(client.Ctx, projectKey, key)
	if err != nil {
		return fmt.Errorf("failed to delete project with key %q for project %q: %s", key, projectKey, err)
	}

	return nil
}

func resourceEnvironmentExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return environmentExists(d.Get("key").(string), d.Get(projKey).(string), metaRaw.(*Client))
}

func environmentExists(key string, projectKey string, meta *Client) (bool, error) {
	_, httpResponse, err := meta.LaunchDarkly.EnvironmentsApi.GetEnvironment(meta.Ctx, projectKey, key)
	if httpResponse != nil && httpResponse.StatusCode == 404 {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get project with key %q for project %q: %v", key, projectKey, err)
	}

	return true, nil
}

func resourceEnvironmentImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	project := defaultProjectKey
	key := d.Id()

	if strings.Contains(d.Id(), "/") {
		parts := strings.SplitN(d.Id(), "/", 2)

		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("ID must have format <key> or <project>/<key>")
		}

		project, key = parts[0], parts[1]
	}

	d.Set(projKey, project)
	d.Set("key", key)
	d.SetId(key)

	if err := resourceEnvironmentRead(d, meta); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}
