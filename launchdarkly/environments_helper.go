package launchdarkly

import (
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go"
)

// baseEnvironmentSchema covers the overlap between the data source and resource schemas
// certain attributes are required for the resource that are not for the data source and so those
// will need to be differentiated
func baseEnvironmentSchema(forProject bool) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		KEY: {
			Type:     schema.TypeString,
			Required: true,
			// Don't force new if the environment schema will be nested in a project
			ForceNew:     !forProject,
			ValidateFunc: validateKey(),
		},
		API_KEY: {
			Type:      schema.TypeString,
			Computed:  true,
			Sensitive: true,
		},
		MOBILE_KEY: {
			Type:      schema.TypeString,
			Computed:  true,
			Sensitive: true,
		},
		CLIENT_SIDE_ID: {
			Type:      schema.TypeString,
			Computed:  true,
			Sensitive: true,
		},
		DEFAULT_TTL: {
			Type:     schema.TypeInt,
			Optional: true,
			Computed: true,
			// Default TTL should be between 0 and 60 minutes: https://docs.launchdarkly.com/docs/environments
			ValidateFunc: validation.IntBetween(0, 60),
			Description:  "The TTL for the environment. This must be between 0 and 60 minutes. The TTL setting only applies to environments using the PHP SDK.",
		},
		SECURE_MODE: {
			Computed:    true,
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Secure mode ensures a user of the client-side SDK cannot impersonate another user",
		},
		DEFAULT_TRACK_EVENTS: {
			Computed:    true,
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether or not to send data export events for every flag created in the environment",
		},
		REQUIRE_COMMENTS: {
			Computed:    true,
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether or not to require comments for a flag and segment changes in this environment",
		},
		CONFIRM_CHANGES: {
			Computed:    true,
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Whether or not to require confirmation for a flag and segment changes in this environment",
		},
		TAGS: tagsSchema(),
	}
}

func getEnvironmentUpdatePatches(config map[string]interface{}) []ldapi.PatchOperation {
	// Always include required fields
	name := config[NAME]
	color := config[COLOR]
	patches := []ldapi.PatchOperation{
		patchReplace("/name", name),
		patchReplace("/color", color),
	}

	// Add optional fields if they exist
	defaultTtl, ok := config[DEFAULT_TTL]
	if ok {
		patches = append(patches, patchReplace("/defaultTtl", defaultTtl))
	}

	secureMode, ok := config[SECURE_MODE]
	if ok {
		patches = append(patches, patchReplace("/secureMode", secureMode))
	}

	defaultTrackEvents, ok := config[DEFAULT_TRACK_EVENTS]
	if ok {
		patches = append(patches, patchReplace("/defaultTrackEvents", defaultTrackEvents))
	}

	requireComments, ok := config[REQUIRE_COMMENTS]
	if ok {
		patches = append(patches, patchReplace("/requireComments", requireComments))
	}

	confirmChanges, ok := config[CONFIRM_CHANGES]
	if ok {
		patches = append(patches, patchReplace("/confirmChanges", confirmChanges))
	}

	tags, ok := config[TAGS]
	if ok {
		envTags := stringsFromSchemaSet(tags.(*schema.Set))
		patches = append(patches, patchReplace("/tags", &envTags))
	}
	return patches
}

func environmentSchema(forProject bool) map[string]*schema.Schema {
	schemaMap := baseEnvironmentSchema(forProject)
	schemaMap[NAME] = &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
	}
	schemaMap[COLOR] = &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
	}
	return schemaMap
}

func dataSourceEnvironmentSchema(forPoject bool) map[string]*schema.Schema {
	schemaMap := baseEnvironmentSchema(forPoject)
	schemaMap[NAME] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	}
	schemaMap[COLOR] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	}
	return schemaMap
}

func environmentPostsFromResourceData(d *schema.ResourceData) []ldapi.EnvironmentPost {
	schemaEnvList := d.Get(ENVIRONMENTS).([]interface{})
	envs := make([]ldapi.EnvironmentPost, len(schemaEnvList))
	for i, env := range schemaEnvList {
		envs[i] = environmentPostFromResourceData(env)
	}
	return envs
}

func environmentPostFromResourceData(env interface{}) ldapi.EnvironmentPost {
	envMap := env.(map[string]interface{})
	envPost := ldapi.EnvironmentPost{
		Name:  envMap[NAME].(string),
		Key:   envMap[KEY].(string),
		Color: envMap[COLOR].(string),
	}

	if defaultTTL, ok := envMap[DEFAULT_TTL]; ok {
		envPost.DefaultTtl = float32(defaultTTL.(int))
	}
	return envPost
}

type envResourceData map[string]interface{}

func environmentsToResourceDataMap(envs []ldapi.Environment) map[string]envResourceData {
	envMap := make(map[string]envResourceData, len(envs))
	for _, env := range envs {
		envMap[env.Key] = environmentToResourceData(env)
	}

	return envMap
}

func environmentToResourceData(env ldapi.Environment) envResourceData {
	return envResourceData{
		KEY:                  env.Key,
		NAME:                 env.Name,
		API_KEY:              env.ApiKey,
		MOBILE_KEY:           env.MobileKey,
		CLIENT_SIDE_ID:       env.Id,
		COLOR:                env.Color,
		DEFAULT_TTL:          int(env.DefaultTtl),
		SECURE_MODE:          env.SecureMode,
		DEFAULT_TRACK_EVENTS: env.DefaultTrackEvents,
		REQUIRE_COMMENTS:     env.RequireComments,
		CONFIRM_CHANGES:      env.ConfirmChanges,
		TAGS:                 env.Tags,
	}
}

func rawEnvironmentConfigsToKeyList(rawEnvs []interface{}) []string {
	keys := make([]string, 0, len(rawEnvs))
	for _, rawEnv := range rawEnvs {
		env := rawEnv.(map[string]interface{})
		envKey := env[KEY].(string)
		keys = append(keys, envKey)
	}
	return keys
}

func environmentRead(d *schema.ResourceData, meta interface{}, isDataSource bool) error {
	client := meta.(*Client)
	projectKey := d.Get(PROJECT_KEY).(string)
	key := d.Get(KEY).(string)

	envRaw, res, err := handleRateLimit(func() (interface{}, *http.Response, error) {
		return client.ld.EnvironmentsApi.GetEnvironment(client.ctx, projectKey, key)
	})
	if isStatusNotFound(res) && !isDataSource {
		log.Printf("[WARN] failed to find environment with key %q in project %q, removing from state", key, projectKey)
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get environment with key %q for project key: %q: %v", key, projectKey, handleLdapiErr(err))
	}

	env := envRaw.(ldapi.Environment)
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
