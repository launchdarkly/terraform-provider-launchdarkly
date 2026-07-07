package launchdarkly

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	ldapi "github.com/launchdarkly/api-client-go/v23"
)

// Release methods supported by a release policy. A guarded release rolls out
// while watching metrics and can roll back automatically; a progressive
// release advances through a fixed schedule of allocation stages.
const (
	RELEASE_METHOD_GUARDED      = "guarded-release"
	RELEASE_METHOD_PROGRESSIVE  = "progressive-release"
	RELEASE_POLICY_BETA_VERSION = "beta"
)

// releasePolicyScopeModel mirrors the `scope` single-nested attribute.
type releasePolicyScopeModel struct {
	EnvironmentKeys types.Set `tfsdk:"environment_keys"`
	FlagTagKeys     types.Set `tfsdk:"flag_tag_keys"`
}

// releasePolicyStageModel mirrors one element of a config's `stages` list.
type releasePolicyStageModel struct {
	Allocation     types.Int64 `tfsdk:"allocation"`
	DurationMillis types.Int64 `tfsdk:"duration_millis"`
}

// guardedReleaseConfigModel mirrors the `guarded_release_config` single-nested
// attribute.
type guardedReleaseConfigModel struct {
	RolloutContextKind   types.String `tfsdk:"rollout_context_kind"`
	MinSampleSize        types.Int64  `tfsdk:"min_sample_size"`
	RollbackOnRegression types.Bool   `tfsdk:"rollback_on_regression"`
	MetricKeys           types.Set    `tfsdk:"metric_keys"`
	MetricGroupKeys      types.Set    `tfsdk:"metric_group_keys"`
	Stages               types.List   `tfsdk:"stages"`
}

// progressiveReleaseConfigModel mirrors the `progressive_release_config`
// single-nested attribute.
type progressiveReleaseConfigModel struct {
	RolloutContextKind types.String `tfsdk:"rollout_context_kind"`
	Stages             types.List   `tfsdk:"stages"`
}

var releasePolicyScopeAttrTypes = map[string]attr.Type{
	SCOPE_ENVIRONMENT_KEYS: types.SetType{ElemType: types.StringType},
	FLAG_TAG_KEYS:          types.SetType{ElemType: types.StringType},
}

var releasePolicyStageAttrTypes = map[string]attr.Type{
	ALLOCATION:      types.Int64Type,
	DURATION_MILLIS: types.Int64Type,
}

var releasePolicyStageObjectType = types.ObjectType{AttrTypes: releasePolicyStageAttrTypes}

var guardedReleaseConfigAttrTypes = map[string]attr.Type{
	ROLLOUT_CONTEXT_KIND:   types.StringType,
	MIN_SAMPLE_SIZE:        types.Int64Type,
	ROLLBACK_ON_REGRESSION: types.BoolType,
	METRIC_KEYS:            types.SetType{ElemType: types.StringType},
	METRIC_GROUP_KEYS:      types.SetType{ElemType: types.StringType},
	STAGES:                 types.ListType{ElemType: releasePolicyStageObjectType},
}

var progressiveReleaseConfigAttrTypes = map[string]attr.Type{
	ROLLOUT_CONTEXT_KIND: types.StringType,
	STAGES:               types.ListType{ElemType: releasePolicyStageObjectType},
}

// newReleasePolicyBetaClient returns a beta-configured client for the
// release-policies endpoints. The generated ReleasePoliciesBetaApi request
// builders require an explicit per-request .LDAPIVersion("beta"); we use a
// beta-configured client here too so the LD-API-Version default is consistent
// with the per-request header.
func newReleasePolicyBetaClient(c *Client) (*Client, error) {
	return newBetaClient(c.apiKey, c.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
}

// releasePolicyIdToKeys splits a composite release policy ID into its project
// key and policy key. The expected format is `project_key/policy_key`.
func releasePolicyIdToKeys(id string) (projectKey string, policyKey string, err error) {
	if strings.Count(id, "/") != 1 {
		return "", "", fmt.Errorf("found unexpected release policy id format: %q expected format: 'project_key/policy_key'", id)
	}
	parts := strings.SplitN(id, "/", 2)
	if parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("found unexpected release policy id format: %q expected format: 'project_key/policy_key'", id)
	}
	return parts[0], parts[1], nil
}

// int64ValueFromInt32Pointer maps an optional API int32 into a framework
// Int64, returning null when the pointer is nil.
func int64ValueFromInt32Pointer(p *int32) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*p))
}

// boolValueOrNullFromPointer maps an optional API bool into a framework Bool,
// returning null when the pointer is nil.
func boolValueOrNullFromPointer(p *bool) types.Bool {
	if p == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*p)
}

// ── scope ──────────────────────────────────────────────────────────────────

func releasePolicyScopeToAPI(ctx context.Context, obj types.Object) (*ldapi.ReleasePolicyScope, diag.Diagnostics) {
	var diags diag.Diagnostics
	if obj.IsNull() || obj.IsUnknown() {
		return nil, diags
	}
	var m releasePolicyScopeModel
	diags.Append(obj.As(ctx, &m, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}
	scope := &ldapi.ReleasePolicyScope{}
	envs, d := stringSliceFromSet(ctx, m.EnvironmentKeys)
	diags.Append(d...)
	scope.EnvironmentKeys = envs
	tags, d := stringSliceFromSet(ctx, m.FlagTagKeys)
	diags.Append(d...)
	scope.FlagTagKeys = tags
	return scope, diags
}

func releasePolicyScopeToObject(ctx context.Context, scope *ldapi.ReleasePolicyScope, existing types.Object) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	if scope == nil {
		return types.ObjectNull(releasePolicyScopeAttrTypes), diags
	}
	var existingEnvs, existingTags types.Set
	if !existing.IsNull() && !existing.IsUnknown() {
		var em releasePolicyScopeModel
		diags.Append(existing.As(ctx, &em, basetypes.ObjectAsOptions{})...)
		existingEnvs = em.EnvironmentKeys
		existingTags = em.FlagTagKeys
	}
	envs, d := setFromStringSlicePreservingPlan(ctx, scope.EnvironmentKeys, existingEnvs)
	diags.Append(d...)
	tags, d := setFromStringSlicePreservingPlan(ctx, scope.FlagTagKeys, existingTags)
	diags.Append(d...)
	m := releasePolicyScopeModel{EnvironmentKeys: envs, FlagTagKeys: tags}
	obj, d := types.ObjectValueFrom(ctx, releasePolicyScopeAttrTypes, m)
	diags.Append(d...)
	return obj, diags
}

// ── stages ─────────────────────────────────────────────────────────────────

func releasePolicyStagesToAPI(ctx context.Context, list types.List) ([]ldapi.ReleasePolicyStage, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}
	var models []releasePolicyStageModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]ldapi.ReleasePolicyStage, len(models))
	for i, s := range models {
		out[i] = ldapi.ReleasePolicyStage{
			Allocation:     int32(s.Allocation.ValueInt64()),
			DurationMillis: s.DurationMillis.ValueInt64(),
		}
	}
	return out, diags
}

func releasePolicyStagesToList(stages []ldapi.ReleasePolicyStage) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(stages) == 0 {
		return types.ListNull(releasePolicyStageObjectType), diags
	}
	elems := make([]attr.Value, 0, len(stages))
	for _, s := range stages {
		obj, d := types.ObjectValue(releasePolicyStageAttrTypes, map[string]attr.Value{
			ALLOCATION:      types.Int64Value(int64(s.Allocation)),
			DURATION_MILLIS: types.Int64Value(s.DurationMillis),
		})
		diags.Append(d...)
		elems = append(elems, obj)
	}
	list, d := types.ListValue(releasePolicyStageObjectType, elems)
	diags.Append(d...)
	return list, diags
}

// ── guarded release config ──────────────────────────────────────────────────

func guardedReleaseConfigToAPI(ctx context.Context, obj types.Object) (*ldapi.GuardedReleaseConfig, diag.Diagnostics) {
	var diags diag.Diagnostics
	if obj.IsNull() || obj.IsUnknown() {
		return nil, diags
	}
	var m guardedReleaseConfigModel
	diags.Append(obj.As(ctx, &m, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}
	cfg := &ldapi.GuardedReleaseConfig{}
	if !m.RolloutContextKind.IsNull() && !m.RolloutContextKind.IsUnknown() && m.RolloutContextKind.ValueString() != "" {
		v := m.RolloutContextKind.ValueString()
		cfg.RolloutContextKindKey = &v
	}
	if !m.MinSampleSize.IsNull() && !m.MinSampleSize.IsUnknown() {
		v := int32(m.MinSampleSize.ValueInt64())
		cfg.MinSampleSize = &v
	}
	if !m.RollbackOnRegression.IsNull() && !m.RollbackOnRegression.IsUnknown() {
		v := m.RollbackOnRegression.ValueBool()
		cfg.RollbackOnRegression = &v
	}
	metricKeys, d := stringSliceFromSet(ctx, m.MetricKeys)
	diags.Append(d...)
	cfg.MetricKeys = metricKeys
	metricGroupKeys, d := stringSliceFromSet(ctx, m.MetricGroupKeys)
	diags.Append(d...)
	cfg.MetricGroupKeys = metricGroupKeys
	stages, d := releasePolicyStagesToAPI(ctx, m.Stages)
	diags.Append(d...)
	cfg.Stages = stages
	return cfg, diags
}

func guardedReleaseConfigToObject(ctx context.Context, cfg *ldapi.GuardedReleaseConfig, existing types.Object) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	if cfg == nil {
		return types.ObjectNull(guardedReleaseConfigAttrTypes), diags
	}
	var existingMetricKeys, existingMetricGroupKeys types.Set
	if !existing.IsNull() && !existing.IsUnknown() {
		var em guardedReleaseConfigModel
		diags.Append(existing.As(ctx, &em, basetypes.ObjectAsOptions{})...)
		existingMetricKeys = em.MetricKeys
		existingMetricGroupKeys = em.MetricGroupKeys
	}
	metricKeys, d := setFromStringSlicePreservingPlan(ctx, cfg.MetricKeys, existingMetricKeys)
	diags.Append(d...)
	metricGroupKeys, d := setFromStringSlicePreservingPlan(ctx, cfg.MetricGroupKeys, existingMetricGroupKeys)
	diags.Append(d...)
	stages, d := releasePolicyStagesToList(cfg.Stages)
	diags.Append(d...)
	m := guardedReleaseConfigModel{
		RolloutContextKind:   stringValueOrNullFromPointer(cfg.RolloutContextKindKey),
		MinSampleSize:        int64ValueFromInt32Pointer(cfg.MinSampleSize),
		RollbackOnRegression: boolValueOrNullFromPointer(cfg.RollbackOnRegression),
		MetricKeys:           metricKeys,
		MetricGroupKeys:      metricGroupKeys,
		Stages:               stages,
	}
	obj, d := types.ObjectValueFrom(ctx, guardedReleaseConfigAttrTypes, m)
	diags.Append(d...)
	return obj, diags
}

// ── progressive release config ──────────────────────────────────────────────

func progressiveReleaseConfigToAPI(ctx context.Context, obj types.Object) (*ldapi.ProgressiveReleaseConfig, diag.Diagnostics) {
	var diags diag.Diagnostics
	if obj.IsNull() || obj.IsUnknown() {
		return nil, diags
	}
	var m progressiveReleaseConfigModel
	diags.Append(obj.As(ctx, &m, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}
	cfg := &ldapi.ProgressiveReleaseConfig{}
	if !m.RolloutContextKind.IsNull() && !m.RolloutContextKind.IsUnknown() && m.RolloutContextKind.ValueString() != "" {
		v := m.RolloutContextKind.ValueString()
		cfg.RolloutContextKindKey = &v
	}
	stages, d := releasePolicyStagesToAPI(ctx, m.Stages)
	diags.Append(d...)
	cfg.Stages = stages
	return cfg, diags
}

func progressiveReleaseConfigToObject(ctx context.Context, cfg *ldapi.ProgressiveReleaseConfig) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	if cfg == nil {
		return types.ObjectNull(progressiveReleaseConfigAttrTypes), diags
	}
	stages, d := releasePolicyStagesToList(cfg.Stages)
	diags.Append(d...)
	m := progressiveReleaseConfigModel{
		RolloutContextKind: stringValueOrNullFromPointer(cfg.RolloutContextKindKey),
		Stages:             stages,
	}
	obj, d := types.ObjectValueFrom(ctx, progressiveReleaseConfigAttrTypes, m)
	diags.Append(d...)
	return obj, diags
}
