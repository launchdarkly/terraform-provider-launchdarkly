package launchdarkly

import (
	"fmt"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	ldapi "github.com/launchdarkly/api-client-go/v24"
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

	t.Run("inner email must match map key case exactly", func(t *testing.T) {
		// The API reports emails lowercased, so a differently-cased configured
		// email would plan one casing and apply another.
		e := entry("writer")
		e.Email = types.StringValue("A@example.com")
		err := validateMemberBatch(map[string]teamMembersEntryModel{"a@example.com": e})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exactly")
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

func TestPatchableAttrsDiffer(t *testing.T) {
	withTeams := func(role string, teams ...string) teamMembersEntryModel {
		e := entry(role)
		set, _ := types.SetValueFrom(ctxBackground(), types.StringType, teams)
		e.TeamKeys = set
		return e
	}

	t.Run("team-only difference is not patchable", func(t *testing.T) {
		// Team membership is reconciled with grouped per-team requests. If a
		// team-only difference triggered a member PATCH, adopting a 50-member
		// batch whose teams are missing would cost 50 wasted requests.
		a := withTeams("reader", "team-a", "team-b")
		b := withTeams("reader")
		assert.False(t, patchableAttrsDiffer(a, b))
		assert.True(t, memberAttrsDiffer(a, b), "diff classification must still see the team change")
	})

	t.Run("role difference is patchable", func(t *testing.T) {
		assert.True(t, patchableAttrsDiffer(entry("reader"), entry("writer")))
	})

	t.Run("identical entries differ in neither", func(t *testing.T) {
		assert.False(t, patchableAttrsDiffer(entry("reader"), entry("reader")))
		assert.False(t, memberAttrsDiffer(entry("reader"), entry("reader")))
	})
}

func TestMarkNewEntriesComputedUnknown(t *testing.T) {
	objType := teamMembersEntryObjectType()
	toObj := func(t *testing.T, e teamMembersEntryModel) attr.Value {
		v, d := types.ObjectValueFrom(ctxBackground(), objType.AttrTypes, e)
		require.False(t, d.HasError(), "diags: %v", d)
		return v
	}

	t.Run("new entry with null role and id gets both planned unknown", func(t *testing.T) {
		// A member declared with only custom_roles plans role as null; the API
		// then defaults it (reader) and hydration writing that back would fail
		// plan-versus-apply consistency, exactly like the minted id.
		e := entry("")
		e.Role = types.StringNull()
		set, _ := types.SetValueFrom(ctxBackground(), types.StringType, []string{"some-role"})
		e.CustomRoles = set
		planned, d := types.MapValue(objType, map[string]attr.Value{"new@example.com": toObj(t, e)})
		require.False(t, d.HasError())

		out, d := markNewEntriesComputedUnknown(objType, planned, map[string]teamMembersEntryModel{})
		require.False(t, d.HasError(), "diags: %v", d)
		attrs := out.Elements()["new@example.com"].(types.Object).Attributes()
		assert.True(t, attrs[ID].IsUnknown(), "id must be planned unknown")
		assert.True(t, attrs[ROLE].IsUnknown(), "null role on a new entry must be planned unknown")
	})

	t.Run("explicit role on a new entry is left alone", func(t *testing.T) {
		planned, d := types.MapValue(objType, map[string]attr.Value{"new@example.com": toObj(t, entry("writer"))})
		require.False(t, d.HasError())

		out, d := markNewEntriesComputedUnknown(objType, planned, map[string]teamMembersEntryModel{})
		require.False(t, d.HasError())
		attrs := out.Elements()["new@example.com"].(types.Object).Attributes()
		assert.True(t, attrs[ID].IsUnknown())
		assert.Equal(t, types.StringValue("writer"), attrs[ROLE])
	})

	t.Run("existing entries untouched", func(t *testing.T) {
		e := entry("")
		e.Role = types.StringNull()
		planned, d := types.MapValue(objType, map[string]attr.Value{"old@example.com": toObj(t, e)})
		require.False(t, d.HasError())

		prior := map[string]teamMembersEntryModel{"old@example.com": entryWithID("reader", "5f0cd446a77cba0b4c5644a7")}
		out, d := markNewEntriesComputedUnknown(objType, planned, prior)
		require.False(t, d.HasError())
		attrs := out.Elements()["old@example.com"].(types.Object).Attributes()
		assert.True(t, attrs[ROLE].IsNull(), "existing entries keep UseStateForUnknown semantics")
	})
}

func TestBatchValidatorDefersUnknownMembersMap(t *testing.T) {
	// A members map built from another resource's apply-time output is wholly
	// unknown at plan time. Validation must defer to apply, not fail the plan
	// with a decode error.
	ctx := ctxBackground()
	var schemaResp resource.SchemaResponse
	(&TeamMembersResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())

	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	vals := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, at := range objType.AttributeTypes {
		if name == MEMBERS {
			vals[name] = tftypes.NewValue(at, tftypes.UnknownValue)
			continue
		}
		vals[name] = tftypes.NewValue(at, nil)
	}

	req := resource.ValidateConfigRequest{Config: tfsdk.Config{
		Raw:    tftypes.NewValue(objType, vals),
		Schema: schemaResp.Schema,
	}}
	var resp resource.ValidateConfigResponse
	teamMembersBatchValidator{}.ValidateResource(ctx, req, &resp)
	assert.False(t, resp.Diagnostics.HasError(),
		"unknown members map must defer validation, got: %v", resp.Diagnostics)
}

func TestExcludePlannedIDs(t *testing.T) {
	t.Run("rename keeps the adopted member alive", func(t *testing.T) {
		// A member changed email; the operator renamed the map key and the new
		// key adopted the SAME member ID. The old key's deletion must be
		// skipped or the just-adopted person is deleted.
		members := map[string]teamMembersEntryModel{
			"new@example.com":   entryWithID("reader", "5f0cd446a77cba0b4c5644a7"),
			"other@example.com": entryWithID("reader", "5f0cd446a77cba0b4c5644a8"),
		}
		out := excludePlannedIDs([]string{"5f0cd446a77cba0b4c5644a7"}, members)
		assert.Empty(t, out)
	})

	t.Run("genuine removals still delete", func(t *testing.T) {
		members := map[string]teamMembersEntryModel{
			"keep@example.com": entryWithID("reader", "5f0cd446a77cba0b4c5644a7"),
		}
		out := excludePlannedIDs([]string{"5f0cd446a77cba0b4c5644b1", "5f0cd446a77cba0b4c5644b2"}, members)
		assert.Equal(t, []string{"5f0cd446a77cba0b4c5644b1", "5f0cd446a77cba0b4c5644b2"}, out)
	})
}

func TestModifyPlanDefersUnknownMembersMap(t *testing.T) {
	// Mirrors TestBatchValidatorDefersUnknownMembersMap: a wholly unknown
	// members map must not fail planning in ModifyPlan either.
	ctx := ctxBackground()
	var schemaResp resource.SchemaResponse
	(&TeamMembersResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())

	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	vals := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, at := range objType.AttributeTypes {
		if name == MEMBERS {
			vals[name] = tftypes.NewValue(at, tftypes.UnknownValue)
			continue
		}
		vals[name] = tftypes.NewValue(at, nil)
	}
	raw := tftypes.NewValue(objType, vals)

	req := resource.ModifyPlanRequest{
		Plan:  tfsdk.Plan{Raw: raw, Schema: schemaResp.Schema},
		State: tfsdk.State{Raw: tftypes.NewValue(objType, nil), Schema: schemaResp.Schema},
	}
	resp := resource.ModifyPlanResponse{Plan: req.Plan}
	(&TeamMembersResource{}).ModifyPlan(ctx, req, &resp)
	assert.False(t, resp.Diagnostics.HasError(),
		"unknown members map must defer planning, got: %v", resp.Diagnostics)
}

func TestRefreshPreservesEmptyRoleAttributes(t *testing.T) {
	// An explicitly configured role_attributes = {} must stay an empty map
	// after hydration, not become null — that mismatch fails plan-versus-apply
	// consistency after the write already succeeded.
	ctx := ctxBackground()
	member := ldapi.Member{Id: "5f0cd446a77cba0b4c5644a7", Email: "a@example.com", Role: "reader"}

	t.Run("configured empty map stays empty", func(t *testing.T) {
		entry := entry("reader")
		entry.RoleAttributes = types.MapValueMust(types.ListType{ElemType: types.StringType}, map[string]attr.Value{})
		var diags diag.Diagnostics
		refreshEntryFromMember(ctx, "a@example.com", &entry, &member, newCustomRoleResolver(nil), &diags)
		require.False(t, diags.HasError(), "diags: %v", diags)
		assert.False(t, entry.RoleAttributes.IsNull())
		assert.Len(t, entry.RoleAttributes.Elements(), 0)
	})

	t.Run("omitted role attributes stay null", func(t *testing.T) {
		entry := entry("reader")
		var diags diag.Diagnostics
		refreshEntryFromMember(ctx, "a@example.com", &entry, &member, newCustomRoleResolver(nil), &diags)
		require.False(t, diags.HasError(), "diags: %v", diags)
		assert.True(t, entry.RoleAttributes.IsNull())
	})
}
