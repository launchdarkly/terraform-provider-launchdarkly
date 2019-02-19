package launchdarkly

import (
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/launchdarkly/api-client-go"
)

func environmentSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		key: &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
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
		color: &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
		},
		default_ttl: &schema.Schema{
			Type:     schema.TypeFloat,
			Optional: true,
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
	schemaEnvs := d.Get(environments).(*schema.Set)

	envs := make([]ldapi.EnvironmentPost, schemaEnvs.Len())
	for i, env := range schemaEnvs.List() {
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

	if defaultTtl, ok := envMap[default_ttl]; ok {
		// What's up with the type assertions/casting?
		// 1. terraform stores the value as an untyped float64, so we coerce float64 from the interface{}
		// 2. ld api expects a float32 so we cast it to float32
		envPost.DefaultTtl = float32(defaultTtl.(float64))
	}
	return envPost
}

func environmentsToResourceData(envs []ldapi.Environment) interface{} {
	transformed := make([]interface{}, len(envs))

	for i, env := range envs {
		transformed[i] = map[string]interface{}{
			key:                  env.Key,
			name:                 env.Name,
			api_key:              env.ApiKey,
			mobile_key:           env.MobileKey,
			color:                env.Color,
			default_ttl:          env.DefaultTtl,
			secure_mode:          env.SecureMode,
			default_track_events: env.DefaultTrackEvents,
			tags:                 env.Tags,
		}
	}
	return transformed
}

// https://godoc.org/github.com/hashicorp/terraform/helper/schema#SchemaSetFunc
func environmentHash(value interface{}) int {
	return hashcode.String(environmentPostFromResourceData(value).Key)
}
