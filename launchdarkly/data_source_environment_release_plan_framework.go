package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var _ datasource.DataSource = &EnvironmentReleasePlanDataSource{}

type EnvironmentReleasePlanDataSource struct {
	client *Client
}

func NewEnvironmentReleasePlanDataSource() datasource.DataSource {
	return &EnvironmentReleasePlanDataSource{}
}

func (d *EnvironmentReleasePlanDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_release_plan"
}

func (d *EnvironmentReleasePlanDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly environment release plan data source.\n\n~> **Beta:** This data source wraps a beta LaunchDarkly API (the `release-plans` endpoints, accessed with the `LD-API-Version: beta` header). Beta resources may change or be removed in future versions.\n\nThis data source allows you to retrieve release plan information from a LaunchDarkly environment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID in the format `project_key/environment_key/key`.",
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The release plan's project key.",
			},
			ENVIRONMENT_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The key of the environment the release plan belongs to.",
			},
			KEY: schema.StringAttribute{
				Required:    true,
				Description: "The unique human-readable key that references the release plan.",
			},
			NAME: schema.StringAttribute{
				Computed:    true,
				Description: "The human-friendly name for the release plan.",
			},
			DESCRIPTION: schema.StringAttribute{
				Computed:    true,
				Description: "The description of the release plan's purpose.",
			},
			RELEASE_METHOD: schema.StringAttribute{
				Computed:    true,
				Description: "The release method the plan uses. Currently always `progressive_release`.",
			},
			VERSION: schema.Int64Attribute{
				Computed:    true,
				Description: "The release plan's version. Starts at 1 and increments on every update.",
			},
			IS_DEFAULT: schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this plan is the environment's default release plan.",
			},
			PROGRESSIVE_RELEASE: schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Configuration for the progressive release this plan runs.",
				Attributes: map[string]schema.Attribute{
					RANDOMIZATION_UNIT: schema.StringAttribute{
						Computed:    true,
						Description: "The context kind key used to bucket contexts for allocation.",
					},
					ADVANCEMENT_MODE: schema.StringAttribute{
						Computed:    true,
						Description: "Whether a release started from this plan advances to the next phase automatically (`automatic`) or waits for a user or API action (`manual`).",
					},
					CLEANUP_BEHAVIOR: schema.StringAttribute{
						Computed:    true,
						Description: "What happens to the flag's targeting rules once a release started from this plan completes. One of `remove_rules_and_update_default_rule` or `leave_rules_in_place`.",
					},
					PHASES: schema.ListNestedAttribute{
						Computed:    true,
						Description: "An ordered list of phases the release progresses through.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								DESCRIPTION: schema.StringAttribute{
									Computed:    true,
									Description: "The description of this phase.",
								},
								STEPS: schema.ListNestedAttribute{
									Computed:    true,
									Description: "Ordered allocation increments within this phase.",
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											ALLOCATION: schema.Int64Attribute{
												Computed:    true,
												Description: "The traffic allocation for this step, in per-100,000 units. `100000` is 100% of traffic.",
											},
											DURATION_MILLIS: schema.Int64Attribute{
												Computed:    true,
												Description: "How long this step holds, in milliseconds, before the next step starts.",
											},
										},
									},
								},
								CLAUSES: frameworkClausesDataSourceAttribute(),
							},
						},
					},
					METRICS: schema.MapNestedAttribute{
						Computed:    true,
						Description: "Metrics and metric groups monitored during the rollout, keyed by metric or metric group key.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								IS_GROUP: schema.BoolAttribute{
									Computed:    true,
									Description: "Whether the key identifies a metric group (`true`) or a single metric (`false`).",
								},
								AUTO_ROLLBACK: schema.BoolAttribute{
									Computed:    true,
									Description: "Whether the release automatically rolls back if this metric or metric group regresses.",
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *EnvironmentReleasePlanDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *EnvironmentReleasePlanDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data EnvironmentReleasePlanResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	envKey := data.EnvironmentKey.ValueString()
	key := data.Key.ValueString()

	plan, _, err := getEnvironmentReleasePlan(ctx, d.client, projectKey, envKey, key)
	if err != nil {
		resp.Diagnostics.AddError(handleLdapiErr(err).Error(), "")
		return
	}

	environmentReleasePlanToModel(ctx, projectKey, envKey, plan, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
