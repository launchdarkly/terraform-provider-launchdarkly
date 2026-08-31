package launchdarkly

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// entry builds a members-map entry with a role set, which satisfies the
// "at least one of role or custom_roles" rule.
func entry(role string) teamMembersEntryModel {
	return teamMembersEntryModel{
		Email:       types.StringNull(),
		Role:        types.StringValue(role),
		CustomRoles: types.SetNull(types.StringType),
		TeamKeys:    types.SetNull(types.StringType),
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
		bare := teamMembersEntryModel{
			Email:       types.StringNull(),
			Role:        types.StringNull(),
			CustomRoles: types.SetNull(types.StringType),
			TeamKeys:    types.SetNull(types.StringType),
		}
		err := validateMemberBatch(map[string]teamMembersEntryModel{"a@example.com": bare})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one of")
	})

	t.Run("custom_roles alone is sufficient", func(t *testing.T) {
		e := teamMembersEntryModel{
			Email:       types.StringNull(),
			Role:        types.StringNull(),
			CustomRoles: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("some-role")}),
			TeamKeys:    types.SetNull(types.StringType),
		}
		require.NoError(t, validateMemberBatch(map[string]teamMembersEntryModel{"a@example.com": e}))
	})

	t.Run("unknown values defer to apply", func(t *testing.T) {
		e := teamMembersEntryModel{
			Email:       types.StringUnknown(),
			Role:        types.StringUnknown(),
			CustomRoles: types.SetUnknown(types.StringType),
			TeamKeys:    types.SetNull(types.StringType),
		}
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
