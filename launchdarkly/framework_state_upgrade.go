package launchdarkly

// framework_state_upgrade.go holds helpers used by per-resource
// schema-version 0 → 1 state upgraders that rewrite the SDKv2 zero-value
// shape ("", [], {}) into framework null semantics.
//
// Background: the pre-v3 SDKv2 provider serialised unset Optional
// attributes as their Go zero value rather than as null. The framework
// port reads the same API responses but returns null for the same
// "user-didn't-set-this" case, producing spurious "[] -> null" /
// "\"\" -> null" plan diffs on first refresh against pre-v3 state. The
// upgraders here migrate state in-place once on first read so subsequent
// plans see the correct null shape and no diff.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// nullIfEmptyString returns a null types.String when v carries the empty
// string. Pass-through for null / unknown.
func nullIfEmptyString(v types.String) types.String {
	if v.IsNull() || v.IsUnknown() {
		return v
	}
	if v.ValueString() == "" {
		return types.StringNull()
	}
	return v
}

// nullIfEmptyList returns a typed null List when l is a non-null
// zero-length list. Element type is preserved from the input so the
// upgraded state matches the current schema's expected list shape.
func nullIfEmptyList(ctx context.Context, l types.List) types.List {
	if l.IsNull() || l.IsUnknown() {
		return l
	}
	if len(l.Elements()) == 0 {
		return types.ListNull(l.ElementType(ctx))
	}
	return l
}

// nullIfEmptySet is nullIfEmptyList for types.Set.
func nullIfEmptySet(ctx context.Context, s types.Set) types.Set {
	if s.IsNull() || s.IsUnknown() {
		return s
	}
	if len(s.Elements()) == 0 {
		return types.SetNull(s.ElementType(ctx))
	}
	return s
}

// nullIfEmptyMap is nullIfEmptyList for types.Map.
func nullIfEmptyMap(ctx context.Context, m types.Map) types.Map {
	if m.IsNull() || m.IsUnknown() {
		return m
	}
	if len(m.Elements()) == 0 {
		return types.MapNull(m.ElementType(ctx))
	}
	return m
}
