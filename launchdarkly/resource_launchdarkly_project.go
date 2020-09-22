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
			State: resourceProjectImport,
		},

		Schema: map[string]*schema.Schema{
			KEY: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateKey(),
			},
			NAME: {
				Type:     schema.TypeString,
				Required: true,
			},
			INCLUDE_IN_SNIPPET: {
				Type:     schema.TypeBool,
				Optional: true,
			},
			TAGS: tagsSchema(),
			ENVIRONMENTS: {
				Type:     schema.TypeList,
				Optional: true,
				Computed: false,
				Elem: &schema.Resource{
					Schema: environmentSchema(true),
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

	_, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.ProjectsApi.PostProject(client.ctx, projectBody)
	})
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
	return projectRead(d, metaRaw, false)
}

func resourceProjectUpdate(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(KEY).(string)
	projName := d.Get(NAME)
	projTags := stringsFromResourceData(d, TAGS)
	includeInSnippet := d.Get(INCLUDE_IN_SNIPPET)

	patch := []ldapi.PatchOperation{
		patchReplace("/name", &projName),
		patchReplace("/tags", &projTags),
		patchReplace("/includeInSnippetByDefault", includeInSnippet),
	}

	_, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return handleNoConflict(func() (interface{}, *http.Response, error) {
			return client.ld.ProjectsApi.PatchProject(client.ctx, projectKey, patch)
		})
	})
	if err != nil {
		return fmt.Errorf("failed to update project with key %q: %s", projectKey, handleLdapiErr(err))
	}
	// Update environments if necessary
	schemaEnvList, environmentsFound := d.GetOk(ENVIRONMENTS)
	if environmentsFound {
		// Get the project so we can see if we need to create any environments or just update existing environments
		rawProject, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
			return client.ld.ProjectsApi.GetProject(client.ctx, projectKey)
		})
		if err != nil {
			return fmt.Errorf("failed to load project %q before updating environments: %s", projectKey, handleLdapiErr(err))
		}
		project := rawProject.(ldapi.Project)

		environmentConfigs := schemaEnvList.([]interface{})
		for _, env := range environmentConfigs {
			envConfig := env.(map[string]interface{})
			envKey := envConfig[KEY].(string)
			// Check if the environment already exists. If it does not exist, create it
			exists := environmentExistsInProject(project, envKey)
			if !exists {
				envPost := environmentPostFromResourceData(env)
				_, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
					return client.ld.EnvironmentsApi.PostEnvironment(client.ctx, projectKey, envPost)
				})
				if err != nil {
					return fmt.Errorf("failed to create environment %q in project %q: %s", envKey, projectKey, handleLdapiErr(err))
				}
			}

			patches := getEnvironmentUpdatePatches(envConfig)
			_, _, err = handleRateLimit(func() (interface{}, *http.Response, error) {
				return handleNoConflict(func() (interface{}, *http.Response, error) {
					return client.ld.EnvironmentsApi.PatchEnvironment(client.ctx, projectKey, envKey, patches)
				})
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

	_, _, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		res, err := client.ld.ProjectsApi.DeleteProject(client.ctx, projectKey)
		return nil, res, err
	})

	if err != nil {
		return fmt.Errorf("failed to delete project with key %q: %s", projectKey, handleLdapiErr(err))
	}

	return nil
}

func resourceProjectExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return projectExists(d.Get(KEY).(string), metaRaw.(*Client))
}

func projectExists(projectKey string, meta *Client) (bool, error) {
	_, res, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return meta.ld.ProjectsApi.GetProject(meta.ctx, projectKey)
	})
	if isStatusNotFound(res) {
		log.Println("got 404 when getting project. returning false.")
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get project with key %q: %v", projectKey, handleLdapiErr(err))
	}

	return true, nil
}

func resourceProjectImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	_ = d.Set(KEY, d.Id())

	return []*schema.ResourceData{d}, nil
}
