package tests

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-mux/tf5muxserver"
	"github.com/launchdarkly/terraform-provider-launchdarkly/launchdarkly"
)

const LAUNCHDARKLY_ACCESS_TOKEN = "LAUNCHDARKLY_ACCESS_TOKEN"

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN); v == "" {
		t.Fatalf("%s env var must be set for acceptance tests", LAUNCHDARKLY_ACCESS_TOKEN)
	}
}

// testAccFrameworkMuxProviders builds the same tf5muxserver as main.go.
//
// The root launchdarkly package owns the canonical wiring in its
// _test.go file (TestAccProtoV5ProviderFactories), but Go test symbols are
// package-scoped, so this sub-package rebuilds the mux locally rather than
// importing across the test boundary. Keep both factories in sync.
func testAccFrameworkMuxProviders(ctx context.Context, _ *testing.T) map[string]func() (tfprotov5.ProviderServer, error) {
	sdkV2Provider := launchdarkly.Provider()
	frameworkProvider := launchdarkly.NewPluginProvider("test")

	return map[string]func() (tfprotov5.ProviderServer, error){
		"launchdarkly": func() (tfprotov5.ProviderServer, error) {
			return tf5muxserver.NewMuxServer(ctx,
				sdkV2Provider.GRPCProvider,
				providerserver.NewProtocol5(frameworkProvider()),
			)
		},
	}
}
