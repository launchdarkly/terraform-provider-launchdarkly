package launchdarkly

import (
	"fmt"

	ldapi "github.com/launchdarkly/api-client-go"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceProjectCreate,
		Read:   resourceProjectRead,
		Update: resourceProjectUpdate,
		Delete: resourceProjectDelete,
		Exists: resourceProjectExists,

		Importer: &schema.ResourceImporter{
			State: resourceProjectImport,
		},

		Schema: map[string]*schema.Schema{
			key: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			name: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			tags: tagsSchema(),
			environments: &schema.Schema{
				Type:     schema.TypeSet,
				Set:      environmentHash,
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
	projectKey := d.Get(key).(string)
	name := d.Get(name).(string)
	envs := environmentPostsFromResourceData(d)

	projectBody := ldapi.ProjectBody{
		Name: name,
		Key:  projectKey,
	}

	if len(envs) > 0 {
		projectBody.Environments = envs
	}

	_, _, err := client.LaunchDarkly.ProjectsApi.PostProject(client.Ctx, projectBody)
	if err != nil {
		return fmt.Errorf("failed to create project with name %s and projectKey %s: %v", name, projectKey, handleLdapiErr(err))
	}

	// LaunchDarkly's api does not allow tags to be passed in during project creation so we do an update
	err = resourceProjectUpdate(d, metaRaw)
	if err != nil {
		return fmt.Errorf("failed to update project with name %s and projectKey %s: %v", name, projectKey, err)
	}

	// update envs if needed
	schemaEnvs := d.Get(environments).(*schema.Set)
	for _, env := range schemaEnvs.List() {
		envMap := env.(map[string]interface{})
		envKey := envMap[key].(string)

		// we already posted the projectKey, name, color, and default_ttl, so we skip patching those fields.
		var patch []ldapi.PatchOperation

		// optional fields:
		if defaultTtl, ok := envMap[default_ttl]; ok {
			patch = append(patch, patchReplace("/defaultTtl", &defaultTtl))
		}

		if secureMode, ok := envMap[secure_mode]; ok {
			patch = append(patch, patchReplace("/secureMode", &secureMode))
		}

		if defaultTrackEvents, ok := envMap[default_track_events]; ok {
			patch = append(patch, patchReplace("/defaultTrackEvents", &defaultTrackEvents))
		}

		if len(patch) > 0 {
			_, _, err := client.LaunchDarkly.EnvironmentsApi.PatchEnvironment(client.Ctx, projectKey, envKey, patch)
			if err != nil {
				return fmt.Errorf("failed to update environment with key %q for project: %q: %+v", envKey, projectKey, err)
			}
		}
	}
	d.SetId(projectKey)
	return resourceProjectRead(d, metaRaw)
}

func resourceProjectRead(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(key).(string)

	project, _, err := client.LaunchDarkly.ProjectsApi.GetProject(client.Ctx, projectKey)
	if err != nil {
		return fmt.Errorf("failed to get project with key %q: %v", projectKey, err)
	}

	d.Set(key, project.Key)
	d.Set(name, project.Name)

	envsRaw := environmentsToResourceData(project.Environments)
	err = d.Set(environments, envsRaw)
	if err != nil {
		return fmt.Errorf("could not set environments on project with key %q: %v", project.Key, err)
	}
	err = d.Set(tags, project.Tags)
	if err != nil {
		return fmt.Errorf("could not set tags on project with key %q: %v", project.Key, err)
	}
	return nil
}

func resourceProjectUpdate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(key).(string)
	name := d.Get(name)
	tags := stringsFromResourceData(d, tags)

	patch := []ldapi.PatchOperation{
		patchReplace("/name", &name),
		patchReplace("/tags", &tags),
	}

	_, _, err := client.LaunchDarkly.ProjectsApi.PatchProject(client.Ctx, projectKey, patch)
	if err != nil {
		return fmt.Errorf("failed to update project with key %q: %s", projectKey, handleLdapiErr(err))
	}

	return resourceProjectRead(d, metaRaw)
}

func resourceProjectDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(key).(string)

	_, err := client.LaunchDarkly.ProjectsApi.DeleteProject(client.Ctx, projectKey)
	if err != nil {
		return fmt.Errorf("failed to delete project with key %q: %s", projectKey, handleLdapiErr(err))
	}

	return nil
}

func resourceProjectExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return projectExists(d.Get(key).(string), metaRaw.(*Client))
}

func projectExists(projectKey string, meta *Client) (bool, error) {
	_, httpResponse, err := meta.LaunchDarkly.ProjectsApi.GetProject(meta.Ctx, projectKey)
	if httpResponse != nil && httpResponse.StatusCode == 404 {
		fmt.Println("got 404 when getting project. returning false.")
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get project with key %q: %v", projectKey, handleLdapiErr(err))
	}

	return true, nil
}

func resourceProjectImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	d.Set(key, d.Id())

	if err := resourceProjectRead(d, meta); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}
