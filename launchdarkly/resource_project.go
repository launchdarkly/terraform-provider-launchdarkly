package launchdarkly

import (
	"fmt"

	ldapi "github.com/launchdarkly/api-client-go"

	"github.com/hashicorp/terraform/helper/schema"
)

const (
	defaultProjectKey = "default"
	projKey           = "project_key"
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
			"key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"tags":         tagsSchema(),
			"environments": environmentsSchema(),
		},
	}
}

func resourceProjectCreate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	key := d.Get("key").(string)
	name := d.Get("name").(string)
	envs := environmentsSetFromResourceData(d)

	projectBody := ldapi.ProjectBody{
		Name: name,
		Key:  key,
	}

	if len(envs) > 0 {
		projectBody.Environments = envs
	}

	_, err := client.LaunchDarkly.ProjectsApi.PostProject(client.Ctx, projectBody)
	if err != nil {
		return fmt.Errorf("failed to create project with name %s and key %s: %v", name, key, err)
	}

	// LaunchDarkly's api does not allow tags to be passed in during project creation so we do an update
	err = resourceProjectUpdate(d, metaRaw)
	if err != nil {
		return fmt.Errorf("failed to update project with name %s and key %s: %v", name, key, err)
	}

	d.SetId(key)
	return resourceProjectRead(d, metaRaw)
}

func resourceProjectRead(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	key := d.Get("key").(string)

	project, _, err := client.LaunchDarkly.ProjectsApi.GetProject(client.Ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get project with key %q: %v", key, err)
	}

	d.Set("environments", environmentsToResourceData(project.Environments))
	d.Set("key", project.Key)
	d.Set("name", project.Name)
	d.Set("tags", project.Tags)
	return nil
}

func resourceProjectUpdate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	key := d.Get("key").(string)
	name := d.Get("name")
	tags := stringSetFromResourceData(d, "tags")

	patch := []ldapi.PatchOperation{
		patchReplace("/name", &name),
		patchReplace("/tags", &tags),
	}

	_, _, err := client.LaunchDarkly.ProjectsApi.PatchProject(client.Ctx, key, patch)
	if err != nil {
		return fmt.Errorf("failed to update project with key %q: %s", key, err)
	}

	return resourceProjectRead(d, metaRaw)
}

func resourceProjectDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	key := d.Get("key").(string)

	_, err := client.LaunchDarkly.ProjectsApi.DeleteProject(client.Ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete project with key %q: %s", key, err)
	}

	return nil
}

func resourceProjectExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return projectExists(d.Get("key").(string), metaRaw.(*Client))
}

func projectExists(key string, meta *Client) (bool, error) {
	_, httpResponse, err := meta.LaunchDarkly.ProjectsApi.GetProject(meta.Ctx, key)
	if httpResponse != nil && httpResponse.StatusCode == 404 {
		fmt.Println("got 404 when getting project. returning false.")
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get project with key %q: %v", key, err)
	}

	return true, nil
}

func resourceProjectImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	d.SetId(d.Id())
	d.Set("key", d.Id())

	if err := resourceProjectRead(d, meta); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}
