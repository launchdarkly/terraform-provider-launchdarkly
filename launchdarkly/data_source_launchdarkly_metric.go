package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceMetric() *schema.Resource {
	schemaMap := baseMetricSchema(true)
	return &schema.Resource{
		ReadContext: dataSourceMetricRead,
		Schema:      schemaMap,
	}
}

func dataSourceMetricRead(ctx context.Context, d *schema.ResourceData, raw interface{}) diag.Diagnostics {
	return metricRead(ctx, d, raw, true)
}
