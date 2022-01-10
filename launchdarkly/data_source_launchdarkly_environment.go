package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceEnvironment() *schema.Resource {
	envSchema := dataSourceEnvironmentSchema(false)
	envSchema[PROJECT_KEY] = &schema.Schema{
		Type:             schema.TypeString,
		Required:         true,
		ValidateDiagFunc: validateKey(),
	}
	return &schema.Resource{
		ReadContext: dataSourceEnvironmentRead,
		Schema:      envSchema,
	}
}

func dataSourceEnvironmentRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return environmentRead(ctx, d, meta, true)
}
