package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSegment() *schema.Resource {
	schemaMap := baseSegmentSchema(segmentSchemaOptions{isDataSource: true})
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
	return &schema.Resource{
		ReadContext: dataSourceSegmentRead,
		Schema:      schemaMap,
	}
}

func dataSourceSegmentRead(ctx context.Context, d *schema.ResourceData, raw interface{}) diag.Diagnostics {
	return segmentRead(ctx, d, raw, true)
}
