package launchdarkly

import (
	"fmt"
	"log"
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
	secureMode := d.Get(SECURE_MODE).(bool)
	defaultTrackEvents := d.Get(DEFAULT_TRACK_EVENTS).(bool)
	tags := stringsFromSchemaSet(d.Get(TAGS).(*schema.Set))
	requireComments := d.Get(REQUIRE_COMMENTS).(bool)
	confirmChanges := d.Get(CONFIRM_CHANGES).(bool)

	envPost := ldapi.EnvironmentPost{
		Name:               name,
		Key:                key,
		Color:              color,
		DefaultTtl:         defaultTTL,
		SecureMode:         secureMode,
		DefaultTrackEvents: defaultTrackEvents,
		Tags:               tags,
		RequireComments:    requireComments,
		ConfirmChanges:     confirmChanges,
	}

	_, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.EnvironmentsApi.PostEnvironment(client.ctx, projectKey, envPost)
	})
	if err != nil {
		return fmt.Errorf("failed to create environment: [%+v] for project key: %s: %s", envPost, projectKey, handleLdapiErr(err))
	}

	d.SetId(projectKey + "/" + key)
	return resourceEnvironmentRead(d, metaRaw)
}

func resourceEnvironmentRead(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)

	envRaw, res, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.EnvironmentsApi.GetEnvironment(client.ctx, projectKey, key)
	})
	env := envRaw.(ldapi.Environment)
	if isStatusNotFound(res) {
		log.Printf("[WARN] failed to find environment with key %q in project %q, removing from state", key, projectKey)
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

	_, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return handleNoConflict(func() (interface{}, *http.Response, error) {
			return client.ld.EnvironmentsApi.PatchEnvironment(client.ctx, projectKey, key, patch)
		})
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

	_, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		res, err := client.ld.EnvironmentsApi.DeleteEnvironment(client.ctx, projectKey, key)
		return nil, res, err
	})

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

	return []*schema.ResourceData{d}, nil
}
