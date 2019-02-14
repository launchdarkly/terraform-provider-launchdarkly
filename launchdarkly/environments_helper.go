package launchdarkly

import (
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
)

// used when creating a project.
func environmentsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Set:      environmentsSchemaSetFunc,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				// TODO: enable same fields as environment post for consistency.
				//  Will require updates to each env after project creation.
				"name": {
					Type:     schema.TypeString,
					Required: true,
				},
				"key": {
					Type:     schema.TypeString,
					Required: true,
				},
				"color": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"default_ttl": {
					Type:     schema.TypeFloat,
					Optional: true,
				},
			},
		},
	}
}

func environmentsSetFromResourceData(d *schema.ResourceData) []ldapi.EnvironmentPost {
	schemaEnvs := d.Get("environments").(*schema.Set)

	envs := make([]ldapi.EnvironmentPost, schemaEnvs.Len())
	for i, env := range schemaEnvs.List() {
		envs[i] = environmentFromResourceData(env)
	}
	return envs
}

func environmentFromResourceData(env interface{}) ldapi.EnvironmentPost {
	envMap := env.(map[string]interface{})
	return ldapi.EnvironmentPost{
		Name:       envMap["name"].(string),
		Key:        envMap["key"].(string),
		Color:      envMap["color"].(string),
		DefaultTtl: float32(envMap["default_ttl"].(float64)),
	}
}

func environmentsToResourceData(envs []ldapi.Environment) interface{} {
	transformed := make([]interface{}, len(envs))

	for i, variation := range envs {
		transformed[i] = map[string]interface{}{
			"name":        variation.Name,
			"key":         variation.Key,
			"color":       variation.Color,
			"default_ttl": variation.DefaultTtl,
		}
	}
	return transformed
}

// https://godoc.org/github.com/hashicorp/terraform/helper/schema#SchemaSetFunc
func environmentsSchemaSetFunc(value interface{}) int {
	return hashcode.String(environmentFromResourceData(value).Key)
}
