package launchdarkly

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-mux/tf5muxserver"
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

// testAccProtoV5ProviderFactories serves the same muxed provider main.go serves:
// the SDKv2 provider plus the plugin-framework provider. Tests for
// framework-only resources (launchdarkly_team_role_mapping is the only one on
// this branch) must use this rather than testAccProviders, which serves the
// SDKv2 half alone and therefore does not know those resource types exist.
var testAccProtoV5ProviderFactories = map[string]func() (tfprotov5.ProviderServer, error){
	"launchdarkly": func() (tfprotov5.ProviderServer, error) {
		muxServer, err := tf5muxserver.NewMuxServer(
			context.Background(),
			Provider().GRPCProvider,
			providerserver.NewProtocol5(NewPluginProvider(version)()),
		)
		if err != nil {
			return nil, err
		}
		return muxServer.ProviderServer(), nil
	},
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN); v == "" {
		t.Fatalf("%s env var must be set for acceptance tests", LAUNCHDARKLY_ACCESS_TOKEN)
	}
}
