package launchdarkly

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceFeatureFlag() *schema.Resource {
	schemaMap := baseFeatureFlagSchema()
	schemaMap[NAME] = &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The feature flag's description",
	}
	schemaMap[VARIATION_TYPE] = &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
		ForceNew: true,
		Description: fmt.Sprintf("The uniform type for all variations. Can be either %q, %q, %q, or %q.",
			BOOL_VARIATION, STRING_VARIATION, NUMBER_VARIATION, JSON_VARIATION),
		ValidateFunc: validateVariationType,
	}
	return &schema.Resource{
		Read:   dataSourceFeatureFlagRead,
		Schema: schemaMap,
	}
}

func dataSourceFeatureFlagRead(d *schema.ResourceData, raw interface{}) error {
	return featureFlagRead(d, raw, true)
}
