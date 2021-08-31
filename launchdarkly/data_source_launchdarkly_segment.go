package launchdarkly

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func dataSourceSegment() *schema.Resource {
	schemaMap := baseSegmentSchema()
	schemaMap[PROJECT_KEY] = &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validateKey(),
		Description:  "The segment's project key.",
	}
	schemaMap[ENV_KEY] = &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validateKey(),
		Description:  "The segment's environment key.",
	}
	schemaMap[KEY] = &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validateKey(),
		Description:  "The unique key that references the segment.",
	}
	schemaMap[NAME] = &schema.Schema{
		Type:        schema.TypeString,
		Computed:    true,
		Description: "The human-friendly name for the segment.",
	}
	return &schema.Resource{
		Read:   dataSourceSegmentRead,
		Schema: schemaMap,
	}
}

func dataSourceSegmentRead(d *schema.ResourceData, raw interface{}) error {
	return segmentRead(d, raw, true)
}
