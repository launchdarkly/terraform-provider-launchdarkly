package tests

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/launchdarkly/terraform-provider-launchdarkly/launchdarkly"
)

const LAUNCHDARKLY_ACCESS_TOKEN = "LAUNCHDARKLY_ACCESS_TOKEN"

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN); v == "" {
		t.Fatalf("%s env var must be set for acceptance tests", LAUNCHDARKLY_ACCESS_TOKEN)
	}
}

// testAccFrameworkMuxProviders serves the framework provider as v5,
// matching main.go's wire protocol. The root launchdarkly package owns
// the canonical wiring in provider_test.go, but Go test symbols are
// package-scoped, so this sub-package rebuilds the factory locally.
// (Slated for removal in 5.1a once this sub-package collapses back into
// the root pkg.)
func testAccFrameworkMuxProviders(_ context.Context, _ *testing.T) map[string]func() (tfprotov5.ProviderServer, error) {
	return map[string]func() (tfprotov5.ProviderServer, error){
		"launchdarkly": providerserver.NewProtocol5WithError(launchdarkly.NewPluginProvider("test")()),
	}
}
