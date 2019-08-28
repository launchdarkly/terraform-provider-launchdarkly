package launchdarkly

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// This map is most commonly constructed once in a common init() method of the Provider’s main test file,
// and includes an object of the current Provider type. https://www.terraform.io/docs/extend/testing/acceptance-tests/testcase.html
var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"launchdarkly": testAccProvider,
	}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv(launchDarklyAPIKeyEnvVar); v == "" {
		t.Fatalf("%s env var must be set for acceptance tests", launchDarklyAPIKeyEnvVar)
	}
}

// Tags are a TypeSet. TF represents this a as a map of hashes to actual values.
// The hashes are always the same for a given value so this is repeatable.
func testAccTagKey(val string) string {
	return fmt.Sprintf("tags.%d", hashcode.String(val))
}
