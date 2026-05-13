package launchdarkly

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

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

// firstMemberIDForTest fetches the first account member via raw HTTP.
// api-client-go's strict UnmarshalJSON guards reject responses that
// include any member whose nested integrationMetadata lacks the
// required externalId field, so we can't go through AccountMembersApi.
// Result is memoised — the account's member set is effectively static
// for a single `go test` invocation.
var (
	firstMemberIDOnce sync.Once
	firstMemberIDInst string
	firstMemberIDErr  error
)

func firstMemberIDForTest(t *testing.T) string {
	t.Helper()
	firstMemberIDOnce.Do(func() {
		host := os.Getenv(LAUNCHDARKLY_API_HOST)
		if host == "" {
			host = DEFAULT_LAUNCHDARKLY_HOST
		}
		if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
			host = "https://" + host
		}
		token := os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN)
		if token == "" {
			firstMemberIDErr = fmt.Errorf("%s env var must be set for acceptance tests", LAUNCHDARKLY_ACCESS_TOKEN)
			return
		}
		req, err := http.NewRequest(http.MethodGet, host+"/api/v2/members?limit=1", nil)
		if err != nil {
			firstMemberIDErr = fmt.Errorf("build request: %s", err)
			return
		}
		req.Header.Set("Authorization", token)
		req.Header.Set("LD-API-Version", APIVersion)
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			firstMemberIDErr = fmt.Errorf("GET /members: %s", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			firstMemberIDErr = fmt.Errorf("unexpected status %d", resp.StatusCode)
			return
		}
		var out struct {
			Items []struct {
				ID string `json:"_id"`
			} `json:"items"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			firstMemberIDErr = fmt.Errorf("decode: %s", err)
			return
		}
		if len(out.Items) == 0 {
			firstMemberIDErr = fmt.Errorf("no members in account")
			return
		}
		firstMemberIDInst = out.Items[0].ID
	})
	if firstMemberIDErr != nil {
		t.Fatalf("firstMemberIDForTest: %s", firstMemberIDErr)
	}
	return firstMemberIDInst
}
