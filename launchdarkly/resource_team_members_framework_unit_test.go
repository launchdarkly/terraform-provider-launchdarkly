package launchdarkly

import (
	"fmt"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// entry builds a members-map entry with a role set, which satisfies the
// "at least one of role or custom_roles" rule. Every attribute is given a
// concrete type, matching how the framework decodes real configuration.
func entry(role string) teamMembersEntryModel {
	return teamMembersEntryModel{
		ID:             types.StringNull(),
		Email:          types.StringNull(),
		FirstName:      types.StringNull(),
		LastName:       types.StringNull(),
		Role:           types.StringValue(role),
		CustomRoles:    types.SetNull(types.StringType),
		TeamKeys:       types.SetNull(types.StringType),
		RoleAttributes: types.MapNull(types.ListType{ElemType: types.StringType}),
	}
}

func entryWithID(role, id string) teamMembersEntryModel {
	e := entry(role)
	e.ID = types.StringValue(id)
	return e
}

func TestValidateMemberBatch(t *testing.T) {
	t.Run("valid batch passes", func(t *testing.T) {
		require.NoError(t, validateMemberBatch(map[string]teamMembersEntryModel{
			"a@example.com": entry("writer"),
			"b@example.com": entry("reader"),
		}))
	})

	t.Run("empty batch rejected", func(t *testing.T) {
		require.Error(t, validateMemberBatch(nil))
		require.Error(t, validateMemberBatch(map[string]teamMembersEntryModel{}))
	})

	t.Run("uppercase map key rejected", func(t *testing.T) {
		err := validateMemberBatch(map[string]teamMembersEntryModel{"A@Example.com": entry("writer")})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "lowercase")
	})

	t.Run("non-email map key rejected", func(t *testing.T) {
		require.Error(t, validateMemberBatch(map[string]teamMembersEntryModel{"not-an-email": entry("writer")}))
	})

	t.Run("inner email must match map key when set", func(t *testing.T) {
		e := entry("writer")
		e.Email = types.StringValue("other@example.com")
		err := validateMemberBatch(map[string]teamMembersEntryModel{"a@example.com": e})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must equal its map key")
	})

	t.Run("inner email matching map key is fine", func(t *testing.T) {
		e := entry("writer")
		e.Email = types.StringValue("a@example.com")
		require.NoError(t, validateMemberBatch(map[string]teamMembersEntryModel{"a@example.com": e}))
	})

	t.Run("entry with neither role nor custom_roles rejected", func(t *testing.T) {
		bare := entry("")
		bare.Role = types.StringNull()
		err := validateMemberBatch(map[string]teamMembersEntryModel{"a@example.com": bare})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one of")
	})

	t.Run("custom_roles alone is sufficient", func(t *testing.T) {
		e := entry("")
		e.Role = types.StringNull()
		e.CustomRoles = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("some-role")})
		require.NoError(t, validateMemberBatch(map[string]teamMembersEntryModel{"a@example.com": e}))
	})

	t.Run("unknown values defer to apply", func(t *testing.T) {
		e := entry("")
		e.Email = types.StringUnknown()
		e.Role = types.StringUnknown()
		e.CustomRoles = types.SetUnknown(types.StringType)
		require.NoError(t, validateMemberBatch(map[string]teamMembersEntryModel{"a@example.com": e}),
			"unknown role/custom_roles must not fail validation")
	})

	t.Run("batch size bounds", func(t *testing.T) {
		exactly50 := map[string]teamMembersEntryModel{}
		for i := 0; i < teamMembersMaxBatchSize; i++ {
			exactly50[fmt.Sprintf("u%d@example.com", i)] = entry("reader")
		}
		require.NoError(t, validateMemberBatch(exactly50))

		exactly50[fmt.Sprintf("u%d@example.com", teamMembersMaxBatchSize)] = entry("reader")
		err := validateMemberBatch(exactly50)
		require.Error(t, err)
		assert.Contains(t, err.Error(), fmt.Sprintf("%d", teamMembersMaxBatchSize))
	})
}

func TestParseMembersConflict(t *testing.T) {
	t.Run("in-account duplicates are recoverable and normalized", func(t *testing.T) {
		body := []byte(`{"code":"email_already_exists_in_account","message":"...","invalid_emails":["a@x.com","B@X.com"]}`)
		emails, ok := parseMembersConflict(body)
		require.True(t, ok)
		assert.Equal(t, []string{"a@x.com", "b@x.com"}, emails)
	})

	t.Run("cross-account conflicts are not recoverable", func(t *testing.T) {
		// Multi-account membership is allowed today, so this error is a
		// legacy/race path. Treating it as recoverable would silently skip
		// members, so it must surface to the operator instead.
		body := []byte(`{"code":"email_taken_in_different_account","invalid_emails":["a@x.com"]}`)
		_, ok := parseMembersConflict(body)
		assert.False(t, ok)
	})

	t.Run("other error codes are not conflicts", func(t *testing.T) {
		_, ok := parseMembersConflict([]byte(`{"code":"seat_limit_reached"}`))
		assert.False(t, ok)
	})

	t.Run("malformed bodies are not conflicts", func(t *testing.T) {
		_, ok := parseMembersConflict([]byte(`not json`))
		assert.False(t, ok)
		_, ok = parseMembersConflict(nil)
		assert.False(t, ok)
	})

	t.Run("conflict code with no emails is not actionable", func(t *testing.T) {
		_, ok := parseMembersConflict([]byte(`{"code":"email_already_exists_in_account","invalid_emails":[]}`))
		assert.False(t, ok, "no emails means nothing to adopt or remove")
	})
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestDiffMemberBatches(t *testing.T) {
	t.Run("classifies added, removed, changed, and retained", func(t *testing.T) {
		state := map[string]teamMembersEntryModel{
			"keep@x.com":   entryWithID("reader", "id-keep"),
			"gone@x.com":   entryWithID("reader", "id-gone"),
			"change@x.com": entryWithID("reader", "id-change"),
		}
		plan := map[string]teamMembersEntryModel{
			"keep@x.com":   entry("reader"),
			"new@x.com":    entry("writer"),
			"change@x.com": entry("admin"),
		}

		d := diffMemberBatches(state, plan)
		assert.Equal(t, []string{"new@x.com"}, sortedKeys(d.toCreate))
		assert.Equal(t, []string{"id-gone"}, d.toDeleteIDs)
		assert.Equal(t, []string{"change@x.com"}, sortedKeys(d.toPatch))
		assert.Equal(t, map[string]string{"keep@x.com": "id-keep"}, d.retained)
	})

	t.Run("changed entries carry the prior member ID forward", func(t *testing.T) {
		state := map[string]teamMembersEntryModel{"change@x.com": entryWithID("reader", "id-change")}
		plan := map[string]teamMembersEntryModel{"change@x.com": entry("admin")}

		d := diffMemberBatches(state, plan)
		require.Contains(t, d.toPatch, "change@x.com")
		assert.Equal(t, "id-change", d.toPatch["change@x.com"].ID.ValueString(),
			"a patched entry must keep its ID or the update would lose track of the member")
	})

	t.Run("name-only differences are not changes", func(t *testing.T) {
		// LaunchDarkly does not let the provider update names, so treating a
		// name difference as a change would produce a patch that can never
		// converge.
		prior := entryWithID("reader", "id-1")
		prior.FirstName = types.StringValue("Old")
		desired := entry("reader")
		desired.FirstName = types.StringValue("New")

		d := diffMemberBatches(
			map[string]teamMembersEntryModel{"a@x.com": prior},
			map[string]teamMembersEntryModel{"a@x.com": desired},
		)
		assert.Empty(t, d.toPatch)
		assert.Equal(t, map[string]string{"a@x.com": "id-1"}, d.retained)
	})

	t.Run("entries without an ID are not queued for deletion", func(t *testing.T) {
		state := map[string]teamMembersEntryModel{"pending@x.com": entry("reader")}
		d := diffMemberBatches(state, map[string]teamMembersEntryModel{"other@x.com": entry("reader")})
		assert.Empty(t, d.toDeleteIDs, "there is nothing to delete without a member ID")
	})
}

func TestIsFullReplacement(t *testing.T) {
	state := map[string]teamMembersEntryModel{
		"a@x.com": entryWithID("reader", "id-a"),
		"b@x.com": entryWithID("reader", "id-b"),
	}

	t.Run("replacing every managed member is a full replacement", func(t *testing.T) {
		d := diffMemberBatches(state, map[string]teamMembersEntryModel{
			"new1@x.com": entry("reader"),
			"new2@x.com": entry("reader"),
		})
		assert.True(t, isFullReplacement(state, d))
	})

	t.Run("retaining one member is not a full replacement", func(t *testing.T) {
		d := diffMemberBatches(state, map[string]teamMembersEntryModel{
			"a@x.com":    entry("reader"),
			"new1@x.com": entry("reader"),
		})
		assert.False(t, isFullReplacement(state, d))
	})

	t.Run("empty prior state has nothing to protect", func(t *testing.T) {
		empty := map[string]teamMembersEntryModel{}
		d := diffMemberBatches(empty, map[string]teamMembersEntryModel{"new@x.com": entry("reader")})
		assert.False(t, isFullReplacement(empty, d))
	})

	t.Run("unmanaged entries do not count toward the total", func(t *testing.T) {
		mixed := map[string]teamMembersEntryModel{
			"managed@x.com":   entryWithID("reader", "id-1"),
			"unmanaged@x.com": entry("reader"),
		}
		d := diffMemberBatches(mixed, map[string]teamMembersEntryModel{"new@x.com": entry("reader")})
		assert.True(t, isFullReplacement(mixed, d),
			"the one member with an ID is being removed, so this replaces everything managed")
	})
}
