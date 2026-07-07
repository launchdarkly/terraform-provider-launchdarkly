package launchdarkly

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v23"
)

var _ datasource.DataSource = &ReleasePolicyDataSource{}

type ReleasePolicyDataSource struct {
	client *Client
}

type ReleasePolicyDataSourceModel struct {
	ID                       types.String `tfsdk:"id"`
	ProjectKey               types.String `tfsdk:"project_key"`
	Key                      types.String `tfsdk:"key"`
	Name                     types.String `tfsdk:"name"`
	ReleaseMethod            types.String `tfsdk:"release_method"`
	Rank                     types.Int64  `tfsdk:"rank"`
	Scope                    types.Object `tfsdk:"scope"`
	GuardedReleaseConfig     types.Object `tfsdk:"guarded_release_config"`
	ProgressiveReleaseConfig types.Object `tfsdk:"progressive_release_config"`
}

func NewReleasePolicyDataSource() datasource.DataSource {
	return &ReleasePolicyDataSource{}
}

func (d *ReleasePolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_release_policy"
}

func (d *ReleasePolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly release policy data source.\n\n~> **Beta:** This data source wraps a beta LaunchDarkly API (the `release-policies` endpoints, accessed with the `LD-API-Version: beta` header). Beta resources may change or be removed in future versions.\n\nThis data source allows you to retrieve release policy information from your LaunchDarkly project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID in the format `project_key/key`.",
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The release policy's project key.",
			},
			KEY: schema.StringAttribute{
				Required:    true,
				Description: "The unique human-readable key that references the release policy.",
			},
			NAME: schema.StringAttribute{
				Computed:    true,
				Description: "The human-friendly name for the release policy.",
			},
			RELEASE_METHOD: schema.StringAttribute{
				Computed:    true,
				Description: "The release method this policy uses. One of `guarded-release` or `progressive-release`.",
			},
			RANK: schema.Int64Attribute{
				Computed:    true,
				Description: "The rank (priority) of the release policy within the project.",
			},
			SCOPE: schema.SingleNestedAttribute{
				Computed:    true,
				Description: "The scope that determines which environments and flags this release policy applies to.",
				Attributes: map[string]schema.Attribute{
					SCOPE_ENVIRONMENT_KEYS: schema.SetAttribute{
						Computed:    true,
						ElementType: types.StringType,
						Description: "The set of environment keys this policy applies to.",
					},
					FLAG_TAG_KEYS: schema.SetAttribute{
						Computed:    true,
						ElementType: types.StringType,
						Description: "The set of flag tags this policy applies to.",
					},
				},
			},
			GUARDED_RELEASE_CONFIG: schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Configuration for a `guarded-release`.",
				Attributes: map[string]schema.Attribute{
					ROLLOUT_CONTEXT_KIND: schema.StringAttribute{
						Computed:    true,
						Description: "The context kind key used as the randomization unit for the rollout.",
					},
					MIN_SAMPLE_SIZE: schema.Int64Attribute{
						Computed:    true,
						Description: "The minimum number of samples required before the policy makes a release decision.",
					},
					ROLLBACK_ON_REGRESSION: schema.BoolAttribute{
						Computed:    true,
						Description: "Whether to automatically roll back the release when a monitored metric regresses.",
					},
					METRIC_KEYS: schema.SetAttribute{
						Computed:    true,
						ElementType: types.StringType,
						Description: "The set of metric keys monitored during the guarded release.",
					},
					METRIC_GROUP_KEYS: schema.SetAttribute{
						Computed:    true,
						ElementType: types.StringType,
						Description: "The set of metric group keys monitored during the guarded release.",
					},
					STAGES: releasePolicyDataSourceStagesSchema(),
				},
			},
			PROGRESSIVE_RELEASE_CONFIG: schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Configuration for a `progressive-release`.",
				Attributes: map[string]schema.Attribute{
					ROLLOUT_CONTEXT_KIND: schema.StringAttribute{
						Computed:    true,
						Description: "The context kind key used as the randomization unit for the rollout.",
					},
					STAGES: releasePolicyDataSourceStagesSchema(),
				},
			},
		},
	}
}

func releasePolicyDataSourceStagesSchema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Computed:    true,
		Description: "An ordered list of rollout stages.",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				ALLOCATION: schema.Int64Attribute{
					Computed:    true,
					Description: "The percentage of traffic (0-100) allocated to the new variation during this stage.",
				},
				DURATION_MILLIS: schema.Int64Attribute{
					Computed:    true,
					Description: "The duration of this stage, in milliseconds.",
				},
			},
		},
	}
}

func (d *ReleasePolicyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *ReleasePolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		return
	}

	var data ReleasePolicyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	beta, err := newReleasePolicyBetaClient(d.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}

	projectKey := data.ProjectKey.ValueString()
	key := data.Key.ValueString()

	var policy *ldapi.ReleasePolicy
	err = beta.withConcurrency(beta.ctx, func() error {
		policy, _, err = beta.ld.ReleasePoliciesBetaApi.GetReleasePolicy(beta.ctx, projectKey, key).
			LDAPIVersion(RELEASE_POLICY_BETA_VERSION).
			Execute()
		return err
	})
	if err != nil {
		// Surface the raw upstream error so ExpectError regex matches
		// "Error: 404 Not Found:" directly against the summary.
		resp.Diagnostics.AddError(handleLdapiErr(err).Error(), "")
		return
	}

	data.ID = types.StringValue(projectKey + "/" + key)
	data.Key = types.StringValue(policy.Key)
	data.Name = types.StringValue(policy.Name)
	data.ReleaseMethod = types.StringValue(string(policy.ReleaseMethod))
	data.Rank = types.Int64Value(int64(policy.Rank))

	scopeObj, diags := releasePolicyScopeToObject(ctx, policy.Scope, types.ObjectNull(releasePolicyScopeAttrTypes))
	resp.Diagnostics.Append(diags...)
	data.Scope = scopeObj

	guardedObj, diags := guardedReleaseConfigToObject(ctx, policy.GuardedReleaseConfig, types.ObjectNull(guardedReleaseConfigAttrTypes))
	resp.Diagnostics.Append(diags...)
	data.GuardedReleaseConfig = guardedObj

	progressiveObj, diags := progressiveReleaseConfigToObject(ctx, policy.ProgressiveReleaseConfig)
	resp.Diagnostics.Append(diags...)
	data.ProgressiveReleaseConfig = progressiveObj

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
