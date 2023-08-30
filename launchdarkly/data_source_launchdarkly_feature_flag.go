package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceFeatureFlag() *schema.Resource {
	schemaMap := baseFeatureFlagSchema(featureFlagSchemaOptions{isDataSource: true})
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
	return &schema.Resource{
		ReadContext: dataSourceFeatureFlagRead,
		Schema:      schemaMap,
		Description: `Provides a LaunchDarkly feature flag data source.

This data source allows you to retrieve feature flag information from your LaunchDarkly organization.`,
	}
}

func dataSourceFeatureFlagRead(ctx context.Context, d *schema.ResourceData, raw interface{}) diag.Diagnostics {
	return featureFlagRead(ctx, d, raw, true)
}
