package launchdarkly

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// Can't use validation.ToDiagFunc converted validators on TypeList at the moment
// https://github.com/hashicorp/terraform-plugin-sdk/issues/734
//
//nolint:staticcheck // SA1019 TODO: return SchemaValidateDiagFunc type
func validateKeyNoDiag() schema.SchemaValidateFunc {
	return validation.StringMatch(
		regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`),
		"Must contain only letters, numbers, '.', '-', or '_' and must start with an alphanumeric",
	)
}

func validateKey() schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(validation.StringMatch(
		regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`),
		"Must contain only letters, numbers, '.', '-', or '_' and must start with an alphanumeric",
	))
}

// validateViewKey validates a view key. LaunchDarkly normalizes view keys to
// lowercase server-side, so an uppercase key in configuration can never
// round-trip: the API stores the lowercased form, Read pulls that back, and the
// resulting permanent diff forces a replace on every plan. Rejecting it up
// front turns that into a plan-time error that names the fix.
//
// Applies to every attribute a user can type a view key into, not just
// launchdarkly_view.key — a reference site (view_links.view_key,
// feature_flag.view_keys, ...) that names an uppercase key would otherwise fail
// later as an opaque 404.
func validateViewKey() schema.SchemaValidateDiagFunc {
	return func(i interface{}, path cty.Path) diag.Diagnostics {
		if diags := validateKey()(i, path); diags.HasError() {
			return diags
		}
		return validateLowercase("LaunchDarkly normalizes view keys to lowercase")(i, path)
	}
}

// validateLowercase rejects any string containing uppercase characters. Kept
// separate from the key regex so the diagnostic can name the real problem and
// suggest the normalized value, rather than emitting the generic "letters,
// numbers, '.', '-', or '_'" message for a value that already satisfies those
// rules.
//
// reason is folded into the diagnostic to explain why this particular attribute
// must be lowercase; pass "" to omit it.
func validateLowercase(reason string) schema.SchemaValidateDiagFunc {
	return func(i interface{}, path cty.Path) diag.Diagnostics {
		got, ok := i.(string)
		if !ok {
			return diag.Diagnostics{{
				Severity:      diag.Error,
				Summary:       "Invalid value",
				Detail:        fmt.Sprintf("expected type of value to be string, got %T", i),
				AttributePath: path,
			}}
		}
		lowered := strings.ToLower(got)
		if got == lowered {
			return nil
		}
		detail := fmt.Sprintf("expected value to be lowercase, got %q", got)
		if reason != "" {
			detail += ". " + reason
		}
		detail += fmt.Sprintf(". Use %q instead", lowered)
		return diag.Diagnostics{{
			Severity:      diag.Error,
			Summary:       "Invalid value",
			Detail:        detail,
			AttributePath: path,
		}}
	}
}

// Can't use validation.ToDiagFunc converted validators on TypeList at the moment
// https://github.com/hashicorp/terraform-plugin-sdk/issues/734
//
//nolint:staticcheck // SA1019 TODO: return SchemaValidateDiagFunc type
func validateKeyAndLength(minLength, maxLength int) schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(validation.All(
		validation.StringMatch(
			regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`),
			"Must contain only letters, numbers, '.', '-', or '_' and must start with an alphanumeric",
		),
		validation.StringLenBetween(minLength, maxLength),
	))
}

func validateID() schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(validation.All(
		validation.StringMatch(regexp.MustCompile(`^[a-fA-F0-9]*$`), "Must be a 24 character hexadecimal string"),
		validation.StringLenBetween(24, 24),
	))
}

// Can't use validation.ToDiagFunc converted validators on TypeList at the moment
// https://github.com/hashicorp/terraform-plugin-sdk/issues/734
//
//nolint:staticcheck // SA1019 TODO: return SchemaValidateDiagFunc type
func validateTagsNoDiag() schema.SchemaValidateFunc {
	return validation.All(
		validation.StringLenBetween(1, 64),
		validation.StringMatch(
			regexp.MustCompile(`^[a-zA-Z0-9_.-]*$`),
			"Must contain only letters, numbers, '.', '-', or '_' and be at most 64 characters",
		),
	)
}

// func validateTags() schema.SchemaValidateDiagFunc {
// 	return validation.ToDiagFunc(validation.All(
// 		validation.StringLenBetween(1, 64),
// 		validation.StringMatch(
// 			regexp.MustCompile(`^[a-zA-Z0-9_.-]*$`),
// 			"Must contain only letters, numbers, '.', '-', or '_' and be at most 64 characters",
// 		),
// 	))
// }

func validateOp() schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(validation.StringInSlice([]string{
		"in",
		"endsWith",
		"startsWith",
		"matches",
		"contains",
		"lessThan",
		"greaterThan",
		"lessThanOrEqual",
		"greaterThanOrEqual",
		"before",
		"after",
		"segmentMatch",
		"semVerEqual",
		"semVerLessThan",
		"semVerGreaterThan",
	}, false))
}
