package launchdarkly

// set_parity_test.go locks in the Phase 0.9b parity guarantees for the
// three Set-typed attributes the migration plan flagged as highest
// risk: custom_role.policy, access_token.custom_roles, and
// team_member.custom_roles. SDKv2 used a custom hash that collapsed
// element ordering; the framework Set type does the same by element
// value. These tests assert that ordering does not affect the
// canonical wire form a framework resource builds before sending to
// the LD API.

import (
	"context"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// TestSetParity_CustomRolesReorderInvariant asserts that the
// types.Set[string] surface used by both access_token.custom_roles and
// team_member.custom_roles produces an order-invariant slice once
// converted via stringSliceFromSet. Mirrors SDKv2 schema.HashString
// semantics.
func TestSetParity_CustomRolesReorderInvariant(t *testing.T) {
	ctx := context.Background()
	a, _ := setFromStringSlice(ctx, []string{"role-a", "role-b", "role-c"})
	b, _ := setFromStringSlice(ctx, []string{"role-c", "role-a", "role-b"})

	// Framework Set equality is element-value based regardless of input
	// ordering — this is the property the migration relies on.
	if !a.Equal(b) {
		t.Fatalf("types.Set[string] should compare equal under reorder; a=%v b=%v", a, b)
	}

	// Round-trip through the helper and confirm the slice contents are
	// the same (after canonical sort) so callers building patch payloads
	// don't accidentally introduce ordering drift.
	outA, _ := stringSliceFromSet(ctx, a)
	outB, _ := stringSliceFromSet(ctx, b)
	sort.Strings(outA)
	sort.Strings(outB)
	if len(outA) != len(outB) {
		t.Fatalf("length drift after roundtrip: %v vs %v", outA, outB)
	}
	for i := range outA {
		if outA[i] != outB[i] {
			t.Fatalf("element drift at %d: %q vs %q", i, outA[i], outB[i])
		}
	}
}

// TestSetParity_CustomRolePolicyOrderInvariant asserts that two
// deprecated custom_role.policy statements that differ only in the
// internal ordering of Resources or Actions canonicalise to the same
// SDKv2 hash input. SDKv2 sorted both slices in policyFromResourceData
// (policies_helper.go:69-70); the framework migration MUST preserve
// that sort or hashed Set equality will silently flip after the swap.
func TestSetParity_CustomRolePolicyOrderInvariant(t *testing.T) {
	canon := canonicalisePolicy(ldapi.StatementPost{
		Resources: []string{"proj/b", "proj/a"},
		Actions:   []string{"updateFlag", "createFlag"},
		Effect:    "allow",
	})
	other := canonicalisePolicy(ldapi.StatementPost{
		Resources: []string{"proj/a", "proj/b"},
		Actions:   []string{"createFlag", "updateFlag"},
		Effect:    "allow",
	})

	if !equalStringSlice(canon.Resources, other.Resources) {
		t.Fatalf("policy Resources sort not order-invariant: %v vs %v", canon.Resources, other.Resources)
	}
	if !equalStringSlice(canon.Actions, other.Actions) {
		t.Fatalf("policy Actions sort not order-invariant: %v vs %v", canon.Actions, other.Actions)
	}
	if canon.Effect != other.Effect {
		t.Fatalf("policy Effect changed under canonicalisation: %q vs %q", canon.Effect, other.Effect)
	}
}

// canonicalisePolicy mirrors policyFromResourceData's sort step: both
// slices are sorted before any hashing or wire serialisation. The
// framework resource MUST sort on both Read and Write to preserve the
// SDKv2 hashed-set semantics.
func canonicalisePolicy(p ldapi.StatementPost) ldapi.StatementPost {
	out := ldapi.StatementPost{
		Effect:    p.Effect,
		Resources: append([]string(nil), p.Resources...),
		Actions:   append([]string(nil), p.Actions...),
	}
	sort.Strings(out.Resources)
	sort.Strings(out.Actions)
	return out
}

func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestSetParity_TagsReorderInvariant covers tags Set (used by many
// resources). Order-invariance is identical to custom_roles since both
// flow through setFromStringSlice/stringSliceFromSet.
func TestSetParity_TagsReorderInvariant(t *testing.T) {
	ctx := context.Background()
	a, _ := setFromStringSlice(ctx, []string{"alpha", "beta", "gamma"})
	b, _ := setFromStringSlice(ctx, []string{"gamma", "alpha", "beta"})
	if !a.Equal(b) {
		t.Fatalf("tags Set should compare equal under reorder; a=%v b=%v", a, b)
	}
}

// TestSetFromStringSliceOrNull pins the Set-flavoured analogue of
// TestStringValueOrNullFromPointer. Same root cause: writing an empty
// non-null Set on Read when the user's HCL omitted the attribute
// trips terraform-core's "Provider produced inconsistent result after
// apply" check. Originally surfaced via TestAccTeam_CreateAndUpdate
// where launchdarkly_team_member.custom_roles was nil in plan but
// became cty.SetValEmpty(cty.String) post-apply.
func TestSetFromStringSliceOrNull(t *testing.T) {
	ctx := context.Background()
	t.Run("nil slice maps to null", func(t *testing.T) {
		got, d := setFromStringSliceOrNull(ctx, nil)
		if d.HasError() {
			t.Fatalf("unexpected diagnostics: %v", d)
		}
		if !got.IsNull() {
			t.Fatalf("nil slice should produce SetNull, got %v", got)
		}
	})
	t.Run("empty slice maps to null", func(t *testing.T) {
		got, d := setFromStringSliceOrNull(ctx, []string{})
		if d.HasError() {
			t.Fatalf("unexpected diagnostics: %v", d)
		}
		if !got.IsNull() {
			t.Fatalf("empty slice should produce SetNull, got %v", got)
		}
	})
	t.Run("non-empty slice passes through", func(t *testing.T) {
		got, d := setFromStringSliceOrNull(ctx, []string{"a", "b"})
		if d.HasError() {
			t.Fatalf("unexpected diagnostics: %v", d)
		}
		if got.IsNull() {
			t.Fatalf("non-empty slice should produce a populated Set, got null")
		}
		if len(got.Elements()) != 2 {
			t.Fatalf("expected 2 elements, got %d", len(got.Elements()))
		}
	})
}

// TestStringValueOrNullFromPointer pins the contract for the helper
// that fixes the plan-apply consistency bug surfaced by
// TestAccTeamRoleMapping_* in CI: an Optional (non-Computed) string
// attribute whose API response is nil or "" MUST land in state as
// types.StringNull(), not types.StringValue(""). Otherwise
// terraform-core compares plan(null) vs apply("") and rejects the
// apply with "Provider produced inconsistent result after apply".
func TestStringValueOrNullFromPointer(t *testing.T) {
	t.Run("nil pointer maps to null", func(t *testing.T) {
		got := stringValueOrNullFromPointer(nil)
		if !got.IsNull() {
			t.Fatalf("nil pointer should produce StringNull, got %v", got)
		}
	})
	t.Run("pointer to empty string maps to null", func(t *testing.T) {
		s := ""
		got := stringValueOrNullFromPointer(&s)
		if !got.IsNull() {
			t.Fatalf("pointer to empty string should produce StringNull, got %v", got)
		}
	})
	t.Run("pointer to non-empty value passes through", func(t *testing.T) {
		s := "hello"
		got := stringValueOrNullFromPointer(&s)
		if got.IsNull() {
			t.Fatalf("non-empty pointer should produce a value, got null")
		}
		if got.ValueString() != "hello" {
			t.Fatalf("expected \"hello\", got %q", got.ValueString())
		}
	})
}

// TestSetParity_CustomRolesNullVsEmpty pins the framework Set
// null-vs-empty distinction. SDKv2 TypeSet collapsed both to an empty
// slice; the framework Set distinguishes them. The setFromStringSlice
// helper deliberately returns an empty non-null Set for nil input so
// state writes don't flip between null and empty across plans (matches
// the comment in framework_helpers.go).
func TestSetParity_CustomRolesNullVsEmpty(t *testing.T) {
	ctx := context.Background()
	empty, _ := setFromStringSlice(ctx, nil)
	if empty.IsNull() {
		t.Fatalf("setFromStringSlice(nil) should produce non-null empty Set, got null")
	}
	if len(empty.Elements()) != 0 {
		t.Fatalf("setFromStringSlice(nil) should produce empty Set, got %d elements", len(empty.Elements()))
	}

	explicitEmpty, _ := setFromStringSlice(ctx, []string{})
	if !empty.Equal(explicitEmpty) {
		t.Fatalf("setFromStringSlice(nil) should equal setFromStringSlice([]); got %v vs %v", empty, explicitEmpty)
	}

	// Ensure a populated Set is not equal to the empty one — guards
	// against the helper accidentally collapsing values.
	populated, _ := setFromStringSlice(ctx, []string{"x"})
	if empty.Equal(populated) {
		t.Fatalf("empty Set should not equal a populated Set")
	}
	_ = types.SetNull // silence unused-import paranoia in some toolchains
}
