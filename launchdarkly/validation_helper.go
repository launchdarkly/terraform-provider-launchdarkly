package launchdarkly

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func validateKey() schema.SchemaValidateFunc {
	return validation.StringMatch(
		regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`),
		"Must contain only letters, numbers, '.', '-', or '_' and must start with an alphanumeric",
	)
}

func validateID() schema.SchemaValidateFunc {
	return validation.All(
		validation.StringMatch(regexp.MustCompile(`^[a-fA-F0-9]*$`), "Must be a 24 character hexadecimal string"),
		validation.StringLenBetween(24, 24),
	)
}

func validateTags() schema.SchemaValidateFunc {
	return validation.All(
		validation.StringLenBetween(1, 64),
		validation.StringMatch(
			regexp.MustCompile(`^[a-zA-Z0-9_.-]*$`),
			"Must contain only letters, numbers, '.', '-', or '_' and be at most 64 characters",
		),
	)
}

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
