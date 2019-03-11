package launchdarkly

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/launchdarkly/api-client-go"
)

func resourceEnvironment() *schema.Resource {
	envSchema := environmentSchema()
	envSchema[project_key] = &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
		Default:  defaultProjectKey,
		ForceNew: true,
	}

	return &schema.Resource{
		Create: resourceEnvironmentCreate,
		Read:   resourceEnvironmentRead,
		Update: resourceEnvironmentUpdate,
		Delete: resourceEnvironmentDelete,
		Exists: resourceEnvironmentExists,

		Importer: &schema.ResourceImporter{
			State: resourceEnvironmentImport,
		},
		Schema: envSchema,
	}
}

func resourceEnvironmentCreate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(project_key).(string)
	key := d.Get(key).(string)
	name := d.Get(name).(string)
	color := d.Get(color).(string)
	defaultTtl := float32(d.Get(default_ttl).(float64))

	envPost := ldapi.EnvironmentPost{
		Name:       name,
		Key:        key,
		Color:      color,
		DefaultTtl: defaultTtl,
	}

	_, _, err := client.LaunchDarkly.EnvironmentsApi.PostEnvironment(client.Ctx, projectKey, envPost)
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
	projectKey := d.Get(project_key).(string)
	key := d.Get(key).(string)

	env, _, err := client.LaunchDarkly.EnvironmentsApi.GetEnvironment(client.Ctx, projectKey, key)
	if err != nil {
		return fmt.Errorf("failed to get environment with key %q for project key: %q: %v", key, projectKey, err)
	}

	d.Set(key, env.Key)
	d.Set(name, env.Name)
	d.Set(api_key, env.ApiKey)
	d.Set(mobile_key, env.MobileKey)
	d.Set(color, env.Color)
	d.Set(default_ttl, env.DefaultTtl)
	d.Set(secure_mode, env.SecureMode)
	d.Set(default_track_events, env.DefaultTrackEvents)
	d.Set(tags, env.Tags)
	return nil
}

func resourceEnvironmentUpdate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)

	//required fields
	projectKey := d.Get(project_key).(string)
	key := d.Get(key).(string)
	name := d.Get(name)
	color := d.Get(color)

	patch := []ldapi.PatchOperation{
		patchReplace("/name", &name),
		patchReplace("/color", &color),
	}

	// optional fields
	if defaultTtl, ok := d.GetOk(default_ttl); ok {
		patch = append(patch, patchReplace("/defaultTtl", &defaultTtl))
	}

	if secureMode, ok := d.GetOk(secure_mode); ok {
		patch = append(patch, patchReplace("/secureMode", &secureMode))
	}

	if defaultTrackEvents, ok := d.GetOk(default_track_events); ok {
		patch = append(patch, patchReplace("/defaultTrackEvents", &defaultTrackEvents))
	}

	if _, ok := d.GetOk(tags); ok {
		tagSet := stringsFromResourceData(d, tags)
		patch = append(patch, patchReplace("/tags", &tagSet))
	}

	_, _, err := client.LaunchDarkly.EnvironmentsApi.PatchEnvironment(client.Ctx, projectKey, key, patch)
	if err != nil {
		return fmt.Errorf("failed to update environment with key %q for project: %q: %s", key, projectKey, err)
	}

	return resourceEnvironmentRead(d, metaRaw)
}

func resourceEnvironmentDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(project_key).(string)
	key := d.Get(key).(string)

	_, err := client.LaunchDarkly.EnvironmentsApi.DeleteEnvironment(client.Ctx, projectKey, key)
	if err != nil {
		return fmt.Errorf("failed to delete project with key %q for project %q: %s", key, projectKey, err)
	}

	return nil
}

func resourceEnvironmentExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return environmentExists(d.Get(project_key).(string), d.Get(key).(string), metaRaw.(*Client))
}

func environmentExists(projectKey string, key string, meta *Client) (bool, error) {
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

	d.Set(project_key, project)
	d.Set(key, key)
	d.SetId(key)

	if err := resourceEnvironmentRead(d, meta); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}
