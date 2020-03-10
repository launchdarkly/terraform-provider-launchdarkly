package launchdarkly

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go"
)

func environmentSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
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
		COLOR: &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
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
