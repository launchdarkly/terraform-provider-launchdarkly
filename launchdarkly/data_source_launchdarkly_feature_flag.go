package launchdarkly

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceFeatureFlag() *schema.Resource {
	schemaMap := baseFeatureFlagSchema()
	schemaMap[NAME] = &schema.Schema{
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The feature flag's human-readable name",
	}
	schemaMap[VARIATION_TYPE] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
		Description: fmt.Sprintf("The uniform type for all variations. Can be either %q, %q, %q, or %q.",
			BOOL_VARIATION, STRING_VARIATION, NUMBER_VARIATION, JSON_VARIATION),
	}
	schemaMap[CLIENT_SIDE_AVAILABILITY] = &schema.Schema{
		Type:     schema.TypeMap,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"using_environment_id": {
					Type:     schema.TypeBool,
					Optional: true,
				},
				"using_mobile_key": {
					Type:     schema.TypeBool,
					Optional: true,
				},
			},
		},
	}
	return &schema.Resource{
		Read:   dataSourceFeatureFlagRead,
		Schema: schemaMap,
	}
}

func dataSourceFeatureFlagRead(d *schema.ResourceData, raw interface{}) error {
	return featureFlagRead(d, raw, true)
}
