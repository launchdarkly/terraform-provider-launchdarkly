package launchdarkly

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

func resourceEnvironment() *schema.Resource {
	envSchema := environmentSchema()
	envSchema[PROJECT_KEY] = &schema.Schema{
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
	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)
	name := d.Get(NAME).(string)
	color := d.Get(COLOR).(string)
	defaultTTL := float32(d.Get(DEFAULT_TTL).(int))

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
	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)

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
	_ = d.Set(NAME, env.Name)
	_ = d.Set(API_KEY, env.ApiKey)
	_ = d.Set(MOBILE_KEY, env.MobileKey)
	_ = d.Set(CLIENT_SIDE_ID, env.Id)
	_ = d.Set(COLOR, env.Color)
	_ = d.Set(DEFAULT_TTL, int(env.DefaultTtl))
	_ = d.Set(SECURE_MODE, env.SecureMode)
	_ = d.Set(DEFAULT_TRACK_EVENTS, env.DefaultTrackEvents)
	_ = d.Set(TAGS, env.Tags)
	_ = d.Set(REQUIRE_COMMENTS, env.RequireComments)
	_ = d.Set(CONFIRM_CHANGES, env.ConfirmChanges)
	return nil
}

func resourceEnvironmentUpdate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)

	//required fields
	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)
	name := d.Get(NAME)
	color := d.Get(COLOR)
	tags := stringsFromResourceData(d, TAGS)
	requireComments := d.Get(REQUIRE_COMMENTS)
	confirmChanges := d.Get(CONFIRM_CHANGES)

	patch := []ldapi.PatchOperation{
		patchReplace("/name", &name),
		patchReplace("/color", &color),
		patchReplace("/defaultTtl", d.Get(DEFAULT_TTL)),
		patchReplace("/secureMode", d.Get(SECURE_MODE)),
		patchReplace("/defaultTrackEvents", d.Get(DEFAULT_TRACK_EVENTS)),
		patchReplace("/tags", &tags),
		patchReplace("/requireComments", &requireComments),
		patchReplace("/confirmChanges", &confirmChanges),
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
	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)

	_, err := client.ld.EnvironmentsApi.DeleteEnvironment(client.ctx, projectKey, key)
	if err != nil {
		return fmt.Errorf("failed to delete project with key %q for project %q: %s", key, projectKey, handleLdapiErr(err))
	}

	return nil
}

func resourceEnvironmentExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return environmentExists(d.Get(PROJECT_KEY).(string), d.Get(KEY).(string), metaRaw.(*Client))
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

	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(KEY, envKey)

	if err := resourceEnvironmentRead(d, meta); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}
