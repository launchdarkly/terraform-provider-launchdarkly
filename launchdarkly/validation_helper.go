package launchdarkly

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

//nolint:staticcheck // SA1019 TODO: return SchemaValidateDiagFunc type
func validateKey() schema.SchemaValidateFunc {
	return validation.StringMatch(
		regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`),
		"Must contain only letters, numbers, '.', '-', or '_' and must start with an alphanumeric",
	)
}

//nolint:staticcheck // SA1019 TODO: return SchemaValidateDiagFunc type
func validateKeyAndLength(minLength, maxLength int) schema.SchemaValidateFunc {
	return validation.All(
		validation.StringMatch(
			regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`),
			"Must contain only letters, numbers, '.', '-', or '_' and must start with an alphanumeric",
		),
		validation.StringLenBetween(minLength, maxLength),
	)
}

//nolint:staticcheck // SA1019 TODO: return SchemaValidateDiagFunc type
func validateID() schema.SchemaValidateFunc {
	return validation.All(
		validation.StringMatch(regexp.MustCompile(`^[a-fA-F0-9]*$`), "Must be a 24 character hexadecimal string"),
		validation.StringLenBetween(24, 24),
	)
}

//nolint:staticcheck // SA1019 TODO: return SchemaValidateDiagFunc type
func validateTags() schema.SchemaValidateFunc {
	return validation.All(
		validation.StringLenBetween(1, 64),
		validation.StringMatch(
			regexp.MustCompile(`^[a-zA-Z0-9_.-]*$`),
			"Must contain only letters, numbers, '.', '-', or '_' and be at most 64 characters",
		),
	)
}

//nolint:staticcheck // SA1019 TODO: return SchemaValidateDiagFunc type
func validateOp() schema.SchemaValidateFunc {
	return validation.StringInSlice([]string{
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
	}, false)
}
