package launchdarkly

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"
)

func TestOptionalSchemaSetFromInterface(t *testing.T) {
	t.Parallel()

	require.Nil(t, optionalSchemaSetFromInterface(nil))
	require.Nil(t, optionalSchemaSetFromInterface("not-a-set"))
	require.Nil(t, optionalSchemaSetFromInterface(42))

	valid := schema.NewSet(schema.HashString, []interface{}{"a", "b"})
	got := optionalSchemaSetFromInterface(valid)
	require.NotNil(t, got)
	require.Equal(t, 2, got.Len())
}

func TestInterfaceSliceFromAny(t *testing.T) {
	t.Parallel()

	require.NotNil(t, interfaceSliceFromAny(nil), "nil should normalize to empty slice, not nil")
	require.Empty(t, interfaceSliceFromAny(nil))
	require.Empty(t, interfaceSliceFromAny(42))
	require.Empty(t, interfaceSliceFromAny("slice"))

	sl := []interface{}{"a", "b"}
	require.Equal(t, sl, interfaceSliceFromAny(sl))
}

// Simulates embedded provider behavior: optional blocks unset → nil from d.Get (issue #387).
func TestOptionalSetListAndGetOptionalInterfaceSlice_unsetOptionalBlocks(t *testing.T) {
	t.Parallel()

	d := schema.TestResourceDataRaw(t, map[string]*schema.Schema{
		"custom_roles": {
			Type:     schema.TypeSet,
			Optional: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
			Set:      schema.HashString,
		},
		"policy_statements": {
			Type:     schema.TypeList,
			Optional: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"effect": {Type: schema.TypeString, Required: true},
				},
			},
		},
	}, map[string]interface{}{})

	require.Empty(t, optionalSetList(d, "custom_roles"))
	require.Empty(t, getOptionalInterfaceSlice(d, "policy_statements"))
}

func TestOptionalBoolFromResourceData(t *testing.T) {
	t.Parallel()
	withTrue := schema.TestResourceDataRaw(t, map[string]*schema.Schema{
		"x": {Type: schema.TypeBool, Optional: true},
	}, map[string]interface{}{"x": true})
	require.True(t, optionalBoolFromResourceData(withTrue, "x", false))

	unset := schema.TestResourceDataRaw(t, map[string]*schema.Schema{
		"x": {Type: schema.TypeBool, Optional: true},
	}, map[string]interface{}{})
	require.False(t, optionalBoolFromResourceData(unset, "x", true))

	wrongType := schema.TestResourceDataRaw(t, map[string]*schema.Schema{
		"x": {Type: schema.TypeString, Optional: true},
	}, map[string]interface{}{"x": "yes"})
	// Non-bool value: use default
	require.True(t, optionalBoolFromResourceData(wrongType, "x", true))
}

func TestOptionalIntFromResourceData(t *testing.T) {
	t.Parallel()
	withVal := schema.TestResourceDataRaw(t, map[string]*schema.Schema{
		"x": {Type: schema.TypeInt, Optional: true},
	}, map[string]interface{}{"x": 42})
	require.Equal(t, 42, optionalIntFromResourceData(withVal, "x", -1))

	// schema present, value unset → returns int zero value (the default arg only applies when d.Get returns nil)
	unset := schema.TestResourceDataRaw(t, map[string]*schema.Schema{
		"x": {Type: schema.TypeInt, Optional: true},
	}, map[string]interface{}{})
	require.Equal(t, 0, optionalIntFromResourceData(unset, "x", 7))

	// wrong type at the schema layer → default applies via the type-assertion guard
	wrongType := schema.TestResourceDataRaw(t, map[string]*schema.Schema{
		"x": {Type: schema.TypeString, Optional: true},
	}, map[string]interface{}{"x": "not-an-int"})
	require.Equal(t, 99, optionalIntFromResourceData(wrongType, "x", 99))
}

func TestPoliciesFromResourceData_nilPolicyNoPanic(t *testing.T) {
	t.Parallel()

	d := schema.TestResourceDataRaw(t, resourceCustomRole().Schema, map[string]interface{}{
		KEY:               "k",
		NAME:              "n",
		BASE_PERMISSIONS:  "reader",
		POLICY_STATEMENTS: []interface{}{},
	})
	// Omit POLICY — Terraform CLI often yields an empty set; helpers must tolerate nil like embedded SDK.
	require.NotPanics(t, func() {
		_ = policiesFromResourceData(d)
	})
}

func TestEffectiveEnvKeyFromIDOrAttr(t *testing.T) {
	t.Parallel()

	withAttr := schema.TestResourceDataRaw(t, map[string]*schema.Schema{
		ENV_KEY: {Type: schema.TypeString, Required: true},
	}, map[string]interface{}{ENV_KEY: "  name-dev  "})
	require.Equal(t, "name-dev", effectiveEnvKeyFromIDOrAttr(withAttr))

	fromID := schema.TestResourceDataRaw(t, map[string]*schema.Schema{
		ENV_KEY: {Type: schema.TypeString, Optional: true},
	}, map[string]interface{}{})
	fromID.SetId("crossplane-project/name-dev/my-flag")
	require.Equal(t, "name-dev", effectiveEnvKeyFromIDOrAttr(fromID))

	attrWins := schema.TestResourceDataRaw(t, map[string]*schema.Schema{
		ENV_KEY: {Type: schema.TypeString, Optional: true},
	}, map[string]interface{}{ENV_KEY: "production"})
	attrWins.SetId("proj/other-env/flag")
	require.Equal(t, "production", effectiveEnvKeyFromIDOrAttr(attrWins))
}

func TestEffectiveCustomRoleKey(t *testing.T) {
	t.Parallel()

	withKey := schema.TestResourceDataRaw(t, resourceCustomRole().Schema, map[string]interface{}{
		KEY:              "my-role",
		NAME:             "n",
		BASE_PERMISSIONS: "reader",
	})
	require.Equal(t, "my-role", effectiveCustomRoleKey(withKey))

	fromID := schema.TestResourceDataRaw(t, map[string]*schema.Schema{
		KEY: {Type: schema.TypeString, Optional: true},
	}, map[string]interface{}{})
	fromID.SetId("  allow-product-manager-tag  ")
	require.Equal(t, "allow-product-manager-tag", effectiveCustomRoleKey(fromID))
}
