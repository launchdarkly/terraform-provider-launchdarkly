package launchdarkly

// Regression coverage for https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/545.
//
// Accounts without the Views entitlement reject JSON-patch ops on
// /requireViewAssociationForNewFlags and /requireViewAssociationForNewSegments
// with a 400, and omit both fields from project GET responses. The test
// account used in CI has Views, so this file emulates such an account with a
// local reverse proxy in front of the real API and points an in-process
// provider at it. Everything other than the two gated paths hits real
// LaunchDarkly, so project/environment CRUD stays realistic.
//
// The provider's Configure strips any scheme from api_host and forces https,
// so the proxy cannot be injected via LAUNCHDARKLY_API_HOST; instead the
// provider is wrapped and Configure is overridden to build the client against
// the proxy URL directly.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const viewAssociationPathPrefix = "/requireViewAssociationForNew"

var viewAssociationFieldRe = regexp.MustCompile(`,?"requireViewAssociationForNew(Flags|Segments)":(true|false)`)

// newNoViewsProxy starts a reverse proxy to the real LaunchDarkly API that
// behaves like an account without the Views entitlement.
func newNoViewsProxy(t *testing.T) *httptest.Server {
	t.Helper()

	upstreamHost := os.Getenv(LAUNCHDARKLY_API_HOST)
	if upstreamHost == "" {
		upstreamHost = DEFAULT_LAUNCHDARKLY_HOST
	}
	if !strings.HasPrefix(upstreamHost, "http") {
		upstreamHost = "https://" + upstreamHost
	}
	upstream, err := url.Parse(upstreamHost)
	if err != nil {
		t.Fatalf("invalid upstream host %q: %v", upstreamHost, err)
	}

	rp := httputil.NewSingleHostReverseProxy(upstream)
	director := rp.Director
	rp.Director = func(r *http.Request) {
		director(r)
		r.Host = upstream.Host
		// Plain bodies so ModifyResponse can rewrite JSON.
		r.Header.Set("Accept-Encoding", "identity")
	}
	rp.ModifyResponse = func(resp *http.Response) error {
		if !strings.HasPrefix(resp.Request.URL.Path, "/api/v2/projects") ||
			!strings.Contains(resp.Header.Get("Content-Type"), "json") {
			return nil
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		_ = resp.Body.Close()
		stripped := viewAssociationFieldRe.ReplaceAll(body, nil)
		resp.Body = io.NopCloser(bytes.NewReader(stripped))
		resp.ContentLength = int64(len(stripped))
		resp.Header.Set("Content-Length", fmt.Sprint(len(stripped)))
		return nil
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/api/v2/projects/") {
			body, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(body))
			var ops []map[string]interface{}
			_ = json.Unmarshal(body, &ops)
			for _, op := range ops {
				if p, _ := op["path"].(string); strings.HasPrefix(p, viewAssociationPathPrefix) {
					// Exact shape returned by the API on accounts without Views (issue #545).
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintf(w, `{"code":"invalid_request","message":"Unsupported json-patch operation: path \"%s\" does not exist"}`, p)
					return
				}
			}
		}
		rp.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// proxiedProvider wraps the real provider and overrides Configure so the
// client talks to the given base URL (scheme included) instead of api_host.
type proxiedProvider struct {
	provider.Provider
	baseURL string
}

func (p *proxiedProvider) Configure(_ context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	client, err := newClient(os.Getenv(LAUNCHDARKLY_ACCESS_TOKEN), p.baseURL, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create LaunchDarkly client", err.Error())
		return
	}
	resp.ResourceData = client
	resp.DataSourceData = client
}

func proxiedProviderFactories(baseURL string) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"launchdarkly": providerserver.NewProtocol6WithError(&proxiedProvider{
			Provider: NewPluginProvider("test")(),
			baseURL:  baseURL,
		}),
	}
}

// failOnOrphanedProject asserts, after the test case has run its destroy, that
// the project is gone from the real account. A project that exists here was
// created but never written to state (the #545 failure mode). The orphan is
// deleted so a regression does not litter the test account.
func failOnOrphanedProject(t *testing.T, projectKey string) {
	t.Helper()
	client := mustTestAccClient()
	_, res, err := client.ld.ProjectsApi.GetProject(client.ctx, projectKey).Execute()
	if isStatusNotFound(res) {
		return
	}
	if err != nil {
		t.Errorf("checking for orphaned project %q: %v", projectKey, err)
		return
	}
	t.Errorf("project %q still exists after destroy: it was created but never written to state", projectKey)
	if _, err := client.ld.ProjectsApi.DeleteProject(client.ctx, projectKey).Execute(); err != nil {
		t.Logf("cleanup of orphaned project %q failed: %v", projectKey, err)
	}
}

func testAccProjectNoViewsConfig(projectKey, name, extraAttrs string) string {
	return fmt.Sprintf(`
resource "launchdarkly_project" "no_views" {
	key  = "%s"
	name = "%s"
	%s
	environments = {
	  "test-env" = {
	    name  = "Test Environment"
	    color = "010101"
	  }
	}
}
`, projectKey, name, extraAttrs)
}

// TestAccProject_NoViewsEntitlement_CreateWithoutAttrs: a project that does not
// mention the view association attributes must create, update and destroy on
// an account without Views. Before #545 the create-time default-false patch
// 400'd and left an orphaned project.
func TestAccProject_NoViewsEntitlement_CreateWithoutAttrs(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "launchdarkly_project.no_views"
	srv := newNoViewsProxy(t)
	t.Cleanup(func() { failOnOrphanedProject(t, projectKey) })

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: proxiedProviderFactories(srv.URL),
		CheckDestroy:             testAccCheckProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectNoViewsConfig(projectKey, "No Views Test", ""),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, KEY, projectKey),
					resource.TestCheckResourceAttr(resourceName, REQUIRE_VIEW_ASSOCIATION_FOR_NEW_FLAGS, "false"),
					resource.TestCheckResourceAttr(resourceName, REQUIRE_VIEW_ASSOCIATION_FOR_NEW_SEGMENTS, "false"),
				),
			},
			{
				// Update path must not re-send the unchanged default-false settings.
				Config: testAccProjectNoViewsConfig(projectKey, "No Views Test Renamed", ""),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, NAME, "No Views Test Renamed"),
				),
			},
		},
	})
}

// TestAccProject_NoViewsEntitlement_ExplicitTrueLandsInState: asking for view
// association on an account without Views must fail the apply, but the
// project must still be written to state so Terraform manages (and here,
// destroys) it rather than leaving an orphan.
func TestAccProject_NoViewsEntitlement_ExplicitTrueLandsInState(t *testing.T) {
	projectKey := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	srv := newNoViewsProxy(t)
	t.Cleanup(func() { failOnOrphanedProject(t, projectKey) })

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProtoV6ProviderFactories: proxiedProviderFactories(srv.URL),
		CheckDestroy:             testAccCheckProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccProjectNoViewsConfig(projectKey, "No Views Explicit True", "require_view_association_for_new_flags = true"),
				ExpectError: regexp.MustCompile(`failed to update view association settings for project.*requireViewAssociationForNewFlags.*does not exist`),
			},
		},
	})
}
