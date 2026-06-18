package launchdarkly

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// forceBetaAPIVersionTransport rewrites the LD-API-Version header to exactly
// "beta" on every outbound request.
//
// The generated ReleasePipelinesBetaApi request builders do not expose a
// per-request .LDAPIVersion("beta") setter, so the only client-side hook is
// the configuration's default header. That alone is insufficient: the
// generated prepareRequest unconditionally *appends* "20240415" to
// LD-API-Version when the operation did not set it, and then *appends* the
// configured default — yielding a two-valued header ("20240415", "beta") whose
// first value (the stable version) wins server-side and 404s the beta route.
// Rewriting at the transport layer with Header.Set collapses it to a single
// "beta" value. Forcing it here (rather than via AddDefaultHeader) is what
// makes the beta endpoints resolve.
type forceBetaAPIVersionTransport struct {
	base http.RoundTripper
}

func (t *forceBetaAPIVersionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("LD-API-Version", "beta")
	return t.base.RoundTrip(req)
}

// newReleasePipelineBetaClient returns a beta-configured client for the
// release-pipelines endpoints.
func newReleasePipelineBetaClient(c *Client) (*Client, error) {
	beta, err := newBetaClient(c.apiKey, c.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		return nil, err
	}
	for _, cfg := range []*ldapi.Configuration{beta.ld.GetConfig(), beta.ld404Retry.GetConfig()} {
		base := cfg.HTTPClient.Transport
		if base == nil {
			base = http.DefaultTransport
		}
		cfg.HTTPClient.Transport = &forceBetaAPIVersionTransport{base: base}
	}
	return beta, nil
}

// Attribute type maps for the nested phases/audiences structures. These must
// stay in lockstep with the schema declarations in
// resource_release_pipeline_framework.go.
var releasePipelineAudienceConfigAttrTypes = map[string]attr.Type{
	NOTIFY_MEMBER_IDS: types.SetType{ElemType: types.StringType},
	NOTIFY_TEAM_KEYS:  types.SetType{ElemType: types.StringType},
	RELEASE_STRATEGY:  types.StringType,
	REQUIRE_APPROVAL:  types.BoolType,
}

var releasePipelineAudienceAttrTypes = map[string]attr.Type{
	CONFIGURATION:   types.ObjectType{AttrTypes: releasePipelineAudienceConfigAttrTypes},
	ENVIRONMENT_KEY: types.StringType,
	NAME:            types.StringType,
	SEGMENT_KEYS:    types.SetType{ElemType: types.StringType},
}

var releasePipelinePhaseAttrTypes = map[string]attr.Type{
	AUDIENCES: types.ListType{ElemType: types.ObjectType{AttrTypes: releasePipelineAudienceAttrTypes}},
	NAME:      types.StringType,
}

// releasePipelinePhaseModel mirrors one element of the `phases` nested
// attribute.
type releasePipelinePhaseModel struct {
	Name      types.String `tfsdk:"name"`
	Audiences types.List   `tfsdk:"audiences"`
}

// releasePipelineAudienceModel mirrors one element of a phase's `audiences`
// nested attribute.
type releasePipelineAudienceModel struct {
	EnvironmentKey types.String `tfsdk:"environment_key"`
	Name           types.String `tfsdk:"name"`
	SegmentKeys    types.Set    `tfsdk:"segment_keys"`
	Configuration  types.Object `tfsdk:"configuration"`
}

// releasePipelineAudienceConfigModel mirrors an audience's optional
// `configuration` object.
type releasePipelineAudienceConfigModel struct {
	ReleaseStrategy types.String `tfsdk:"release_strategy"`
	RequireApproval types.Bool   `tfsdk:"require_approval"`
	NotifyMemberIDs types.Set    `tfsdk:"notify_member_ids"`
	NotifyTeamKeys  types.Set    `tfsdk:"notify_team_keys"`
}

// releasePipelineIdToKeys splits a composite release pipeline ID into its
// project key and pipeline key. The expected format is
// `project_key/pipeline_key`.
func releasePipelineIdToKeys(id string) (projectKey string, pipelineKey string, err error) {
	if strings.Count(id, "/") != 1 {
		return "", "", fmt.Errorf("found unexpected release pipeline id format: %q expected format: 'project_key/pipeline_key'", id)
	}
	parts := strings.SplitN(id, "/", 2)
	if parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("found unexpected release pipeline id format: %q expected format: 'project_key/pipeline_key'", id)
	}
	return parts[0], parts[1], nil
}

// releasePipelineAudienceEnvKey extracts the environment key from an audience
// read back from the API. The beta API returns the environment as a nested
// summary object ({key, name, color, _links}); should a future API revision
// flatten it to `environmentKey`, the value lands in AdditionalProperties — we
// read the typed field first so a client regen takes over transparently
// (observed shape 2026-06-18).
func releasePipelineAudienceEnvKey(a ldapi.Audience) string {
	if a.Environment != nil && a.Environment.Key != "" {
		return a.Environment.Key
	}
	if k, ok := a.AdditionalProperties["environmentKey"].(string); ok {
		return k
	}
	return ""
}

// releasePipelinePhaseInputsFromModels converts the Terraform `phases` list
// into the generated API input type, preserving order (phase order is the
// rollout order). Used for both POST (create) and PUT (update) — both accept
// []CreatePhaseInput.
func releasePipelinePhaseInputsFromModels(ctx context.Context, phases []releasePipelinePhaseModel) ([]ldapi.CreatePhaseInput, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := make([]ldapi.CreatePhaseInput, 0, len(phases))
	for _, p := range phases {
		var audienceModels []releasePipelineAudienceModel
		diags.Append(p.Audiences.ElementsAs(ctx, &audienceModels, false)...)

		audiences := make([]ldapi.AudiencePost, 0, len(audienceModels))
		for _, a := range audienceModels {
			ap := ldapi.AudiencePost{
				EnvironmentKey: a.EnvironmentKey.ValueString(),
				Name:           a.Name.ValueString(),
			}
			segs, d := stringSliceFromSet(ctx, a.SegmentKeys)
			diags.Append(d...)
			if len(segs) > 0 {
				ap.SegmentKeys = segs
			}
			if !a.Configuration.IsNull() && !a.Configuration.IsUnknown() {
				var cfg releasePipelineAudienceConfigModel
				diags.Append(a.Configuration.As(ctx, &cfg, basetypes.ObjectAsOptions{})...)
				ac := ldapi.AudienceConfiguration{
					ReleaseStrategy: cfg.ReleaseStrategy.ValueString(),
					RequireApproval: cfg.RequireApproval.ValueBool(),
				}
				memberIDs, d := stringSliceFromSet(ctx, cfg.NotifyMemberIDs)
				diags.Append(d...)
				if len(memberIDs) > 0 {
					ac.NotifyMemberIds = memberIDs
				}
				teamKeys, d := stringSliceFromSet(ctx, cfg.NotifyTeamKeys)
				diags.Append(d...)
				if len(teamKeys) > 0 {
					ac.NotifyTeamKeys = teamKeys
				}
				ap.Configuration = &ac
			}
			audiences = append(audiences, ap)
		}
		out = append(out, ldapi.CreatePhaseInput{
			Name:      p.Name.ValueString(),
			Audiences: audiences,
		})
	}
	return out, diags
}

// releasePipelineConfigObject builds the Terraform `configuration` object from
// the API representation of an audience configuration.
func releasePipelineConfigObject(ctx context.Context, c *ldapi.AudienceConfiguration) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	memberIDs, d := setFromStringSliceOrNull(ctx, c.NotifyMemberIds)
	diags.Append(d...)
	teamKeys, d := setFromStringSliceOrNull(ctx, c.NotifyTeamKeys)
	diags.Append(d...)
	obj, d := types.ObjectValue(releasePipelineAudienceConfigAttrTypes, map[string]attr.Value{
		RELEASE_STRATEGY:  types.StringValue(c.ReleaseStrategy),
		REQUIRE_APPROVAL:  types.BoolValue(c.RequireApproval),
		NOTIFY_MEMBER_IDS: memberIDs,
		NOTIFY_TEAM_KEYS:  teamKeys,
	})
	diags.Append(d...)
	return obj, diags
}

// releasePipelinePhasesToList converts the API representation of a release
// pipeline's phases into a Terraform list value, preserving API order.
//
// planPhases carries the planned/prior `phases` value so the read can preserve
// the user's null-vs-empty intent for Optional nested attributes (segment_keys
// and the configuration object). On import planPhases is null/unknown — there
// the API response is the only source of truth, so everything is populated.
func releasePipelinePhasesToList(ctx context.Context, apiPhases []ldapi.Phase, planPhases types.List) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	phaseObjType := types.ObjectType{AttrTypes: releasePipelinePhaseAttrTypes}
	audienceObjType := types.ObjectType{AttrTypes: releasePipelineAudienceAttrTypes}

	hasPlan := !planPhases.IsNull() && !planPhases.IsUnknown()
	var planPhaseModels []releasePipelinePhaseModel
	if hasPlan {
		diags.Append(planPhases.ElementsAs(ctx, &planPhaseModels, false)...)
	}

	phaseElems := make([]attr.Value, 0, len(apiPhases))
	for pi, p := range apiPhases {
		// Plan audiences for this phase, when a plan is available and aligned.
		var planAudienceModels []releasePipelineAudienceModel
		if hasPlan && pi < len(planPhaseModels) {
			diags.Append(planPhaseModels[pi].Audiences.ElementsAs(ctx, &planAudienceModels, false)...)
		}

		audienceElems := make([]attr.Value, 0, len(p.Audiences))
		for ai, a := range p.Audiences {
			var planAudience *releasePipelineAudienceModel
			if ai < len(planAudienceModels) {
				planAudience = &planAudienceModels[ai]
			}

			// segment_keys: preserve plan intent when refreshing, full
			// fidelity on import.
			var segSet types.Set
			var segDiags diag.Diagnostics
			if hasPlan && planAudience != nil {
				segSet, segDiags = setFromStringSlicePreservingPlan(ctx, a.SegmentKeys, planAudience.SegmentKeys)
			} else {
				segSet, segDiags = setFromStringSliceOrNull(ctx, a.SegmentKeys)
			}
			diags.Append(segDiags...)

			// configuration: when refreshing and the user omitted it, keep it
			// null even if the API echoes a default object — this prevents
			// "inconsistent result after apply". On import we populate from the
			// API response.
			configObj := types.ObjectNull(releasePipelineAudienceConfigAttrTypes)
			if a.Configuration != nil {
				planConfigNull := hasPlan && planAudience != nil && planAudience.Configuration.IsNull()
				if !planConfigNull {
					var d diag.Diagnostics
					configObj, d = releasePipelineConfigObject(ctx, a.Configuration)
					diags.Append(d...)
				}
			}

			audienceObj, d := types.ObjectValue(releasePipelineAudienceAttrTypes, map[string]attr.Value{
				ENVIRONMENT_KEY: types.StringValue(releasePipelineAudienceEnvKey(a)),
				NAME:            types.StringValue(a.Name),
				SEGMENT_KEYS:    segSet,
				CONFIGURATION:   configObj,
			})
			diags.Append(d...)
			audienceElems = append(audienceElems, audienceObj)
		}

		audienceList, d := types.ListValue(audienceObjType, audienceElems)
		diags.Append(d...)

		phaseObj, d := types.ObjectValue(releasePipelinePhaseAttrTypes, map[string]attr.Value{
			NAME:      types.StringValue(p.Name),
			AUDIENCES: audienceList,
		})
		diags.Append(d...)
		phaseElems = append(phaseElems, phaseObj)
	}

	list, d := types.ListValue(phaseObjType, phaseElems)
	diags.Append(d...)
	return list, diags
}
