package launchdarkly

import "github.com/hashicorp/terraform-plugin-sdk/helper/schema"

func dataSourceFeatureFlagEnvironment() *schema.Resource {
	return &schema.Resource{
		Read:   featureFlagEnvironmentRead,
		Schema: baseFeatureFlagEnvironmentSchema(),
	}
}
