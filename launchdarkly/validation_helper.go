package launchdarkly

import (
	"regexp"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func validateKey() schema.SchemaValidateFunc {
	return validation.All(
		validation.StringMatch(regexp.MustCompile(`^[a-zA-Z0-9_.-]*$`), "Must contain only letters, numbers, dashes, and underscores"),
		validation.StringLenBetween(1, 20))
}
