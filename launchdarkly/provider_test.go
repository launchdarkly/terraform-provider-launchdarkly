package launchdarkly

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-mux/tf5muxserver"
)

// testAccProtoV5ProviderFactories serves the same tf5muxserver as main.go
// so acceptance tests exercise the production protocol surface. Used in
// resource.TestCase via:
//
//	ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
//
// The launchdarkly/tests sub-package maintains its own equivalent factory
// because Go test symbols are package-scoped — keep the two wirings in sync.
var testAccProtoV5ProviderFactories = map[string]func() (tfprotov5.ProviderServer, error){
	"launchdarkly": func() (tfprotov5.ProviderServer, error) {
		ctx := context.Background()
		return tf5muxserver.NewMuxServer(ctx,
			Provider().GRPCProvider,
			providerserver.NewProtocol5(NewPluginProvider("test")()),
		)
	},
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN); v == "" {
		t.Fatalf("%s env var must be set for acceptance tests", LAUNCHDARKLY_ACCESS_TOKEN)
	}
}

// mustTestAccClient builds a *Client from environment variables, mirroring
// the provider Configure code path. Acceptance test CheckDestroy / CheckExists
// helpers historically reached into testAccProvider.Meta() to grab the SDKv2
// provider's configured client. Under the proto-factory pattern the mux server
// is opaque, so helpers construct their own client. The result is memoised
// because every CheckFunc in the suite resolves to the same env-derived client.
var (
	testAccClientOnce sync.Once
	testAccClientInst *Client
)

func mustTestAccClient() *Client {
	testAccClientOnce.Do(func() {
		host := os.Getenv(LAUNCHDARKLY_API_HOST)
		if host == "" {
			host = DEFAULT_LAUNCHDARKLY_HOST
		}
		token := os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN)
		if token == "" {
			panic(fmt.Sprintf("%s must be set for acceptance tests", LAUNCHDARKLY_ACCESS_TOKEN))
		}
		client, err := newClient(token, host, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
		if err != nil {
			panic(fmt.Sprintf("failed to construct test client: %s", err))
		}
		testAccClientInst = client
	})
	return testAccClientInst
}
