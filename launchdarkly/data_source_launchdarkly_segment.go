package launchdarkly

import "github.com/hashicorp/terraform-plugin-sdk/helper/schema"

func dataSourceSegment() *schema.Resource {
	schemaMap := baseSegmentSchema()
	schemaMap[PROJECT_KEY] = &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validateKey(),
	}
	schemaMap[ENV_KEY] = &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validateKey(),
	}
	schemaMap[KEY] = &schema.Schema{
		Type:         schema.TypeString,
		Required:     true,
		ValidateFunc: validateKey(),
	}
	schemaMap[NAME] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	}
	return &schema.Resource{
		Read:   dataSourceSegmentRead,
		Schema: schemaMap,
	}
}

func dataSourceSegmentRead(d *schema.ResourceData, raw interface{}) error {
	return segmentRead(d, raw, true)
}
