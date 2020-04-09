package launchdarkly

import (
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

func resourceProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceProjectCreate,
		Read:   resourceProjectRead,
		Update: resourceProjectUpdate,
		Delete: resourceProjectDelete,
		Exists: resourceProjectExists,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			KEY: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateKey(),
			},
			NAME: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			TAGS: tagsSchema(),
			ENVIRONMENTS: &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: environmentSchema(),
				},
			},
		},
	}
}

func resourceProjectCreate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(KEY).(string)
	name := d.Get(NAME).(string)
	envs := environmentPostsFromResourceData(d)

	d.SetId(projectKey)
	projectBody := ldapi.ProjectBody{
		Name: name,
		Key:  projectKey,
	}

	if len(envs) > 0 {
		projectBody.Environments = envs
	}

	_, _, err := client.ld.ProjectsApi.PostProject(client.ctx, projectBody)
	if err != nil {
		return fmt.Errorf("failed to create project with name %s and projectKey %s: %v", name, projectKey, handleLdapiErr(err))
	}

	// ld's api does not allow tags to be passed in during project creation so we do an update
	err = resourceProjectUpdate(d, metaRaw)
	if err != nil {
		return fmt.Errorf("failed to update project with name %s and projectKey %s: %v", name, projectKey, err)
	}
	return nil
}

func resourceProjectRead(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(KEY).(string)

	project, res, err := client.ld.ProjectsApi.GetProject(client.ctx, projectKey)
	if isStatusNotFound(res) {
		log.Printf("[WARN] failed to find project with key %q, removing from state", projectKey)
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get project with key %q: %v", projectKey, err)
	}

	_ = d.Set(KEY, project.Key)
	_ = d.Set(NAME, project.Name)

	envsRaw := environmentsToResourceData(project.Environments)
	err = d.Set(ENVIRONMENTS, envsRaw)
	if err != nil {
		return fmt.Errorf("could not set environments on project with key %q: %v", project.Key, err)
	}
	err = d.Set(TAGS, project.Tags)
	if err != nil {
		return fmt.Errorf("could not set tags on project with key %q: %v", project.Key, err)
	}
	return nil
}

func resourceProjectUpdate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(KEY).(string)
	projName := d.Get(NAME)
	projTags := stringsFromResourceData(d, TAGS)

	patch := []ldapi.PatchOperation{
		patchReplace("/name", &projName),
		patchReplace("/tags", &projTags),
	}

	_, _, err := repeatUntilNoConflict(func() (interface{}, *http.Response, error) {
		return client.ld.ProjectsApi.PatchProject(client.ctx, projectKey, patch)
	})
	if err != nil {
		return fmt.Errorf("failed to update project with key %q: %s", projectKey, handleLdapiErr(err))
	}
	// Update environments if necessary
	schemaEnvs := d.Get(ENVIRONMENTS).([]interface{})
	for _, env := range schemaEnvs {
		envMap := env.(map[string]interface{})
		envKey := envMap[KEY].(string)

		// we already posted the projectKey, name, color, and default_ttl, so we skip patching those fields.
		envName := envMap[NAME].(string)
		envColor := envMap[COLOR].(string)
		patch := []ldapi.PatchOperation{
			patchReplace("/name", envName),
			patchReplace("/color", envColor),
		}

		// optional fields:
		if defaultTTL, ok := envMap[DEFAULT_TTL]; ok {
			patch = append(patch, patchReplace("/defaultTtl", defaultTTL.(int)))
		}

		if secureMode, ok := envMap[SECURE_MODE]; ok {
			patch = append(patch, patchReplace("/secureMode", &secureMode))
		}

		if defaultTrackEvents, ok := envMap[DEFAULT_TRACK_EVENTS]; ok {
			patch = append(patch, patchReplace("/defaultTrackEvents", &defaultTrackEvents))
		}

		if envTagsSet, ok := envMap[TAGS].(*schema.Set); ok {
			envTags := stringsFromSchemaSet(envTagsSet)
			patch = append(patch, patchReplace("/tags", &envTags))
		}

		if len(patch) > 0 {
			_, _, err := repeatUntilNoConflict(func() (interface{}, *http.Response, error) {
				return client.ld.EnvironmentsApi.PatchEnvironment(client.ctx, projectKey, envKey, patch)
			})
			if err != nil {
				return fmt.Errorf("failed to update environment with key %q for project: %q: %+v", envKey, projectKey, err)
			}
		}
	}

	return resourceProjectRead(d, metaRaw)
}

func resourceProjectDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(KEY).(string)

	_, err := client.ld.ProjectsApi.DeleteProject(client.ctx, projectKey)
	if err != nil {
		return fmt.Errorf("failed to delete project with key %q: %s", projectKey, handleLdapiErr(err))
	}

	return nil
}

func resourceProjectExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return projectExists(d.Get(KEY).(string), metaRaw.(*Client))
}

func projectExists(projectKey string, meta *Client) (bool, error) {
	_, res, err := meta.ld.ProjectsApi.GetProject(meta.ctx, projectKey)
	if isStatusNotFound(res) {
		log.Println("got 404 when getting project. returning false.")
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get project with key %q: %v", projectKey, handleLdapiErr(err))
	}

	return true, nil
}
