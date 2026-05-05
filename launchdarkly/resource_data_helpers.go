package launchdarkly

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// emptyInterfaceSlice is returned in place of nil for list-shaped helpers so callers can iterate
// without nil checks and JSON marshalling stays deterministic.
func emptyInterfaceSlice() []interface{} { return []interface{}{} }

func getOptionalSet(d *schema.ResourceData, key string) *schema.Set {
	return optionalSchemaSetFromInterface(d.Get(key))
}

// optionalSetList returns the contents of a TypeSet attribute as []interface{}.
// Returns an empty (non-nil) slice when the key is missing/unset/wrong type.
func optionalSetList(d *schema.ResourceData, key string) []interface{} {
	s := getOptionalSet(d, key)
	if s == nil {
		return emptyInterfaceSlice()
	}
	return s.List()
}

// getOptionalInterfaceSlice returns a TypeList attribute as []interface{}.
// Returns an empty (non-nil) slice when the key is missing/unset/wrong type.
func getOptionalInterfaceSlice(d *schema.ResourceData, key string) []interface{} {
	return interfaceSliceFromAny(d.Get(key))
}

// optionalSchemaSetFromInterface returns nil when v is nil or not a *schema.Set; otherwise the set.
// Sets retain nil semantics so callers that pass them straight back into helpers (e.g. .List()) can
// branch on a nil set without an extra zero-length allocation.
func optionalSchemaSetFromInterface(v interface{}) *schema.Set {
	if v == nil {
		return nil
	}
	s, ok := v.(*schema.Set)
	if !ok || s == nil {
		return nil
	}
	return s
}

// interfaceSliceFromAny normalizes any-shaped list values to []interface{}.
// Nil or wrong-type input yields an empty (non-nil) slice; the wrong-type case is logged so a
// schema regression is observable instead of silently coerced.
func interfaceSliceFromAny(v interface{}) []interface{} {
	if v == nil {
		return emptyInterfaceSlice()
	}
	s, ok := v.([]interface{})
	if !ok {
		log.Printf("[WARN] interfaceSliceFromAny: expected []interface{}, got %T", v)
		return emptyInterfaceSlice()
	}
	return s
}

// stringListFromOptionalSetValue converts a *schema.Set wrapped in interface{} (e.g. diff.Get / GetChange)
// to a []string for LaunchDarkly API calls. Nil or wrong type yields nil (preserves prior behavior;
// LD API calls treat nil and empty []string identically and a few callers depend on the nil signal).
func stringListFromOptionalSetValue(v interface{}) []string {
	s := optionalSchemaSetFromInterface(v)
	if s == nil {
		return nil
	}
	return interfaceSliceToStringSlice(s.List())
}

// optionalSetListFromAny returns the contents of a *schema.Set value as []interface{}.
// Returns an empty (non-nil) slice when the input is nil or wrong type.
func optionalSetListFromAny(v interface{}) []interface{} {
	s := optionalSchemaSetFromInterface(v)
	if s == nil {
		return emptyInterfaceSlice()
	}
	return s.List()
}

// optionalBoolFromResourceData returns d.Get(key) as bool when it is a non-nil bool value.
// When the key is missing from the schema or Get returns nil (e.g. Upjet-embedded provider),
// defaultVal is used. A wrong-type value (schema regression) is logged at WARN and treated as
// missing so callers do not accidentally observe schema misconfiguration as a real "false".
func optionalBoolFromResourceData(d *schema.ResourceData, key string, defaultVal bool) bool {
	v := d.Get(key)
	if v == nil {
		return defaultVal
	}
	b, ok := v.(bool)
	if !ok {
		log.Printf("[WARN] optionalBoolFromResourceData: %q is not a bool (got %T); using default %v", key, v, defaultVal)
		return defaultVal
	}
	return b
}

// trimmedStringAttr returns strings.TrimSpace(d.Get(key)) for string attributes.
// Returns "" for missing keys; logs at WARN for wrong types so schema regressions are visible.
func trimmedStringAttr(d *schema.ResourceData, key string) string {
	v := d.Get(key)
	if v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		log.Printf("[WARN] trimmedStringAttr: %q is not a string (got %T)", key, v)
		return ""
	}
	return strings.TrimSpace(s)
}

// effectiveEnvKey returns ENV_KEY from config when set; otherwise parses env_key from the resource
// id "project_key/env_key/flag_key" (or "/segment_key"). Returns an explicit error when env_key is
// neither in the attributes nor recoverable from a well-formed id; callers should propagate via
// diag.FromErr so an unset env_key is a real failure rather than an ambient empty string.
func effectiveEnvKey(d *schema.ResourceData) (string, error) {
	if k := trimmedStringAttr(d, ENV_KEY); k != "" {
		return k, nil
	}
	id := strings.TrimSpace(d.Id())
	if id == "" {
		return "", fmt.Errorf("%s is required and resource id is empty", ENV_KEY)
	}
	if strings.Count(id, "/") != 2 {
		return "", fmt.Errorf("%s is empty and resource id %q is not in the form project_key/env_key/<key>", ENV_KEY, id)
	}
	parts := strings.SplitN(id, "/", 3)
	envKey := strings.TrimSpace(parts[1])
	if envKey == "" {
		return "", fmt.Errorf("%s is empty and parsed env_key from id %q is empty", ENV_KEY, id)
	}
	return envKey, nil
}

// effectiveEnvKeyFromIDOrAttr is the legacy ergonomic wrapper around effectiveEnvKey for callsites
// (notably patch path builders) that cannot return an error. Logs at WARN when the lookup fails so
// the underlying issue still surfaces; prefer effectiveEnvKey in new code.
func effectiveEnvKeyFromIDOrAttr(d *schema.ResourceData) string {
	k, err := effectiveEnvKey(d)
	if err != nil {
		log.Printf("[WARN] effectiveEnvKeyFromIDOrAttr: %s", err)
		return ""
	}
	return k
}

// effectiveCustomRoleKeyOrError returns KEY from config when set; otherwise the Terraform resource
// id (Crossplane external-name / observe id), which must be the LaunchDarkly custom role key.
// Returns an error when both are empty.
func effectiveCustomRoleKeyOrError(d *schema.ResourceData) (string, error) {
	if k := trimmedStringAttr(d, KEY); k != "" {
		return k, nil
	}
	id := strings.TrimSpace(d.Id())
	if id == "" {
		return "", fmt.Errorf("%s is required and resource id is empty; set %s or the Terraform resource id to the LaunchDarkly custom role key", KEY, KEY)
	}
	return id, nil
}

// effectiveCustomRoleKey is the legacy ergonomic wrapper around effectiveCustomRoleKeyOrError.
// Returns "" on failure; callers in this provider check for empty and emit a diag.
func effectiveCustomRoleKey(d *schema.ResourceData) string {
	k, err := effectiveCustomRoleKeyOrError(d)
	if err != nil {
		log.Printf("[WARN] effectiveCustomRoleKey: %s", err)
		return ""
	}
	return k
}
