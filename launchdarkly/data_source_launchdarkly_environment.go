package launchdarkly

import "github.com/hashicorp/terraform-plugin-sdk/helper/schema"

func dataSourceEnvironment() *schema.Resource {
	envSchema := dataSourceEnvironmentSchema(false)
	envSchema[PROJECT_KEY] = &schema.Schema{
		Type:         schema.TypeString,
		Optional:     true,
		ForceNew:     true,
		ValidateFunc: validateKey(),
	}
	return &schema.Resource{
		Read:   dataSourceEnvironmentRead,
		Schema: envSchema,
	}
}

func dataSourceEnvironmentRead(d *schema.ResourceData, meta interface{}) error {
	return environmentRead(d, meta, true)
}
