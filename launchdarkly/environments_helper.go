package launchdarkly

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go"
)

func environmentSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		key: &schema.Schema{
			Type:         schema.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: validateKey(),
		},
		name: &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
		},
		api_key: &schema.Schema{
			Type:      schema.TypeString,
			Computed:  true,
			Sensitive: true,
		},
		mobile_key: &schema.Schema{
			Type:      schema.TypeString,
			Computed:  true,
			Sensitive: true,
		},
		client_side_id: {
			Type:      schema.TypeString,
			Computed:  true,
			Sensitive: true,
		},
		color: &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
		},
		default_ttl: &schema.Schema{
			Type:     schema.TypeInt,
			Optional: true,
			// Default TTL should be between 0 and 60 minutes: https://docs.launchdarkly.com/docs/environments
			ValidateFunc: validation.IntBetween(0, 60),
		},
		secure_mode: &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
		},
		default_track_events: &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
		},
		tags: tagsSchema(),
	}
}

func environmentPostsFromResourceData(d *schema.ResourceData) []ldapi.EnvironmentPost {
	schemaEnvs := d.Get(environments).([]interface{})

	envs := make([]ldapi.EnvironmentPost, len(schemaEnvs))
	for i, env := range schemaEnvs {
		envs[i] = environmentPostFromResourceData(env)
	}
	return envs
}

func environmentPostFromResourceData(env interface{}) ldapi.EnvironmentPost {
	envMap := env.(map[string]interface{})
	envPost := ldapi.EnvironmentPost{
		Name:  envMap[name].(string),
		Key:   envMap[key].(string),
		Color: envMap[color].(string),
	}

	if defaultTTL, ok := envMap[default_ttl]; ok {
		envPost.DefaultTtl = float32(defaultTTL.(int))
	}
	return envPost
}

func environmentsToResourceData(envs []ldapi.Environment) []interface{} {
	transformed := make([]interface{}, len(envs))

	for i, env := range envs {
		transformed[i] = map[string]interface{}{
			key:                  env.Key,
			name:                 env.Name,
			api_key:              env.ApiKey,
			mobile_key:           env.MobileKey,
			client_side_id:       env.Id,
			color:                env.Color,
			default_ttl:          int(env.DefaultTtl),
			secure_mode:          env.SecureMode,
			default_track_events: env.DefaultTrackEvents,
			tags:                 env.Tags,
		}
	}
	return transformed
}
