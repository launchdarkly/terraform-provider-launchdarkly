package launchdarkly

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// isOmittedEmbeddedSchemaAttrErr reports whether err is the specific Terraform plugin SDK error
// emitted when a write target is not present in the runtime schema (e.g. Upjet has stripped a
// deprecated attribute). The matchers are intentionally narrow so unrelated errors that merely
// mention attr are never swallowed.
//
// Known SDK error shapes (terraform-plugin-sdk v2):
//   - schema.ResourceData.Set / MapFieldWriter.WriteField:
//     "Invalid address to set: []string{\"<attr>\"}"
//   - schema.ResourceDiff.SetNew via mapFieldReader / readField:
//     ": invalid key: <attr>"  (preceded by a colon and following a known SDK prefix)
func isOmittedEmbeddedSchemaAttrErr(err error, attr string) bool {
	if err == nil || attr == "" {
		return false
	}
	addressPattern := fmt.Sprintf(`Invalid address to set: []string{%q}`, attr)
	invalidKeyExact := fmt.Sprintf(": invalid key: %s", attr)
	for cur := err; cur != nil; cur = errors.Unwrap(cur) {
		s := cur.Error()
		if strings.Contains(s, addressPattern) {
			return true
		}
		if strings.Contains(s, invalidKeyExact) {
			return true
		}
	}
	return false
}

// resourceDiffSetNewSkipMissingKey runs diff.SetNew and treats a missing schema key as success.
// Use when embedders remove deprecated attributes from the runtime schema while provider code
// still references them for Terraform CLI compatibility. A suppressed error is logged at DEBUG
// so the underlying mismatch remains observable.
func resourceDiffSetNewSkipMissingKey(diff *schema.ResourceDiff, key string, value interface{}) error {
	err := diff.SetNew(key, value)
	if err == nil {
		return nil
	}
	if isOmittedEmbeddedSchemaAttrErr(err, key) {
		log.Printf("[DEBUG] schema_compat: suppressed SetNew(%q) on missing schema key: %v", key, err)
		return nil
	}
	return err
}

// resourceDataSetSkipMissingKey runs d.Set and treats a missing schema key as success.
// A suppressed error is logged at DEBUG.
func resourceDataSetSkipMissingKey(d *schema.ResourceData, key string, value interface{}) error {
	err := d.Set(key, value)
	if err == nil {
		return nil
	}
	if isOmittedEmbeddedSchemaAttrErr(err, key) {
		log.Printf("[DEBUG] schema_compat: suppressed Set(%q) on missing schema key: %v", key, err)
		return nil
	}
	return err
}
