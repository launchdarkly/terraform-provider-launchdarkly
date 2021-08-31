package launchdarkly

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func dataSourceFeatureFlagEnvironment() *schema.Resource {
	return &schema.Resource{
		Read:   dataSourceFeatureFlagEnvironmentRead,
		Schema: baseFeatureFlagEnvironmentSchema(true),
	}
}

func dataSourceFeatureFlagEnvironmentRead(d *schema.ResourceData, meta interface{}) error {
	return featureFlagEnvironmentRead(d, meta, true)
}
