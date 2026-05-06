package launchdarkly

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func emptyInterfaceSlice() []interface{} { return []interface{}{} }

func getOptionalSet(d *schema.ResourceData, key string) *schema.Set {
	return optionalSchemaSetFromInterface(d.Get(key))
}

func optionalSetList(d *schema.ResourceData, key string) []interface{} {
	s := getOptionalSet(d, key)
	if s == nil {
		return emptyInterfaceSlice()
	}
	return s.List()
}

func getOptionalInterfaceSlice(d *schema.ResourceData, key string) []interface{} {
	return interfaceSliceFromAny(d.Get(key))
}

// optionalSchemaSetFromInterface preserves nil for absent/wrong-type values so callers can
// distinguish "not set" from "set to empty" — list-shaped helpers normalize to empty.
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

// interfaceSliceFromAny logs at WARN on type mismatch so embedded-schema regressions surface
// instead of silently coercing to empty.
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

// stringListFromOptionalSetValue returns nil (not empty) for nil sets — LD API treats nil and
// empty []string identically and a few callers depend on the nil signal.
func stringListFromOptionalSetValue(v interface{}) []string {
	s := optionalSchemaSetFromInterface(v)
	if s == nil {
		return nil
	}
	return stringsFromSchemaSet(s)
}

func optionalSetListFromAny(v interface{}) []interface{} {
	s := optionalSchemaSetFromInterface(v)
	if s == nil {
		return emptyInterfaceSlice()
	}
	return s.List()
}

// optionalIntFromResourceData uses defaultVal when the schema is missing the key (Upjet-embedded
// provider returns nil from d.Get). Wrong-type values log at WARN so regressions stay visible.
func optionalIntFromResourceData(d *schema.ResourceData, key string, defaultVal int) int {
	v := d.Get(key)
	if v == nil {
		return defaultVal
	}
	i, ok := v.(int)
	if !ok {
		log.Printf("[WARN] optionalIntFromResourceData: %q is not an int (got %T); using default %v", key, v, defaultVal)
		return defaultVal
	}
	return i
}

// optionalBoolFromResourceData uses defaultVal for missing keys (Upjet-embedded provider returns
// nil from d.Get). Wrong-type values log at WARN and fall back to defaultVal — without this guard
// callers would observe a schema regression as a real "false".
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

// trimmedStringAttr returns "" for missing keys; logs at WARN on wrong type so schema regressions
// are visible.
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

// effectiveEnvKey falls back to the env_key embedded in resource id "project_key/env_key/<key>"
// when the attribute is missing — required for the Crossplane / Upjet external-name flow where
// the schema may strip env_key from the resource view.
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

// effectiveEnvKeyFromIDOrAttr is the no-error wrapper for callsites (e.g. patch-path builders)
// that cannot return one. Prefer effectiveEnvKey in new code.
func effectiveEnvKeyFromIDOrAttr(d *schema.ResourceData) string {
	k, err := effectiveEnvKey(d)
	if err != nil {
		log.Printf("[WARN] effectiveEnvKeyFromIDOrAttr: %s", err)
		return ""
	}
	return k
}

// effectiveCustomRoleKeyOrError falls back to d.Id() (Crossplane external-name) when KEY is unset
// — under embedded schemas the LD custom role key is set via the Terraform id, not the attribute.
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

// effectiveCustomRoleKey is the no-error wrapper. Callers check for empty and emit a diag.
func effectiveCustomRoleKey(d *schema.ResourceData) string {
	k, err := effectiveCustomRoleKeyOrError(d)
	if err != nil {
		log.Printf("[WARN] effectiveCustomRoleKey: %s", err)
		return ""
	}
	return k
}
