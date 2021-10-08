package launchdarkly

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

func resourceEnvironment() *schema.Resource {
	envSchema := environmentSchema(false)
	envSchema[PROJECT_KEY] = &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		Description:  "The LaunchDarkly project key",
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

	approvalSettings := d.Get(APPROVAL_SETTINGS)
	if len(approvalSettings.([]interface{})) > 0 {
		err = resourceEnvironmentUpdate(d, metaRaw)
		if err != nil {
			// if there was a problem in the update state, we need to clean up completely by deleting the env
			_, deleteErr := client.ld.EnvironmentsApi.DeleteEnvironment(client.ctx, projectKey, key)
			if deleteErr != nil {
				return fmt.Errorf("failed to clean up environment %q from project %q: %s", key, projectKey, handleLdapiErr(err))
			}
			return fmt.Errorf("failed to update environment with name %q key %q for projectKey %q: %s",
				name, key, projectKey, handleLdapiErr(err))
		}
	}

	d.SetId(projectKey + "/" + key)
	return resourceEnvironmentRead(d, metaRaw)
}

func resourceEnvironmentRead(d *schema.ResourceData, metaRaw interface{}) error {
	return environmentRead(d, metaRaw, false)
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

	oldApprovalSettings, newApprovalSettings := d.GetChange(APPROVAL_SETTINGS)
	approvalPatch, err := approvalPatchFromSettings(oldApprovalSettings, newApprovalSettings)
	if err != nil {
		return err
	}
	patch = append(patch, approvalPatch...)
	_, _, err = handleRateLimit(func() (interface{}, *http.Response, error) {
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
	_, res, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return meta.ld.EnvironmentsApi.GetEnvironment(meta.ctx, projectKey, key)
	})
	if isStatusNotFound(res) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get environment with key %q for project %q: %v", key, projectKey, handleLdapiErr(err))
	}

	return true, nil
}

func environmentExistsInProject(project ldapi.Project, envKey string) bool {
	for _, env := range project.Environments {
		if env.Key == envKey {
			return true
		}
	}
	return false
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
