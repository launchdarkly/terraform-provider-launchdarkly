package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var releasePolicyStageSchemaComputed = &schema.Resource{
	Schema: map[string]*schema.Schema{
		STAGE_ALLOCATION: {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The allocation for this stage (in thousandths, e.g. 25000 = 25%).",
		},
		STAGE_DURATION_MILLIS: {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The duration in milliseconds for this stage.",
		},
	},
}

func dataSourceReleasePolicy() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceReleasePolicyRead,

		Description: `Provides a LaunchDarkly release policy data source. This data source is still in beta.

This data source allows you to retrieve release policy information from your LaunchDarkly organization.

Learn more about [release policies here](https://launchdarkly.com/docs/home/releases/release-policies), and read our [API docs here](https://launchdarkly.com/docs/api/release-policies-beta/).`,

		Schema: map[string]*schema.Schema{
			PROJECT_KEY: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The project key.",
			},
			KEY: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The release policy's unique key.",
			},
			NAME: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The release policy's name.",
			},
			RELEASE_METHOD: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The release method for the release policy.",
			},
			SCOPE: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The scope configuration for the release policy.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						SCOPE_ENVIRONMENT_KEYS: {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "The environment keys for environments the release policy is applied to.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			GUARDED_RELEASE_CONFIG: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Configuration for guarded release.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						ROLLBACK_ON_REGRESSION: {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether to automatically rollback on regression.",
						},
						MIN_SAMPLE_SIZE: {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The minimum sample size for the release policy.",
						},
						STAGES: {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "The stages for the guarded release.",
							Elem:        releasePolicyStageSchemaComputed,
						},
					},
				},
			},
			PROGRESSIVE_RELEASE_CONFIG: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Configuration for progressive release.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						STAGES: {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "The stages for the progressive release.",
							Elem:        releasePolicyStageSchemaComputed,
						},
					},
				},
			},
		},
	}
}

func dataSourceReleasePolicyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return releasePolicyRead(ctx, d, meta, true)
}
