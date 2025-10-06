package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceReleasePolicyOrder() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceReleasePolicyOrderRead,

		Description: `Provides a LaunchDarkly release policy order data source. This resource is still in beta.

This data source allows you to retrieve the priority of release policies within LaunchDarkly projects.

Learn more about [release policies here](https://launchdarkly.com/docs/home/releases/release-policies), and read our [API docs here](https://launchdarkly.com/docs/api/release-policies-beta/).`,

		Schema: map[string]*schema.Schema{
			PROJECT_KEY: {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "The project key.",
				ValidateDiagFunc: validateKeyAndLength(1, 140),
			},
			RELEASE_POLICY_KEYS: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "An ordered list of release policy keys that defines the order of release policies within the project.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceReleasePolicyOrderRead(ctx context.Context, d *schema.ResourceData, metaRaw interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := metaRaw.(*Client)

	projectKey := d.Get(PROJECT_KEY).(string)

	order, err := getReleasePolicyOrder(client, projectKey)
	if err != nil {
		return diag.Errorf("failed to get release policy order for project %q: %s", projectKey, handleLdapiErr(err))
	}

	d.SetId(projectKey)
	_ = d.Set(PROJECT_KEY, projectKey)
	_ = d.Set(RELEASE_POLICY_KEYS, order)

	return diags
}
