package launchdarkly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	ldapi "github.com/launchdarkly/api-client-go/v25"
)

const (
	ENVIRONMENT_RELEASE_PLAN_BETA_VERSION = "beta"
	// The only release method the environment release plans API accepts today.
	RELEASE_PLAN_METHOD_PROGRESSIVE = "progressive_release"
	ADVANCEMENT_MODE_MANUAL         = "manual"
	ADVANCEMENT_MODE_AUTOMATIC      = "automatic"
	CLEANUP_BEHAVIOR_REMOVE_RULES   = "remove_rules_and_update_default_rule"
	CLEANUP_BEHAVIOR_LEAVE_RULES    = "leave_rules_in_place"
	// Step allocations are in per-100,000 units, so 100000 is 100% of traffic.
	RELEASE_PLAN_MAX_ALLOCATION = 100000
)

// api-client-go v25.2.0 does not generate the environment release plans
// endpoints yet, so the structs below mirror the public OpenAPI schemas and the
// calls go over raw HTTP (the same approach as fetchDependentFlags). Replace
// them with the generated client once it ships these endpoints.

type environmentReleasePlanRep struct {
	ID                 string                                   `json:"_id"`
	Key                string                                   `json:"key"`
	Name               string                                   `json:"name"`
	Description        *string                                  `json:"description,omitempty"`
	EnvironmentKey     string                                   `json:"environmentKey"`
	ReleaseMethod      string                                   `json:"releaseMethod"`
	Version            int64                                    `json:"version"`
	Default            *bool                                    `json:"default,omitempty"`
	ProgressiveRelease environmentReleasePlanProgressiveRelease `json:"progressiveRelease"`
}

type environmentReleasePlanPostRequest struct {
	Key                string                                   `json:"key"`
	Name               string                                   `json:"name"`
	Description        *string                                  `json:"description,omitempty"`
	ReleaseMethod      string                                   `json:"releaseMethod"`
	ProgressiveRelease environmentReleasePlanProgressiveRelease `json:"progressiveRelease"`
}

type environmentReleasePlanPutRequest struct {
	Name               string                                   `json:"name"`
	Description        *string                                  `json:"description,omitempty"`
	ReleaseMethod      string                                   `json:"releaseMethod"`
	ProgressiveRelease environmentReleasePlanProgressiveRelease `json:"progressiveRelease"`
}

type environmentReleasePlanProgressiveRelease struct {
	RandomizationUnit string                                  `json:"randomizationUnit,omitempty"`
	AdvancementMode   *string                                 `json:"advancementMode,omitempty"`
	CleanupBehavior   *string                                 `json:"cleanupBehavior,omitempty"`
	Phases            []environmentReleasePlanPhase           `json:"phases"`
	Metrics           map[string]environmentReleasePlanMetric `json:"metrics,omitempty"`
}

type environmentReleasePlanPhase struct {
	Description *string                      `json:"description,omitempty"`
	Steps       []environmentReleasePlanStep `json:"steps"`
	Clauses     []ldapi.Clause               `json:"clauses"`
}

type environmentReleasePlanStep struct {
	Allocation     int64 `json:"allocation"`
	DurationMillis int64 `json:"durationMillis"`
}

type environmentReleasePlanMetric struct {
	IsGroup      bool `json:"isGroup"`
	AutoRollback bool `json:"autoRollback"`
}

// environmentReleasePlanEndpoint builds the release-plans URL for the
// configured host. An empty planKey addresses the collection.
func environmentReleasePlanEndpoint(apiHost, projectKey, envKey, planKey string) string {
	host := apiHost
	if host == "" {
		host = DEFAULT_LAUNCHDARKLY_HOST
	}
	p := fmt.Sprintf("/api/v2/projects/%s/environments/%s/release-plans", url.PathEscape(projectKey), url.PathEscape(envKey))
	if planKey != "" {
		p += "/" + url.PathEscape(planKey)
	}
	if u, err := url.Parse(host); err == nil && u.Scheme != "" {
		u.Path = p
		return u.String()
	}
	return fmt.Sprintf("https://%s%s", host, p)
}

// doEnvironmentReleasePlanRequest sends one beta release-plans request and
// decodes a 2xx body into out when out is non-nil. Errors for 4xx/5xx carry the
// status line and response body so callers can surface them as-is.
func doEnvironmentReleasePlanRequest(ctx context.Context, client *Client, method, endpoint string, body, out interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("LD-API-Version", ENVIRONMENT_RELEASE_PLAN_BETA_VERSION)
	req.Header.Set("User-Agent", fmt.Sprintf("launchdarkly-terraform-provider/%s", version))

	var resp *http.Response
	err = client.withConcurrency(ctx, func() error {
		var e error
		resp, e = client.ld.GetConfig().HTTPClient.Do(req)
		return e
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode >= 400 {
		return resp, fmt.Errorf("%d %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(respBody))
	}
	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return resp, fmt.Errorf("failed to decode environment release plan response: %w", err)
		}
	}
	return resp, nil
}

func getEnvironmentReleasePlan(ctx context.Context, client *Client, projectKey, envKey, key string) (*environmentReleasePlanRep, *http.Response, error) {
	var plan environmentReleasePlanRep
	endpoint := environmentReleasePlanEndpoint(client.apiHost, projectKey, envKey, key)
	res, err := doEnvironmentReleasePlanRequest(ctx, client, http.MethodGet, endpoint, nil, &plan)
	if err != nil {
		return nil, res, err
	}
	return &plan, res, nil
}

func createEnvironmentReleasePlan(ctx context.Context, client *Client, projectKey, envKey string, post environmentReleasePlanPostRequest) (*environmentReleasePlanRep, error) {
	var plan environmentReleasePlanRep
	endpoint := environmentReleasePlanEndpoint(client.apiHost, projectKey, envKey, "")
	if _, err := doEnvironmentReleasePlanRequest(ctx, client, http.MethodPost, endpoint, post, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

func updateEnvironmentReleasePlan(ctx context.Context, client *Client, projectKey, envKey, key string, put environmentReleasePlanPutRequest) (*environmentReleasePlanRep, error) {
	var plan environmentReleasePlanRep
	endpoint := environmentReleasePlanEndpoint(client.apiHost, projectKey, envKey, key)
	if _, err := doEnvironmentReleasePlanRequest(ctx, client, http.MethodPut, endpoint, put, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

func deleteEnvironmentReleasePlan(ctx context.Context, client *Client, projectKey, envKey, key string) (*http.Response, error) {
	endpoint := environmentReleasePlanEndpoint(client.apiHost, projectKey, envKey, key)
	return doEnvironmentReleasePlanRequest(ctx, client, http.MethodDelete, endpoint, nil, nil)
}

func environmentReleasePlanID(projectKey, envKey, key string) string {
	return projectKey + "/" + envKey + "/" + key
}

// environmentReleasePlanIdToKeys splits a composite ID of the form
// `project_key/environment_key/key`.
func environmentReleasePlanIdToKeys(id string) (projectKey, envKey, key string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("found unexpected environment release plan id format: %q expected format: 'project_key/environment_key/key'", id)
	}
	return parts[0], parts[1], parts[2], nil
}

// ── framework models ────────────────────────────────────────────────────────

type environmentReleasePlanProgressiveReleaseModel struct {
	RandomizationUnit types.String `tfsdk:"randomization_unit"`
	AdvancementMode   types.String `tfsdk:"advancement_mode"`
	CleanupBehavior   types.String `tfsdk:"cleanup_behavior"`
	Phases            types.List   `tfsdk:"phases"`
	Metrics           types.Map    `tfsdk:"metrics"`
}

type environmentReleasePlanPhaseModel struct {
	Description types.String `tfsdk:"description"`
	Steps       types.List   `tfsdk:"steps"`
	Clauses     types.List   `tfsdk:"clauses"`
}

type environmentReleasePlanStepModel struct {
	Allocation     types.Int64 `tfsdk:"allocation"`
	DurationMillis types.Int64 `tfsdk:"duration_millis"`
}

type environmentReleasePlanClauseModel struct {
	Attribute   types.String `tfsdk:"attribute"`
	Op          types.String `tfsdk:"op"`
	Values      types.List   `tfsdk:"values"`
	ValueType   types.String `tfsdk:"value_type"`
	Negate      types.Bool   `tfsdk:"negate"`
	ContextKind types.String `tfsdk:"context_kind"`
}

type environmentReleasePlanMetricModel struct {
	IsGroup      types.Bool `tfsdk:"is_group"`
	AutoRollback types.Bool `tfsdk:"auto_rollback"`
}

var environmentReleasePlanStepAttrTypes = map[string]attr.Type{
	ALLOCATION:      types.Int64Type,
	DURATION_MILLIS: types.Int64Type,
}

var environmentReleasePlanStepObjectType = types.ObjectType{AttrTypes: environmentReleasePlanStepAttrTypes}

var environmentReleasePlanPhaseAttrTypes = map[string]attr.Type{
	DESCRIPTION: types.StringType,
	STEPS:       types.ListType{ElemType: environmentReleasePlanStepObjectType},
	CLAUSES:     types.ListType{ElemType: types.ObjectType{AttrTypes: frameworkClauseAttrTypes}},
}

var environmentReleasePlanPhaseObjectType = types.ObjectType{AttrTypes: environmentReleasePlanPhaseAttrTypes}

var environmentReleasePlanMetricAttrTypes = map[string]attr.Type{
	IS_GROUP:      types.BoolType,
	AUTO_ROLLBACK: types.BoolType,
}

var environmentReleasePlanMetricObjectType = types.ObjectType{AttrTypes: environmentReleasePlanMetricAttrTypes}

var environmentReleasePlanProgressiveReleaseAttrTypes = map[string]attr.Type{
	RANDOMIZATION_UNIT: types.StringType,
	ADVANCEMENT_MODE:   types.StringType,
	CLEANUP_BEHAVIOR:   types.StringType,
	PHASES:             types.ListType{ElemType: environmentReleasePlanPhaseObjectType},
	METRICS:            types.MapType{ElemType: environmentReleasePlanMetricObjectType},
}

// ── model ⇄ API conversion ──────────────────────────────────────────────────

func environmentReleasePlanProgressiveReleaseToAPI(ctx context.Context, obj types.Object) (environmentReleasePlanProgressiveRelease, diag.Diagnostics) {
	var diags diag.Diagnostics
	var out environmentReleasePlanProgressiveRelease
	if obj.IsNull() || obj.IsUnknown() {
		return out, diags
	}
	var m environmentReleasePlanProgressiveReleaseModel
	diags.Append(obj.As(ctx, &m, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return out, diags
	}
	out.RandomizationUnit = m.RandomizationUnit.ValueString()
	out.AdvancementMode = stringPointerFromAttr(m.AdvancementMode)
	out.CleanupBehavior = stringPointerFromAttr(m.CleanupBehavior)

	var phases []environmentReleasePlanPhaseModel
	diags.Append(m.Phases.ElementsAs(ctx, &phases, false)...)
	if diags.HasError() {
		return out, diags
	}
	out.Phases = make([]environmentReleasePlanPhase, 0, len(phases))
	for _, p := range phases {
		var steps []environmentReleasePlanStepModel
		diags.Append(p.Steps.ElementsAs(ctx, &steps, false)...)
		clauses, d := frameworkClausesFromList(ctx, p.Clauses)
		diags.Append(d...)
		if diags.HasError() {
			return out, diags
		}
		phase := environmentReleasePlanPhase{
			Description: stringPointerFromAttr(p.Description),
			Steps:       make([]environmentReleasePlanStep, 0, len(steps)),
			Clauses:     clauses,
		}
		for _, s := range steps {
			phase.Steps = append(phase.Steps, environmentReleasePlanStep{
				Allocation:     s.Allocation.ValueInt64(),
				DurationMillis: s.DurationMillis.ValueInt64(),
			})
		}
		out.Phases = append(out.Phases, phase)
	}

	if !m.Metrics.IsNull() && !m.Metrics.IsUnknown() {
		var metrics map[string]environmentReleasePlanMetricModel
		diags.Append(m.Metrics.ElementsAs(ctx, &metrics, false)...)
		if diags.HasError() {
			return out, diags
		}
		out.Metrics = make(map[string]environmentReleasePlanMetric, len(metrics))
		for k, v := range metrics {
			out.Metrics[k] = environmentReleasePlanMetric{
				IsGroup:      v.IsGroup.ValueBool(),
				AutoRollback: v.AutoRollback.ValueBool(),
			}
		}
	}
	return out, diags
}

// environmentReleasePlanProgressiveReleaseToObject maps the API shape back
// into the framework object. existing is the prior plan/state value, used to
// keep an explicit `metrics = {}` from reading back as null.
func environmentReleasePlanProgressiveReleaseToObject(ctx context.Context, api environmentReleasePlanProgressiveRelease, existing types.Object) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	var existingMetrics types.Map
	if !existing.IsNull() && !existing.IsUnknown() {
		var em environmentReleasePlanProgressiveReleaseModel
		diags.Append(existing.As(ctx, &em, basetypes.ObjectAsOptions{})...)
		existingMetrics = em.Metrics
	}
	phases, d := environmentReleasePlanPhasesToList(ctx, api.Phases)
	diags.Append(d...)
	metrics, d := environmentReleasePlanMetricsToMap(api.Metrics, existingMetrics)
	diags.Append(d...)
	m := environmentReleasePlanProgressiveReleaseModel{
		RandomizationUnit: types.StringValue(api.RandomizationUnit),
		AdvancementMode:   stringValueOrNullFromPointer(api.AdvancementMode),
		CleanupBehavior:   stringValueOrNullFromPointer(api.CleanupBehavior),
		Phases:            phases,
		Metrics:           metrics,
	}
	obj, d := types.ObjectValueFrom(ctx, environmentReleasePlanProgressiveReleaseAttrTypes, m)
	diags.Append(d...)
	return obj, diags
}

func environmentReleasePlanPhasesToList(ctx context.Context, phases []environmentReleasePlanPhase) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	elems := make([]attr.Value, 0, len(phases))
	for _, p := range phases {
		stepElems := make([]attr.Value, 0, len(p.Steps))
		for _, s := range p.Steps {
			obj, d := types.ObjectValue(environmentReleasePlanStepAttrTypes, map[string]attr.Value{
				ALLOCATION:      types.Int64Value(s.Allocation),
				DURATION_MILLIS: types.Int64Value(s.DurationMillis),
			})
			diags.Append(d...)
			stepElems = append(stepElems, obj)
		}
		steps, d := types.ListValue(environmentReleasePlanStepObjectType, stepElems)
		diags.Append(d...)
		clauses, d := frameworkClausesValue(ctx, p.Clauses)
		diags.Append(d...)
		obj, d := types.ObjectValue(environmentReleasePlanPhaseAttrTypes, map[string]attr.Value{
			DESCRIPTION: stringValueOrNullFromPointer(p.Description),
			STEPS:       steps,
			CLAUSES:     clauses,
		})
		diags.Append(d...)
		elems = append(elems, obj)
	}
	list, d := types.ListValue(environmentReleasePlanPhaseObjectType, elems)
	diags.Append(d...)
	return list, diags
}

func environmentReleasePlanMetricsToMap(metrics map[string]environmentReleasePlanMetric, existing types.Map) (types.Map, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(metrics) == 0 {
		if existing.IsNull() || existing.IsUnknown() {
			return types.MapNull(environmentReleasePlanMetricObjectType), diags
		}
		return types.MapValueMust(environmentReleasePlanMetricObjectType, map[string]attr.Value{}), diags
	}
	elems := make(map[string]attr.Value, len(metrics))
	for k, v := range metrics {
		obj, d := types.ObjectValue(environmentReleasePlanMetricAttrTypes, map[string]attr.Value{
			IS_GROUP:      types.BoolValue(v.IsGroup),
			AUTO_ROLLBACK: types.BoolValue(v.AutoRollback),
		})
		diags.Append(d...)
		elems[k] = obj
	}
	m, d := types.MapValue(environmentReleasePlanMetricObjectType, elems)
	diags.Append(d...)
	return m, diags
}

// environmentReleasePlanToModel fills the resource model from an API
// representation. existing progressive_release state is read from the model
// before it is overwritten.
func environmentReleasePlanToModel(ctx context.Context, projectKey, envKey string, rep *environmentReleasePlanRep, m *EnvironmentReleasePlanResourceModel, diags *diag.Diagnostics) {
	m.ID = types.StringValue(environmentReleasePlanID(projectKey, envKey, rep.Key))
	m.ProjectKey = types.StringValue(projectKey)
	m.EnvironmentKey = types.StringValue(envKey)
	m.Key = types.StringValue(rep.Key)
	m.Name = types.StringValue(rep.Name)
	m.Description = stringValueOrNullFromPointer(rep.Description)
	m.ReleaseMethod = types.StringValue(rep.ReleaseMethod)
	m.Version = types.Int64Value(rep.Version)
	isDefault := false
	if rep.Default != nil {
		isDefault = *rep.Default
	}
	m.IsDefault = types.BoolValue(isDefault)

	obj, d := environmentReleasePlanProgressiveReleaseToObject(ctx, rep.ProgressiveRelease, m.ProgressiveRelease)
	diags.Append(d...)
	m.ProgressiveRelease = obj
}

// validateEnvironmentReleasePlanPhaseClauses mirrors the API rule that every
// phase's clauses include the randomization unit clause (attribute `kind`, op
// `in`, values containing the randomization unit), so a missing clause fails
// at plan time instead of as a 400 on apply. Phases with unknown values are
// skipped and re-validated once they are known.
func validateEnvironmentReleasePlanPhaseClauses(ctx context.Context, progressive types.Object, diags *diag.Diagnostics) {
	if progressive.IsNull() || progressive.IsUnknown() {
		return
	}
	var m environmentReleasePlanProgressiveReleaseModel
	diags.Append(progressive.As(ctx, &m, basetypes.ObjectAsOptions{})...)
	if diags.HasError() || m.RandomizationUnit.IsNull() || m.RandomizationUnit.IsUnknown() || m.Phases.IsNull() || m.Phases.IsUnknown() {
		return
	}
	unit := m.RandomizationUnit.ValueString()

	var phases []environmentReleasePlanPhaseModel
	diags.Append(m.Phases.ElementsAs(ctx, &phases, false)...)
	if diags.HasError() {
		return
	}
	for i, p := range phases {
		if p.Clauses.IsNull() || p.Clauses.IsUnknown() {
			continue
		}
		var clauses []environmentReleasePlanClauseModel
		diags.Append(p.Clauses.ElementsAs(ctx, &clauses, false)...)
		if diags.HasError() {
			return
		}
		found, undecidable := false, false
		for _, c := range clauses {
			if c.Attribute.IsUnknown() || c.Op.IsUnknown() || c.Values.IsUnknown() {
				undecidable = true
				break
			}
			if c.Attribute.ValueString() != "kind" || c.Op.ValueString() != "in" {
				continue
			}
			values, d := stringSliceFromList(ctx, c.Values)
			diags.Append(d...)
			for _, v := range values {
				if v == unit {
					found = true
				}
			}
		}
		if found || undecidable {
			continue
		}
		diags.AddAttributeError(
			path.Root(PROGRESSIVE_RELEASE).AtName(PHASES).AtListIndex(i).AtName(CLAUSES),
			"Missing randomization unit clause",
			fmt.Sprintf("Each phase's clauses must include the randomization unit clause {attribute = %q, op = %q, values = [%q]} so the release only reaches contexts of that kind.", "kind", "in", unit),
		)
	}
}
