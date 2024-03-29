package launchdarkly

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// This map is most commonly constructed once in a common init() method of the Provider’s main test file,
// and includes an object of the current Provider type. https://www.terraform.io/docs/extend/testing/acceptance-tests/testcase.html
var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"launchdarkly": testAccProvider,
	}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN); v == "" {
		t.Fatalf("%s env var must be set for acceptance tests", LAUNCHDARKLY_ACCESS_TOKEN)
	}
}
