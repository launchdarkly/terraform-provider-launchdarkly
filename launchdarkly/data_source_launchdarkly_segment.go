package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSegment() *schema.Resource {
	schemaMap := baseSegmentSchema(segmentSchemaOptions{isDataSource: true})
	schemaMap = removeInvalidFieldsForDataSource(schemaMap)
	schemaMap[PROJECT_KEY] = &schema.Schema{
		Type:             schema.TypeString,
		Required:         true,
		ValidateDiagFunc: validateKey(),
		Description:      "The segment's project key.",
	}
	schemaMap[ENV_KEY] = &schema.Schema{
		Type:             schema.TypeString,
		Required:         true,
		ValidateDiagFunc: validateKey(),
		Description:      "The segment's environment key.",
	}
	schemaMap[KEY] = &schema.Schema{
		Type:             schema.TypeString,
		Required:         true,
		ValidateDiagFunc: validateKey(),
		Description:      "The unique key that references the segment.",
	}
	schemaMap[NAME] = &schema.Schema{
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The human-friendly name for the segment.",
	}
	schemaMap[VIEWS] = &schema.Schema{
		Type:        schema.TypeList,
		Computed:    true,
		Description: "A list of view keys that this segment is linked to.",
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
	}
	return &schema.Resource{
		ReadContext: dataSourceSegmentRead,
		Schema:      schemaMap,

		Description: `Provides a LaunchDarkly segment data source.

This data source allows you to retrieve segment information from your LaunchDarkly organization.`,
	}
}

func dataSourceSegmentRead(ctx context.Context, d *schema.ResourceData, raw interface{}) diag.Diagnostics {
	return segmentRead(ctx, d, raw, true)
}
