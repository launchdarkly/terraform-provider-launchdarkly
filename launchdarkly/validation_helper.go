package launchdarkly

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// Can't use validation.ToDiagFunc converted validators on TypeList at the moment
// https://github.com/hashicorp/terraform-plugin-sdk/issues/734
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

// Can't use validation.ToDiagFunc converted validators on TypeList at the moment
// https://github.com/hashicorp/terraform-plugin-sdk/issues/734
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

func validateID() schema.SchemaValidateDiagFunc {
	return validation.ToDiagFunc(validation.All(
		validation.StringMatch(regexp.MustCompile(`^[a-fA-F0-9]*$`), "Must be a 24 character hexadecimal string"),
		validation.StringLenBetween(24, 24),
	))
}

// Can't use validation.ToDiagFunc converted validators on TypeList at the moment
// https://github.com/hashicorp/terraform-plugin-sdk/issues/734
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
