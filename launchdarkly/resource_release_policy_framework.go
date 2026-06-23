package launchdarkly

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var (
	_ resource.Resource                   = &ReleasePolicyResource{}
	_ resource.ResourceWithImportState    = &ReleasePolicyResource{}
	_ resource.ResourceWithValidateConfig = &ReleasePolicyResource{}
)

type ReleasePolicyResource struct {
	client *Client
	beta   *Client
}

type ReleasePolicyResourceModel struct {
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

func NewReleasePolicyResource() resource.Resource { return &ReleasePolicyResource{} }

func (r *ReleasePolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_release_policy"
}

func (r *ReleasePolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provides a LaunchDarkly release policy resource.

~> **Beta:** This resource wraps a beta LaunchDarkly API (the ` + "`release-policies`" + ` endpoints, accessed with the ` + "`LD-API-Version: beta`" + ` header). Beta resources may change or be removed in future versions.

This resource lets you create and manage [release policies](https://launchdarkly.com/docs/home/releases) within a LaunchDarkly project. A release policy defines how flag changes roll out to environments — either as a ` + "`guarded-release`" + ` (rollout monitored against metrics, with optional automatic rollback) or a ` + "`progressive-release`" + ` (rollout that advances through a fixed schedule of allocation stages).`,
		Attributes: releasePolicySchemaAttributes(),
	}
}

func releasePolicySchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:      true,
			Description:   "The ID of this resource in the format `project_key/key`.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		PROJECT_KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The release policy's project key.", true),
			Validators:    []validator.String{keyValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The unique human-readable key that references the release policy.", true),
			Validators:    []validator.String{keyValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		NAME: schema.StringAttribute{
			Required:    true,
			Description: "The human-friendly name for the release policy.",
		},
		RELEASE_METHOD: schema.StringAttribute{
			Required:    true,
			Description: "The release method this policy uses. Must be one of `guarded-release` or `progressive-release`. Set `guarded_release_config` for a `guarded-release` and `progressive_release_config` for a `progressive-release`.",
			Validators: []validator.String{
				stringvalidator.OneOf(RELEASE_METHOD_GUARDED, RELEASE_METHOD_PROGRESSIVE),
			},
		},
		RANK: schema.Int64Attribute{
			Computed:      true,
			Description:   "The rank (priority) of the release policy within the project. Rank is assigned and ordered by LaunchDarkly; reorder release policies in the LaunchDarkly UI.",
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
		},
		SCOPE: schema.SingleNestedAttribute{
			Optional:    true,
			Description: "The scope that determines which environments and flags this release policy applies to.",
			Attributes: map[string]schema.Attribute{
				SCOPE_ENVIRONMENT_KEYS: schema.SetAttribute{
					Optional:    true,
					ElementType: types.StringType,
					Description: "The set of environment keys this policy applies to.",
					Validators: []validator.Set{
						setvalidator.ValueStringsAre(keyValidator()),
					},
				},
				FLAG_TAG_KEYS: schema.SetAttribute{
					Optional:    true,
					ElementType: types.StringType,
					Description: "The set of flag tags this policy applies to.",
					Validators: []validator.Set{
						setvalidator.ValueStringsAre(tagValidator()),
					},
				},
			},
		},
		GUARDED_RELEASE_CONFIG: schema.SingleNestedAttribute{
			Optional:    true,
			Description: "Configuration for a `guarded-release`. May only be set when `release_method` is `guarded-release`.",
			Attributes: map[string]schema.Attribute{
				ROLLOUT_CONTEXT_KIND: schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "The context kind key to use as the randomization unit for the rollout.",
				},
				MIN_SAMPLE_SIZE: schema.Int64Attribute{
					Optional:    true,
					Computed:    true,
					Description: "The minimum number of samples required before the policy makes a release decision.",
				},
				ROLLBACK_ON_REGRESSION: schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Description: "Whether to automatically roll back the release when a monitored metric regresses.",
				},
				METRIC_KEYS: schema.SetAttribute{
					Optional:    true,
					ElementType: types.StringType,
					Description: "The set of metric keys to monitor during the guarded release.",
					Validators: []validator.Set{
						setvalidator.ValueStringsAre(keyValidator()),
					},
				},
				METRIC_GROUP_KEYS: schema.SetAttribute{
					Optional:    true,
					ElementType: types.StringType,
					Description: "The set of metric group keys to monitor during the guarded release.",
					Validators: []validator.Set{
						setvalidator.ValueStringsAre(keyValidator()),
					},
				},
				STAGES: releasePolicyStagesSchema(),
			},
		},
		PROGRESSIVE_RELEASE_CONFIG: schema.SingleNestedAttribute{
			Optional:    true,
			Description: "Configuration for a `progressive-release`. May only be set when `release_method` is `progressive-release`.",
			Attributes: map[string]schema.Attribute{
				ROLLOUT_CONTEXT_KIND: schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "The context kind key to use as the randomization unit for the rollout.",
				},
				STAGES: releasePolicyStagesSchema(),
			},
		},
	}
}

func releasePolicyStagesSchema() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Optional:    true,
		Description: "An ordered list of rollout stages. Each stage advances the rollout to the given allocation for the given duration.",
		Validators: []validator.List{
			listvalidator.SizeAtLeast(1),
		},
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				ALLOCATION: schema.Int64Attribute{
					Required:    true,
					Description: "The percentage of traffic (0-100) allocated to the new variation during this stage.",
				},
				DURATION_MILLIS: schema.Int64Attribute{
					Required:    true,
					Description: "The duration of this stage, in milliseconds.",
				},
			},
		},
	}
}

func (r *ReleasePolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
	if r.client == nil {
		return
	}
	beta, err := newReleasePolicyBetaClient(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build LaunchDarkly beta client", err.Error())
		return
	}
	r.beta = beta
}

func (r *ReleasePolicyResource) betaClient() (*Client, error) {
	if r.beta != nil {
		return r.beta, nil
	}
	return newReleasePolicyBetaClient(r.client)
}

// ValidateConfig enforces that the release-method-specific config block matches
// the chosen release_method.
func (r *ReleasePolicyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg ReleasePolicyResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if cfg.ReleaseMethod.IsNull() || cfg.ReleaseMethod.IsUnknown() {
		return
	}
	// Unknown nested objects (e.g. fed from a module output or a count-gated
	// config) are not null in the framework, so guard on both: only flag a
	// genuinely-set opposite config, never one that is still unknown at plan
	// time. The known value is re-validated on the subsequent apply.
	switch cfg.ReleaseMethod.ValueString() {
	case RELEASE_METHOD_GUARDED:
		if !cfg.ProgressiveReleaseConfig.IsNull() && !cfg.ProgressiveReleaseConfig.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root(PROGRESSIVE_RELEASE_CONFIG),
				"Invalid release policy configuration",
				"progressive_release_config must not be set when release_method is \"guarded-release\"",
			)
		}
	case RELEASE_METHOD_PROGRESSIVE:
		if !cfg.GuardedReleaseConfig.IsNull() && !cfg.GuardedReleaseConfig.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root(GUARDED_RELEASE_CONFIG),
				"Invalid release policy configuration",
				"guarded_release_config must not be set when release_method is \"progressive-release\"",
			)
		}
	}
}

func (r *ReleasePolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ReleasePolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	if exists, err := projectExists(projectKey, r.client); !exists {
		if err != nil {
			resp.Diagnostics.AddError("Failed to check project", err.Error())
			return
		}
		resp.Diagnostics.AddError("Project not found", fmt.Sprintf("cannot find project with key %q", projectKey))
		return
	}

	beta, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}

	key := plan.Key.ValueString()
	post := ldapi.PostReleasePolicyRequest{
		Name:          plan.Name.ValueString(),
		Key:           key,
		ReleaseMethod: ldapi.ReleaseMethod(plan.ReleaseMethod.ValueString()),
	}

	scope, d := releasePolicyScopeToAPI(ctx, plan.Scope)
	resp.Diagnostics.Append(d...)
	post.Scope = scope

	guarded, d := guardedReleaseConfigToAPI(ctx, plan.GuardedReleaseConfig)
	resp.Diagnostics.Append(d...)
	post.GuardedReleaseConfig = guarded

	progressive, d := progressiveReleaseConfigToAPI(ctx, plan.ProgressiveReleaseConfig)
	resp.Diagnostics.Append(d...)
	post.ProgressiveReleaseConfig = progressive

	if resp.Diagnostics.HasError() {
		return
	}

	err = beta.withConcurrency(beta.ctx, func() error {
		_, _, e := beta.ld.ReleasePoliciesBetaApi.PostReleasePolicy(beta.ctx, projectKey).
			LDAPIVersion(RELEASE_POLICY_BETA_VERSION).
			PostReleasePolicyRequest(post).
			Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error creating release policy resource: %q", key), err)
		return
	}

	plan.ID = types.StringValue(projectKey + "/" + key)
	r.readIntoModel(ctx, projectKey, key, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ReleasePolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ReleasePolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readIntoModel(ctx, data.ProjectKey.ValueString(), data.Key.ValueString(), &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ReleasePolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ReleasePolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	beta, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	key := plan.Key.ValueString()

	put := ldapi.PutReleasePolicyRequest{
		Name:          plan.Name.ValueString(),
		ReleaseMethod: ldapi.ReleaseMethod(plan.ReleaseMethod.ValueString()),
	}

	scope, d := releasePolicyScopeToAPI(ctx, plan.Scope)
	resp.Diagnostics.Append(d...)
	put.Scope = scope

	guarded, d := guardedReleaseConfigToAPI(ctx, plan.GuardedReleaseConfig)
	resp.Diagnostics.Append(d...)
	put.GuardedReleaseConfig = guarded

	progressive, d := progressiveReleaseConfigToAPI(ctx, plan.ProgressiveReleaseConfig)
	resp.Diagnostics.Append(d...)
	put.ProgressiveReleaseConfig = progressive

	if resp.Diagnostics.HasError() {
		return
	}

	err = beta.withConcurrency(beta.ctx, func() error {
		_, _, e := beta.ld.ReleasePoliciesBetaApi.PutReleasePolicy(beta.ctx, projectKey, key).
			LDAPIVersion(RELEASE_POLICY_BETA_VERSION).
			PutReleasePolicyRequest(put).
			Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error updating release policy resource %q in project %q", key, projectKey), err)
		return
	}

	r.readIntoModel(ctx, projectKey, key, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ReleasePolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ReleasePolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	beta, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}
	var res *http.Response
	err = beta.withConcurrency(beta.ctx, func() error {
		var e error
		res, e = beta.ld.ReleasePoliciesBetaApi.DeleteReleasePolicy(beta.ctx, data.ProjectKey.ValueString(), data.Key.ValueString()).
			LDAPIVersion(RELEASE_POLICY_BETA_VERSION).
			Execute()
		return e
	})
	if err != nil {
		if isStatusNotFound(res) {
			return
		}
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error deleting release policy resource %q", data.Key.ValueString()), err)
	}
}

func (r *ReleasePolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectKey, key, err := releasePolicyIdToKeys(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), projectKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(KEY), key)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *ReleasePolicyResource) readIntoModel(
	ctx context.Context,
	projectKey, key string,
	data *ReleasePolicyResourceModel,
	diags *diag.Diagnostics,
) {
	beta, err := r.betaClient()
	if err != nil {
		diags.AddError("Failed to build beta client", err.Error())
		return
	}

	var policy *ldapi.ReleasePolicy
	var res *http.Response
	err = beta.withConcurrency(beta.ctx, func() error {
		policy, res, err = beta.ld.ReleasePoliciesBetaApi.GetReleasePolicy(beta.ctx, projectKey, key).
			LDAPIVersion(RELEASE_POLICY_BETA_VERSION).
			Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError(fmt.Sprintf("Failed to get release policy %q", key), handleLdapiErr(err).Error())
		return
	}

	data.ID = types.StringValue(projectKey + "/" + key)
	data.ProjectKey = types.StringValue(projectKey)
	data.Key = types.StringValue(policy.Key)
	data.Name = types.StringValue(policy.Name)
	data.ReleaseMethod = types.StringValue(string(policy.ReleaseMethod))
	data.Rank = types.Int64Value(int64(policy.Rank))

	scopeObj, d := releasePolicyScopeToObject(ctx, policy.Scope, data.Scope)
	diags.Append(d...)
	data.Scope = scopeObj

	guardedObj, d := guardedReleaseConfigToObject(ctx, policy.GuardedReleaseConfig, data.GuardedReleaseConfig)
	diags.Append(d...)
	data.GuardedReleaseConfig = guardedObj

	progressiveObj, d := progressiveReleaseConfigToObject(ctx, policy.ProgressiveReleaseConfig)
	diags.Append(d...)
	data.ProgressiveReleaseConfig = progressiveObj
}
