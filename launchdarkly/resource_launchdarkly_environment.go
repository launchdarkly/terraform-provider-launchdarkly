package launchdarkly

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

func resourceEnvironment() *schema.Resource {
	envSchema := environmentSchema()
	envSchema[project_key] = &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		ForceNew:     true,
		ValidateFunc: validateKey(),
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
	defaultTTL := float32(d.Get(default_ttl).(int))

	envPost := ldapi.EnvironmentPost{
		Name:       name,
		Key:        key,
		Color:      color,
		DefaultTtl: defaultTTL,
	}

	_, _, err := client.ld.EnvironmentsApi.PostEnvironment(client.ctx, projectKey, envPost)
	if err != nil {
		return fmt.Errorf("failed to create environment: [%+v] for project key: %s: %s", envPost, projectKey, handleLdapiErr(err))
	}

	// ld's api does not allow some fields to be passed in during env creation so we do an update:
	// https://apidocs.launchdarkly.com/docs/create-environment
	err = resourceEnvironmentUpdate(d, metaRaw)
	if err != nil {
		return fmt.Errorf("failed to update environment with name %q key %q for projectKey %q: %v", name, key, projectKey, err)
	}

	d.SetId(projectKey + "/" + key)
	return resourceEnvironmentRead(d, metaRaw)
}

func resourceEnvironmentRead(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(project_key).(string)
	key := d.Get(key).(string)

	env, res, err := client.ld.EnvironmentsApi.GetEnvironment(client.ctx, projectKey, key)
	if isStatusNotFound(res) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get environment with key %q for project key: %q: %v", key, projectKey, handleLdapiErr(err))
	}

	d.SetId(projectKey + "/" + key)
	_ = d.Set(key, env.Key)
	_ = d.Set(name, env.Name)
	_ = d.Set(api_key, env.ApiKey)
	_ = d.Set(mobile_key, env.MobileKey)
	_ = d.Set(client_side_id, env.Id)
	_ = d.Set(color, env.Color)
	_ = d.Set(default_ttl, int(env.DefaultTtl))
	_ = d.Set(secure_mode, env.SecureMode)
	_ = d.Set(default_track_events, env.DefaultTrackEvents)
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
		patchReplace("/defaultTtl", d.Get(default_ttl)),
		patchReplace("/secureMode", d.Get(secure_mode)),
		patchReplace("/defaultTrackEvents", d.Get(default_track_events)),
	}

	_, _, err := repeatUntilNoConflict(func() (interface{}, *http.Response, error) {
		return client.ld.EnvironmentsApi.PatchEnvironment(client.ctx, projectKey, key, patch)
	})
	if err != nil {
		return fmt.Errorf("failed to update environment with key %q for project: %q: %s", key, projectKey, handleLdapiErr(err))
	}

	return resourceEnvironmentRead(d, metaRaw)
}

func resourceEnvironmentDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(project_key).(string)
	key := d.Get(key).(string)

	_, err := client.ld.EnvironmentsApi.DeleteEnvironment(client.ctx, projectKey, key)
	if err != nil {
		return fmt.Errorf("failed to delete project with key %q for project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	return nil
}

func resourceEnvironmentExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return environmentExists(d.Get(project_key).(string), d.Get(key).(string), metaRaw.(*Client))
}

func environmentExists(projectKey string, key string, meta *Client) (bool, error) {
	_, res, err := meta.ld.EnvironmentsApi.GetEnvironment(meta.ctx, projectKey, key)
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get project with key %q for project %q: %v", key, projectKey, handleLdapiErr(err))
	}

	return true, nil
}

func resourceEnvironmentImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()

	if strings.Count(id, "/") != 1 {
		return nil, fmt.Errorf("found unexpected environment id format: %q expected format: 'project_key/env_key'", id)
	}

	parts := strings.SplitN(d.Id(), "/", 2)

	projectKey, envKey := parts[0], parts[1]

	_ = d.Set(project_key, projectKey)
	_ = d.Set(key, envKey)

	if err := resourceEnvironmentRead(d, meta); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}
