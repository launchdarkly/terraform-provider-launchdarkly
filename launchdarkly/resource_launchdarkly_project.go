package launchdarkly

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v7"
)

// We assign a custom diff in cases where the customer has not assigned a default for CSA or IIS in config
//  in order to respect the LD backend defaults and reflect that in our plans
func customizeProjectDiff(ctx context.Context, diff *schema.ResourceDiff, v interface{}) error {
	config := diff.GetRawConfig()

	// Below values will exist due to the schema, we need to check if they are all null
	snippetInConfig := config.GetAttr(INCLUDE_IN_SNIPPET)
	csaInConfig := config.GetAttr(DEFAULT_CLIENT_SIDE_AVAILABILITY)

	// If we have no keys in the CSA block in the config (length is 0) we know the customer hasn't set any CSA values
	csaKeys := csaInConfig.AsValueSlice()
	if len(csaKeys) == 0 {
		// When we have no values for either clienSideAvailability or includeInSnippet
		// Force an UPDATE call by setting a new value for INCLUDE_IN_SNIPPET in the diff according to project defaults
		if snippetInConfig.IsNull() {
			// We set our values to the LD backend defaults in order to guarantee an update call happening
			// If we don't do this, we can run into an edge case described below
			// IF previous value of INCLUDE_IN_SNIPPET was false
			// AND the project default value for INCLUDE_IN_SNIPPET is true
			// AND the customer removes the INCLUDE_IN_SNIPPET key from the config without replacing with defaultCSA
			// The read would assume no changes are needed, HOWEVER we need to jump back to LD set defaults
			// Hence the setting below
			diff.SetNew(INCLUDE_IN_SNIPPET, false)
			diff.SetNew(CLIENT_SIDE_AVAILABILITY, []map[string]interface{}{{
				USING_ENVIRONMENT_ID: false,
				USING_MOBILE_KEY:     true,
			}})

		}

	}

	return nil
}
func resourceProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceProjectCreate,
		Read:   resourceProjectRead,
		Update: resourceProjectUpdate,
		Delete: resourceProjectDelete,
		Exists: resourceProjectExists,

		CustomizeDiff: customizeProjectDiff,

		Importer: &schema.ResourceImporter{
			StateContext: resourceProjectImport,
		},

		Schema: map[string]*schema.Schema{
			KEY: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The project's unique key",
				ForceNew:     true,
				ValidateFunc: validateKeyAndLength(1, 20),
			},
			NAME: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "A human-readable name for your project",
			},
			INCLUDE_IN_SNIPPET: {
				Type:          schema.TypeBool,
				Optional:      true,
				Description:   "Whether feature flags created under the project should be available to client-side SDKs by default",
				Computed:      true,
				Deprecated:    "'include_in_snippet' is now deprecated. Please migrate to 'default_client_side_availability' to maintain future compatability.",
				ConflictsWith: []string{DEFAULT_CLIENT_SIDE_AVAILABILITY},
			},
			DEFAULT_CLIENT_SIDE_AVAILABILITY: {
				Type:     schema.TypeList,
				Optional: true,
				// Can't set defaults for lists/sets :( https://github.com/hashicorp/terraform-plugin-sdk/issues/142
				// Since we can't set defaults, we run into misleading plans when users remove this attribute from their config
				// As the plan output suggests the values will be changed to -> null, when we actually have LD set defaults of false and true respectively
				// Sorting that by using Computed for now
				Computed:      true,
				Description:   "List determining which SDKs have access to new flags created under the project by default",
				ConflictsWith: []string{INCLUDE_IN_SNIPPET},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						USING_ENVIRONMENT_ID: {
							Type:     schema.TypeBool,
							Required: true,
						},
						USING_MOBILE_KEY: {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
			},
			TAGS: tagsSchema(),
			ENVIRONMENTS: {
				Type:        schema.TypeList,
				Required:    true,
				Description: "List of nested `environments` blocks describing LaunchDarkly environments that belong to the project",
				Computed:    false,
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
	projectBody := ldapi.ProjectPost{
		Name: name,
		Key:  projectKey,
	}

	if len(envs) > 0 {
		projectBody.Environments = &envs
	}

	_, _, err := client.ld.ProjectsApi.PostProject(client.ctx).ProjectPost(projectBody).Execute()
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
	includeInSnippet := d.Get(INCLUDE_IN_SNIPPET).(bool)

	snippetHasChange := d.HasChange(INCLUDE_IN_SNIPPET)
	clientSideHasChange := d.HasChange(DEFAULT_CLIENT_SIDE_AVAILABILITY)
	// GetOkExists is 'deprecated', but needed as optional booleans set to false return a 'false' ok value from GetOk
	// Also not really deprecated as they are keeping it around pending a replacement https://github.com/hashicorp/terraform-plugin-sdk/pull/350#issuecomment-597888969
	_, includeInSnippetOk := d.GetOkExists(INCLUDE_IN_SNIPPET)
	_, clientSideAvailabilityOk := d.GetOk(DEFAULT_CLIENT_SIDE_AVAILABILITY)
	defaultClientSideAvailability := &ldapi.ClientSideAvailabilityPost{
		UsingEnvironmentId: d.Get(fmt.Sprintf("%s.0.using_environment_id", DEFAULT_CLIENT_SIDE_AVAILABILITY)).(bool),
		UsingMobileKey:     d.Get(fmt.Sprintf("%s.0.using_mobile_key", DEFAULT_CLIENT_SIDE_AVAILABILITY)).(bool),
	}

	patch := []ldapi.PatchOperation{
		patchReplace("/name", &projName),
		patchReplace("/tags", &projTags),
	}

	if clientSideAvailabilityOk && clientSideHasChange {
		patch = append(patch, patchReplace("/defaultClientSideAvailability", defaultClientSideAvailability))
	} else if includeInSnippetOk && snippetHasChange {
		// If includeInSnippet is set, still use clientSideAvailability behind the scenes in order to switch UsingMobileKey to false if needed
		patch = append(patch, patchReplace("/defaultClientSideAvailability", &ldapi.ClientSideAvailabilityPost{
			UsingEnvironmentId: includeInSnippet,
			UsingMobileKey:     true,
		}))
	} else {
		// If the user doesn't set either CSA or IIS in config, we set defaults to match API behaviour
		patch = append(patch, patchReplace("/defaultClientSideAvailability", &ldapi.ClientSideAvailabilityPost{
			UsingEnvironmentId: false,
			UsingMobileKey:     true,
		}))
	}

	_, _, err := client.ld.ProjectsApi.PatchProject(client.ctx, projectKey).PatchOperation(patch).Execute()
	if err != nil {
		return fmt.Errorf("failed to update project with key %q: %s", projectKey, handleLdapiErr(err))
	}
	// Update environments if necessary
	oldSchemaEnvList, newSchemaEnvList := d.GetChange(ENVIRONMENTS)
	// Get the project so we can see if we need to create any environments or just update existing environments
	project, _, err := client.ld.ProjectsApi.GetProject(client.ctx, projectKey).Execute()
	if err != nil {
		return fmt.Errorf("failed to load project %q before updating environments: %s", projectKey, handleLdapiErr(err))
	}

	environmentConfigs := newSchemaEnvList.([]interface{})
	oldEnvironmentConfigs := oldSchemaEnvList.([]interface{})
	var oldEnvConfigsForCompare = make(map[string]map[string]interface{}, len(oldEnvironmentConfigs))
	for _, env := range oldEnvironmentConfigs {
		envConfig := env.(map[string]interface{})
		envKey := envConfig[KEY].(string)
		oldEnvConfigsForCompare[envKey] = envConfig
	}
	// save envs in a key:config map so we can more easily figure out which need to be patchRemoved after
	var envConfigsForCompare = make(map[string]map[string]interface{}, len(environmentConfigs))
	for _, env := range environmentConfigs {

		envConfig := env.(map[string]interface{})
		envKey := envConfig[KEY].(string)
		envConfigsForCompare[envKey] = envConfig
		// Check if the environment already exists. If it does not exist, create it
		exists := environmentExistsInProject(project, envKey)
		if !exists {
			envPost := environmentPostFromResourceData(env)
			_, _, err := client.ld.EnvironmentsApi.PostEnvironment(client.ctx, projectKey).EnvironmentPost(envPost).Execute()
			if err != nil {
				return fmt.Errorf("failed to create environment %q in project %q: %s", envKey, projectKey, handleLdapiErr(err))
			}
		}

		var oldEnvConfig map[string]interface{}
		if rawOldConfig, ok := oldEnvConfigsForCompare[envKey]; ok {
			oldEnvConfig = rawOldConfig
		}
		// by default patching an env that was not recently tracked in the state will import it into the tf state
		patch, err := getEnvironmentUpdatePatches(oldEnvConfig, envConfig)
		if err != nil {
			return err
		}
		_, _, err = client.ld.EnvironmentsApi.PatchEnvironment(client.ctx, projectKey, envKey).PatchOperation(patch).Execute()
		if err != nil {
			return fmt.Errorf("failed to update environment with key %q for project: %q: %+v", envKey, projectKey, err)
		}
	}
	// we also want to delete environments that were previously tracked in state and have been removed from the config
	old, _ := d.GetChange(ENVIRONMENTS)
	oldEnvs := old.([]interface{})
	for _, env := range oldEnvs {
		envConfig := env.(map[string]interface{})
		envKey := envConfig[KEY].(string)
		if _, persists := envConfigsForCompare[envKey]; !persists {
			_, err = client.ld.EnvironmentsApi.DeleteEnvironment(client.ctx, projectKey, envKey).Execute()
			if err != nil {
				return fmt.Errorf("failed to delete environment %q in project %q: %s", envKey, projectKey, handleLdapiErr(err))
			}
		}
	}

	return resourceProjectRead(d, metaRaw)
}

func resourceProjectDelete(d *schema.ResourceData, metaRaw interface{}) error {
	client := metaRaw.(*Client)
	projectKey := d.Get(KEY).(string)

	_, err := client.ld.ProjectsApi.DeleteProject(client.ctx, projectKey).Execute()
	if err != nil {
		return fmt.Errorf("failed to delete project with key %q: %s", projectKey, handleLdapiErr(err))
	}

	return nil
}

func resourceProjectExists(d *schema.ResourceData, metaRaw interface{}) (bool, error) {
	return projectExists(d.Get(KEY).(string), metaRaw.(*Client))
}

func projectExists(projectKey string, meta *Client) (bool, error) {
	_, res, err := meta.ld.ProjectsApi.GetProject(meta.ctx, projectKey).Execute()
	if isStatusNotFound(res) {
		log.Println("got 404 when getting project. returning false.")
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get project with key %q: %v", projectKey, handleLdapiErr(err))
	}

	return true, nil
}

func resourceProjectImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	_ = d.Set(KEY, d.Id())

	return []*schema.ResourceData{d}, nil
}
