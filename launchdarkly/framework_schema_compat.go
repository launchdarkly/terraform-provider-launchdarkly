package launchdarkly

// framework_schema_compat.go is the terraform-plugin-framework analogue
// of schema_compat.go. The SDKv2 file exists because Crossplane's Upjet
// embeds this provider and strips deprecated attributes from the runtime
// schema; reads/writes against those keys then fail with very specific
// SDK error shapes that we have to swallow rather than propagate.
//
// Framework-side, the equivalent behaviour produces a different error
// shape because framework data accessors (State / Plan / Config) route
// through fwschemadata.SetAtPath, which emits an "<Description> Write
// Error" attribute diagnostic via diag.NewAttributeErrorDiagnostic when
// the requested path is not present in the schema (see
// terraform-plugin-framework@v1.9.0/internal/fwschemadata/data_set_at_path.go).
//
// Until Crossplane confirms whether their Upjet pipeline actually strips
// attributes from framework-served schemas the same way it does for
// SDKv2, this file ships defensively: callers route writes through
// stateSetSkipMissingKey / planSetSkipMissingKey so deprecated-attribute
// writes don't crash embedded users. If the Upjet investigation in
// Phase 0.6 (see docs/migration-schema-compat-upjet.md) concludes the
// shim is unnecessary, schedule deletion for Phase 5.2.
//
// Matchers are intentionally narrow — same discipline as schema_compat.go:
// they only swallow the specific "write error" shape framework emits for
// a missing-from-schema attribute, never a generic "AddAttributeError"
// from somewhere else in the diagnostics list.

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

// frameworkMissingAttrSummary is the diagnostic summary emitted by
// framework's data-set machinery when an attribute path resolves to
// nothing in the schema. The exact wording is "<Description> Write Error"
// where Description is "config", "plan", or "state". We match on the
// suffix so the helper handles all three accessors uniformly.
const frameworkMissingAttrSummary = "Write Error"

// frameworkMissingAttrDetailPrefix is the canonical prefix of the detail
// message framework emits when the requested attribute path does not
// match a node in the schema. We match against the prefix to remain
// resilient to wording tweaks in future framework releases.
const frameworkMissingAttrDetailPrefix = "An unexpected error was encountered trying to retrieve type information at a given path."

// isOmittedFrameworkAttrDiag reports whether the diagnostic matches the
// shape framework emits for an attribute path that's missing from the
// runtime schema. The attrPath argument scopes the matcher to one
// specific attribute so unrelated diagnostics that happen to share the
// summary suffix do not get swallowed.
func isOmittedFrameworkAttrDiag(d diag.Diagnostic, attrPath path.Path) bool {
	if d == nil {
		return false
	}
	if !strings.HasSuffix(d.Summary(), frameworkMissingAttrSummary) {
		return false
	}
	if !strings.HasPrefix(d.Detail(), frameworkMissingAttrDetailPrefix) {
		return false
	}
	if withPath, ok := d.(diag.DiagnosticWithPath); ok {
		return withPath.Path().Equal(attrPath)
	}
	return false
}

// filterOmittedAttrDiags returns a copy of diags with every diagnostic
// matching the missing-attr-at-path shape removed. The unmatched
// diagnostics flow through unchanged so unrelated errors continue to
// fail loudly.
func filterOmittedAttrDiags(diags diag.Diagnostics, attrPath path.Path) diag.Diagnostics {
	out := make(diag.Diagnostics, 0, len(diags))
	for _, d := range diags {
		if isOmittedFrameworkAttrDiag(d, attrPath) {
			continue
		}
		out = append(out, d)
	}
	return out
}

// stateSetSkipMissingKey runs state.SetAttribute and filters out the
// missing-attribute diagnostic for the supplied path. Use when writing
// to a deprecated attribute that Upjet (or another embedder) may have
// removed from the runtime schema. Other diagnostics are returned
// unchanged so unrelated errors still surface.
func stateSetSkipMissingKey(ctx context.Context, state *tfsdk.State, attrPath path.Path, val interface{}) diag.Diagnostics {
	diags := state.SetAttribute(ctx, attrPath, val)
	return filterOmittedAttrDiags(diags, attrPath)
}

// planSetSkipMissingKey is the plan equivalent of stateSetSkipMissingKey.
func planSetSkipMissingKey(ctx context.Context, plan *tfsdk.Plan, attrPath path.Path, val interface{}) diag.Diagnostics {
	diags := plan.SetAttribute(ctx, attrPath, val)
	return filterOmittedAttrDiags(diags, attrPath)
}
