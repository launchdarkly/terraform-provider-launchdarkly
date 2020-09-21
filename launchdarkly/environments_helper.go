package launchdarkly

import (
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go"
)

// baseEnvironmentSchema covers the overlap between the data source and resource schemas
// certain attributes are required for the resource that are not for the data source and so those
// will need to be differentiated
func baseEnvironmentSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		KEY: &schema.Schema{
			Type:         schema.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: validateKey(),
		},
		API_KEY: &schema.Schema{
			Type:      schema.TypeString,
			Computed:  true,
			Sensitive: true,
		},
		MOBILE_KEY: &schema.Schema{
			Type:      schema.TypeString,
			Computed:  true,
			Sensitive: true,
		},
		CLIENT_SIDE_ID: {
			Type:      schema.TypeString,
			Computed:  true,
			Sensitive: true,
		},
		DEFAULT_TTL: &schema.Schema{
			Type:     schema.TypeInt,
			Optional: true,
			// Default TTL should be between 0 and 60 minutes: https://docs.launchdarkly.com/docs/environments
			ValidateFunc: validation.IntBetween(0, 60),
		},
		SECURE_MODE: &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
		},
		DEFAULT_TRACK_EVENTS: &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
		},
		REQUIRE_COMMENTS: &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
		},
		CONFIRM_CHANGES: &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
		},
		TAGS: tagsSchema(),
	}
}

func environmentSchema() map[string]*schema.Schema {
	schemaMap := baseEnvironmentSchema()
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

func dataSourceEnvironmentSchema() map[string]*schema.Schema {
	schemaMap := baseEnvironmentSchema()
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
	schemaEnvs := d.Get(ENVIRONMENTS).([]interface{})

	envs := make([]ldapi.EnvironmentPost, len(schemaEnvs))
	for i, env := range schemaEnvs {
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

func environmentsToResourceData(envs []ldapi.Environment) []interface{} {
	transformed := make([]interface{}, len(envs))

	for i, env := range envs {
		transformed[i] = map[string]interface{}{
			KEY:                  env.Key,
			NAME:                 env.Name,
			API_KEY:              env.ApiKey,
			MOBILE_KEY:           env.MobileKey,
			CLIENT_SIDE_ID:       env.Id,
			COLOR:                env.Color,
			DEFAULT_TTL:          int(env.DefaultTtl),
			SECURE_MODE:          env.SecureMode,
			DEFAULT_TRACK_EVENTS: env.DefaultTrackEvents,
			TAGS:                 env.Tags,
		}
	}
	return transformed
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
