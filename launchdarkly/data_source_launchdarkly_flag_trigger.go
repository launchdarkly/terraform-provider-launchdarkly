package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceFlagTrigger() *schema.Resource {
	schemaMap := baseFlagTriggerSchema(true)
	schemaMap[ID] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The flag trigger resource ID. This can be found on your trigger URL - please see docs for more info",
	}
	return &schema.Resource{
		ReadContext: dataSourceFlagTriggerRead,
		Schema:      schemaMap,
	}
}

func dataSourceFlagTriggerRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return flagTriggerRead(ctx, d, metaRaw, true)
}
