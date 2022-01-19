package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAuditLogSubscription() *schema.Resource {
	schemaMap := auditLogSubscriptionSchema(true)
	schemaMap[ID] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "The audit log subscription ID",
	}
	return &schema.Resource{
		ReadContext: dataSourceAuditLogSubscriptionRead,
		Schema:      schemaMap,
	}
}

func dataSourceAuditLogSubscriptionRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	return auditLogSubscriptionRead(ctx, d, metaRaw, true)
}
