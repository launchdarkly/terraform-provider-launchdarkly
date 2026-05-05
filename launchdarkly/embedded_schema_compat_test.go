package launchdarkly

// Tests for embedded/Upjet-stripped schema compatibility: project CSA fallback guard,
// nil-safety of optional blocks on custom_role and access_token, and ID-derived fallbacks for
// env_key / custom-role key. None of these require TF_ACC, an LD access token, or the live API
// client; they exercise pure helper functions and the extracted patch builders.

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go/v22"
	"github.com/stretchr/testify/require"
)

// embeddedProjectSchema simulates an Upjet-stripped schema where the deprecated IIS attribute and
// the entire DEFAULT_CLIENT_SIDE_AVAILABILITY block are absent from the runtime schema.
func embeddedProjectSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		KEY:  {Type: schema.TypeString, Required: true, ForceNew: true},
		NAME: {Type: schema.TypeString, Required: true},
		TAGS: {
			Type:     schema.TypeSet,
			Optional: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
			Set:      schema.HashString,
		},
		REQUIRE_VIEW_ASSOCIATION_FOR_NEW_FLAGS:    {Type: schema.TypeBool, Optional: true, Default: false},
		REQUIRE_VIEW_ASSOCIATION_FOR_NEW_SEGMENTS: {Type: schema.TypeBool, Optional: true, Default: false},
	}
}

// containsCSAOp returns whether the given patch ops include any replace targeting
// /defaultClientSideAvailability.
func containsCSAOp(ops []ldapi.PatchOperation) bool {
	for _, op := range ops {
		if op.Path == "/defaultClientSideAvailability" {
			return true
		}
	}
	return false
}

// When the embedded schema strips IIS and CSA, a tags-only update must NOT emit a fallback
// /defaultClientSideAvailability patch. This is the regression covered by PR review item 1.
func TestBuildProjectUpdatePatches_embeddedSchemaOmitsCSAFallback(t *testing.T) {
	t.Parallel()

	d := schema.TestResourceDataRaw(t, embeddedProjectSchema(), map[string]interface{}{
		KEY:  "crossplane-project",
		NAME: "Crossplane Project",
		TAGS: []interface{}{"managed-by-crossplane"},
	})

	ops := buildProjectUpdatePatches(d)
	require.NotEmpty(t, ops)
	require.False(t, containsCSAOp(ops),
		"expected no /defaultClientSideAvailability op when embedded schema omits IIS and CSA; got %+v", ops)

	var paths []string
	for _, op := range ops {
		paths = append(paths, op.Path)
	}
	require.Contains(t, paths, "/name")
	require.Contains(t, paths, "/tags")
}

// When the full schema is in use but neither IIS nor CSA is set in config, the provider keeps its
// historical behavior of forcing /defaultClientSideAvailability to API defaults so downstream
// reads don't drift. This guards against the regression guard from item 1 going too far.
func TestBuildProjectUpdatePatches_fullSchemaKeepsCSAFallback(t *testing.T) {
	t.Parallel()

	d := schema.TestResourceDataRaw(t, resourceProject().Schema, map[string]interface{}{
		KEY:  "p",
		NAME: "Project",
		ENVIRONMENTS: []interface{}{
			map[string]interface{}{
				KEY:   "production",
				NAME:  "Production",
				COLOR: "417505",
			},
		},
	})

	ops := buildProjectUpdatePatches(d)
	require.True(t, containsCSAOp(ops),
		"full schema with no IIS/CSA in config should still emit fallback CSA op; got %+v", ops)
}

// resourceCustomRole.Schema with no policy/policy_statements blocks must not panic when the
// nil-safe helpers iterate over their empty values.
func TestPolicyStatementsFromResourceData_emptyDoesNotPanic(t *testing.T) {
	t.Parallel()

	d := schema.TestResourceDataRaw(t, resourceCustomRole().Schema, map[string]interface{}{
		KEY:              "role",
		NAME:             "Role",
		BASE_PERMISSIONS: "reader",
	})

	require.NotPanics(t, func() {
		_, _ = policyStatementsFromResourceData(getOptionalInterfaceSlice(d, POLICY_STATEMENTS))
		_ = policiesFromResourceData(d)
	})
}

// Access token has multiple optional set/list blocks (custom_roles, policy_statements,
// inline_roles). With none configured, validation must succeed without panicking.
func TestAccessTokenValidate_emptyOptionalBlocksNoPanic(t *testing.T) {
	t.Parallel()

	d := schema.TestResourceDataRaw(t, resourceAccessToken().Schema, map[string]interface{}{
		NAME: "tok",
		ROLE: "reader",
	})

	require.NotPanics(t, func() {
		_ = validateAccessTokenResource(d)
	})
}

// effectiveEnvKey error path: id is empty AND env_key attribute is empty.
func TestEffectiveEnvKey_errorWhenEmpty(t *testing.T) {
	t.Parallel()

	d := schema.TestResourceDataRaw(t, map[string]*schema.Schema{
		ENV_KEY: {Type: schema.TypeString, Optional: true},
	}, map[string]interface{}{})

	_, err := effectiveEnvKey(d)
	require.Error(t, err)
}

// effectiveEnvKey error path: id is malformed (wrong number of slashes).
func TestEffectiveEnvKey_errorWhenIDMalformed(t *testing.T) {
	t.Parallel()

	d := schema.TestResourceDataRaw(t, map[string]*schema.Schema{
		ENV_KEY: {Type: schema.TypeString, Optional: true},
	}, map[string]interface{}{})
	d.SetId("not-a-triple")

	_, err := effectiveEnvKey(d)
	require.Error(t, err)
}

// Happy path: id parses cleanly into project/env/key, env_key attribute is empty.
func TestEffectiveEnvKey_recoversFromID(t *testing.T) {
	t.Parallel()

	d := schema.TestResourceDataRaw(t, map[string]*schema.Schema{
		ENV_KEY: {Type: schema.TypeString, Optional: true},
	}, map[string]interface{}{})
	d.SetId("crossplane-project/name-dev/my-flag")

	got, err := effectiveEnvKey(d)
	require.NoError(t, err)
	require.Equal(t, "name-dev", got)
}

// effectiveCustomRoleKeyOrError error path: nothing to fall back to.
func TestEffectiveCustomRoleKeyOrError_errorWhenEmpty(t *testing.T) {
	t.Parallel()

	d := schema.TestResourceDataRaw(t, map[string]*schema.Schema{
		KEY: {Type: schema.TypeString, Optional: true},
	}, map[string]interface{}{})

	_, err := effectiveCustomRoleKeyOrError(d)
	require.Error(t, err)
}
