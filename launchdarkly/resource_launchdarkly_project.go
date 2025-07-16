package launchdarkly

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v17"
)

// We assign a custom diff in cases where the customer has not assigned a default for CSA or IIS in config
// in order to respect the LD backend defaults and reflect that in our plans
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
			err := diff.SetNew(INCLUDE_IN_SNIPPET, false)
			if err != nil {
				return err
			}
			err = diff.SetNew(DEFAULT_CLIENT_SIDE_AVAILABILITY, []map[string]interface{}{{
				USING_ENVIRONMENT_ID: false,
				USING_MOBILE_KEY:     true,
			}})
			if err != nil {
				return err
			}

		}

	}

	return nil
}
func resourceProject() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceProjectCreate,
		ReadContext:   resourceProjectRead,
		UpdateContext: resourceProjectUpdate,
		DeleteContext: resourceProjectDelete,
		Exists:        resourceProjectExists,

		CustomizeDiff: customizeProjectDiff,

		Importer: &schema.ResourceImporter{
			StateContext: resourceProjectImport,
		},

		Description: `Provides a LaunchDarkly project resource.

This resource allows you to create and manage projects within your LaunchDarkly organization.`,

		Schema: map[string]*schema.Schema{
			KEY: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      addForceNewDescription("The project's unique key.", true),
				ForceNew:         true,
				ValidateDiagFunc: validateKeyAndLength(1, 100),
			},
			NAME: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The project's name.",
			},
			INCLUDE_IN_SNIPPET: {
				Type:          schema.TypeBool,
				Optional:      true,
				Description:   "Whether feature flags created under the project should be available to client-side SDKs by default. Please migrate to `default_client_side_availability` to maintain future compatibility.",
				Computed:      true,
				Deprecated:    "'include_in_snippet' is now deprecated. Please migrate to 'default_client_side_availability' to maintain future compatibility.",
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
				Description:   "A block describing which client-side SDKs can use new flags by default.",
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
			TAGS: tagsSchema(tagsSchemaOptions{isDataSource: false}),
			ENVIRONMENTS: {
				Type:        schema.TypeList,
				Required:    true,
				Description: "List of nested `environments` blocks describing LaunchDarkly environments that belong to the project. When managing LaunchDarkly projects in Terraform, you should always manage your environments as nested project resources.\n\n-> **Note:** Mixing the use of nested `environments` blocks and [`launchdarkly_environment`](/docs/providers/launchdarkly/r/environment.html) resources is not recommended. `launchdarkly_environment` resources should only be used when the encapsulating project is not managed in Terraform.",
				Computed:    false,
				Elem: &schema.Resource{
					Schema: environmentSchema(environmentSchemaOptions{forProject: true, isDataSource: false}),
				},
			},
		},
	}
}

func resourceProjectCreate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
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
		projectBody.Environments = envs
	}

	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, _, err = client.ld.ProjectsApi.PostProject(client.ctx).ProjectPost(projectBody).Execute()
		return err
	})
	if err != nil {
		if !isTimeoutError(err) {
			return diag.Errorf("failed to create project with name %s and projectKey %s: %v", name, projectKey, handleLdapiErr(err))
		}
		fmt.Printf("[DEBUG] Network timeout when making the API call to create project %q. This can happen when there are 20+ environments. In most cases the Terraform apply will still succeed.\n", projectKey)
	}

	// ld's api does not allow tags to be passed in during project creation so we do an update
	updateDiags := resourceProjectUpdate(ctx, d, metaRaw)
	if updateDiags.HasError() {
		updateDiags = append(updateDiags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("failed to update project with name %s and projectKey %s: %v", name, projectKey, err),
		})
		return updateDiags
	}
	return diags
}

func resourceProjectRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return projectRead(ctx, d, metaRaw, false)
}

func resourceProjectUpdate(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	client := metaRaw.(*Client)
	projectKey := d.Get(KEY).(string)
	projName := d.Get(NAME)
	projTags := stringsFromResourceData(d, TAGS)
	includeInSnippet := d.Get(INCLUDE_IN_SNIPPET).(bool)

	snippetHasChange := d.HasChange(INCLUDE_IN_SNIPPET)
	clientSideHasChange := d.HasChange(DEFAULT_CLIENT_SIDE_AVAILABILITY)
	// GetOkExists is 'deprecated', but needed as optional booleans set to false return a 'false' ok value from GetOk
	// Also not really deprecated as they are keeping it around pending a replacement https://github.com/hashicorp/terraform-plugin-sdk/pull/350#issuecomment-597888969
	//nolint:staticcheck // SA1019
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

	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, _, err = client.ld.ProjectsApi.PatchProject(client.ctx, projectKey).PatchOperation(patch).Execute()
		return err
	})
	if err != nil {
		return diag.Errorf("failed to update project with key %q: %s", projectKey, handleLdapiErr(err))
	}
	// Update environments if necessary
	oldSchemaEnvList, newSchemaEnvList := d.GetChange(ENVIRONMENTS)
	// Get the project so we can see if we need to create any environments or just update existing environments
	project, _, err := getFullProject(client, projectKey)
	if err != nil {
		return diag.Errorf("failed to load project %q before updating environments: %s", projectKey, handleLdapiErr(err))
	}

	environmentConfigs := newSchemaEnvList.([]interface{})
	oldEnvironmentConfigs := oldSchemaEnvList.([]interface{})
	oldEnvConfigsForCompare := make(map[string]map[string]interface{}, len(oldEnvironmentConfigs))
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
		exists := environmentExistsInProject(*project, envKey)
		if !exists {
			envPost := environmentPostFromResourceData(env)
			var err error
			err = client.withConcurrency(client.ctx, func() error {
				_, _, err = client.ld.EnvironmentsApi.PostEnvironment(client.ctx, projectKey).EnvironmentPost(envPost).Execute()
				return err
			})
			if err != nil {
				return diag.Errorf("failed to create environment %q in project %q: %s", envKey, projectKey, handleLdapiErr(err))
			}
		}

		var oldEnvConfig map[string]interface{}
		if rawOldConfig, ok := oldEnvConfigsForCompare[envKey]; ok {
			oldEnvConfig = rawOldConfig
		}
		// by default patching an env that was not recently tracked in the state will import it into the tf state
		patch, err := getEnvironmentUpdatePatches(oldEnvConfig, envConfig)
		if err != nil {
			return diag.FromErr(err)
		}
		err = client.withConcurrency(client.ctx, func() error {
			_, _, err = client.ld.EnvironmentsApi.PatchEnvironment(client.ctx, projectKey, envKey).PatchOperation(patch).Execute()
			return err
		})
		if err != nil {
			return diag.Errorf("failed to update project environment with key %q for project: %q: %+v", envKey, projectKey, handleLdapiErr(err))
		}
	}
	// we also want to delete environments that were previously tracked in state and have been removed from the config
	old, _ := d.GetChange(ENVIRONMENTS)
	oldEnvs := old.([]interface{})
	for _, env := range oldEnvs {
		envConfig := env.(map[string]interface{})
		envKey := envConfig[KEY].(string)
		if _, persists := envConfigsForCompare[envKey]; !persists {
			err = client.withConcurrency(client.ctx, func() error {
				_, err = client.ld.EnvironmentsApi.DeleteEnvironment(client.ctx, projectKey, envKey).Execute()
				return err
			})
			if err != nil {
				return diag.Errorf("failed to delete environment %q in project %q: %s", envKey, projectKey, handleLdapiErr(err))
			}
		}
	}

	return resourceProjectRead(ctx, d, metaRaw)
}

func resourceProjectDelete(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := metaRaw.(*Client)
	projectKey := d.Get(KEY).(string)

	var err error
	err = client.withConcurrency(client.ctx, func() error {
		_, err = client.ld.ProjectsApi.DeleteProject(client.ctx, projectKey).Execute()
		return err
	})
	if err != nil {
		if !isTimeoutError(err) {
			return diag.Errorf("failed to delete project with key %q: %s", projectKey, handleLdapiErr(err))
		}
		fmt.Printf("[DEBUG] Got a network timeout error when deleting project %q. This can happen when the project has 20+ environments.\n", projectKey)
	}

	return diags
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
