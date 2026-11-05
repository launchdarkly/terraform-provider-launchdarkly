package launchdarkly

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"
)

// These tests drive the observability filter resources through real
// Terraform runs (resource.UnitTest, no TF_ACC needed) against obsFake, an
// httptest implementation of the beta REST contract.

// obsFakeProvider is the real provider with Configure swapped out so that
// resources talk to the fake over plain HTTP.
type obsFakeProvider struct {
	provider.Provider
	client *Client
}

func (p *obsFakeProvider) Configure(_ context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	resp.ResourceData = p.client
	resp.DataSourceData = p.client
}

func newObsFakeHarness(t *testing.T) (*obsFake, map[string]func() (tfprotov6.ProviderServer, error)) {
	t.Helper()
	fake := newObsFake()
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	client, err := newClient("api-fake-token", server.URL, false, DEFAULT_HTTP_TIMEOUT_S, 4)
	require.NoError(t, err)
	factories := map[string]func() (tfprotov6.ProviderServer, error){
		"launchdarkly": providerserver.NewProtocol6WithError(&obsFakeProvider{Provider: NewPluginProvider("test")(), client: client}),
	}
	return fake, factories
}

func obsRuleHCL(name, key, productType, extra string) string {
	return fmt.Sprintf(`
resource "launchdarkly_observability_filter_rule" %q {
  project_key  = "web"
  key          = %q
  product_type = %q
  name         = "Rule %s"
  query        = "service_name=api"
  operation    = "exclude"
%s
}
`, name, key, productType, key, extra)
}

func checkFake(fake *obsFake, project string, fn func(rules map[string]obsFakeRule, order map[string][]string, limits map[string]*int64) error) resource.TestCheckFunc {
	return func(*terraform.State) error {
		rules, order, limits := fake.snapshot(project)
		return fn(rules, order, limits)
	}
}

func expectOrder(fake *obsFake, project, product string, want ...string) resource.TestCheckFunc {
	return checkFake(fake, project, func(_ map[string]obsFakeRule, order map[string][]string, _ map[string]*int64) error {
		got := order[product]
		if strings.Join(got, ",") != strings.Join(want, ",") {
			return fmt.Errorf("server %s order = %v, want %v", product, got, want)
		}
		return nil
	})
}

func TestObservabilityFilterRule_lifecycle(t *testing.T) {
	fake, factories := newObsFakeHarness(t)
	addr := "launchdarkly_observability_filter_rule.drop_healthz"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		CheckDestroy: checkFake(fake, "web", func(rules map[string]obsFakeRule, order map[string][]string, _ map[string]*int64) error {
			if len(rules) != 0 || len(order["logs"]) != 0 {
				return fmt.Errorf("expected no rules after destroy, got %v / order %v", rules, order)
			}
			return nil
		}),
		Steps: []resource.TestStep{
			{
				Config: obsRuleHCL("drop_healthz", "drop-healthz", "logs", ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "id", "web/drop-healthz"),
					resource.TestCheckResourceAttr(addr, ENABLED, "true"),
					resource.TestCheckResourceAttr(addr, RATE, "1"),
					resource.TestCheckResourceAttr(addr, VERSION, "1"),
					resource.TestCheckNoResourceAttr(addr, ATTRIBUTES),
					expectOrder(fake, "web", "logs", "drop-healthz"),
				),
			},
			{
				Config: obsRuleHCL("drop_healthz", "drop-healthz", "logs", "  enabled = false\n  rate    = 0.25"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, ENABLED, "false"),
					resource.TestCheckResourceAttr(addr, RATE, "0.25"),
					resource.TestCheckResourceAttr(addr, VERSION, "2"),
					checkFake(fake, "web", func(rules map[string]obsFakeRule, _ map[string][]string, _ map[string]*int64) error {
						if r := rules["drop-healthz"]; r.Enabled || r.Rate != 0.25 {
							return fmt.Errorf("server rule not updated: %+v", r)
						}
						return nil
					}),
				),
			},
			{
				ResourceName:      addr,
				ImportState:       true,
				ImportStateId:     "web/drop-healthz",
				ImportStateVerify: true,
			},
		},
	})
}

func TestObservabilityFilterRule_metricsAttributes(t *testing.T) {
	_, factories := newObsFakeHarness(t)
	addr := "launchdarkly_observability_filter_rule.strip"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: obsRuleHCL("strip", "strip-pod", "metrics", `  attributes = ["k8s.pod.name", "k8s.pod.uid"]`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "attributes.#", "2"),
					resource.TestCheckTypeSetElemAttr(addr, "attributes.*", "k8s.pod.uid"),
				),
			},
			{
				Config: obsRuleHCL("strip", "strip-pod", "metrics", `  attributes = []`),
				Check:  resource.TestCheckResourceAttr(addr, "attributes.#", "0"),
			},
			{
				ResourceName:            addr,
				ImportState:             true,
				ImportStateId:           "web/strip-pod",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{ATTRIBUTES}, // [] vs omitted is indistinguishable on import
			},
		},
	})
}

func TestObservabilityFilterRule_removedOutsideTerraform(t *testing.T) {
	fake, factories := newObsFakeHarness(t)
	addr := "launchdarkly_observability_filter_rule.drop_healthz"
	config := obsRuleHCL("drop_healthz", "drop-healthz", "logs", "")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{Config: config},
			{
				// Deleted in the UI: Read gets a 404, drops the rule from
				// state, and Terraform plans to create it again.
				PreConfig: func() { fake.uiDeleteRule("web", "drop-healthz") },
				Config:    config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(addr, plancheck.ResourceActionCreate)},
				},
				Check: expectOrder(fake, "web", "logs", "drop-healthz"),
			},
		},
	})
}

func TestObservabilityFilterRule_keyTaken(t *testing.T) {
	fake, factories := newObsFakeHarness(t)
	fake.uiCreateRule("web", obsFakeRule{Key: "drop-healthz", Name: "UI rule", ProductType: "logs", Query: "x", Operation: "exclude", Rate: 1, Enabled: true})
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config:      obsRuleHCL("drop_healthz", "drop-healthz", "logs", ""),
				ExpectError: regexp.MustCompile(`already exists in project "web"[\s\S]*terraform\s+import\s+<address>\s+web/drop-healthz`),
			},
		},
	})
}

func TestObservabilityFilterRule_versionConflict(t *testing.T) {
	fake, factories := newObsFakeHarness(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{Config: obsRuleHCL("drop_healthz", "drop-healthz", "logs", "")},
			{
				// Someone edits the rule in the UI between Terraform's
				// refresh and its write.
				PreConfig: func() {
					fake.mu.Lock()
					fake.beforeRulePut = func(p *obsFakeProject, key string) { p.rules[key].Version++ }
					fake.mu.Unlock()
				},
				Config:      obsRuleHCL("drop_healthz", "drop-healthz", "logs", "  rate = 0.5"),
				ExpectError: regexp.MustCompile(`changed outside of Terraform[\s\S]*version conflict[\s\S]*409 Conflict \(optimistic_locking_error\)`),
			},
		},
	})
}

func TestObservabilityFilterRule_validation(t *testing.T) {
	_, factories := newObsFakeHarness(t)
	cases := map[string]struct {
		config string
		err    string
	}{
		"bad key":           {obsRuleHCL("r", "-bad", "logs", ""), `Must be 1-256 characters`},
		"long key":          {obsRuleHCL("r", strings.Repeat("a", 257), "logs", ""), `Must be 1-256 characters`},
		"bad product":       {obsRuleHCL("r", "ok", "spans", ""), `value must be one of`},
		"bad operation":     {strings.Replace(obsRuleHCL("r", "ok", "logs", ""), `"exclude"`, `"drop"`, 1), `value must be one of`},
		"rate too high":     {obsRuleHCL("r", "ok", "logs", "  rate = 1.5"), `must be between 0\.0+ and 1\.0+`},
		"metrics rate":      {obsRuleHCL("r", "ok", "metrics", "  rate = 0.5"), `must use .rate = 1.`},
		"attrs on logs":     {obsRuleHCL("r", "ok", "logs", `  attributes = ["a"]`), `only be set when .product_type = "metrics".`},
		"negative limit":    {"resource \"launchdarkly_observability_filter_settings\" \"s\" {\n  project_key        = \"web\"\n  minute_rate_limits = { logs = -1 }\n}\n", `must be at least 0`},
		"duplicate in list": {"resource \"launchdarkly_observability_filter_settings\" \"s\" {\n  project_key = \"web\"\n  rule_order  = { logs = [\"a\", \"a\"] }\n}\n", `duplicate values`},
	}
	steps := []resource.TestStep{}
	for name, tc := range cases {
		t.Logf("case %s", name)
		steps = append(steps, resource.TestStep{Config: tc.config, PlanOnly: true, ExpectError: regexp.MustCompile(tc.err)})
	}
	resource.UnitTest(t, resource.TestCase{ProtoV6ProviderFactories: factories, Steps: steps})
}

func obsSettingsConfig(logsOrder, limits string) string {
	return obsRuleHCL("drop_healthz", "drop-healthz", "logs", "") +
		obsRuleHCL("sample_debug", "sample-debug", "logs", "  rate = 0.1") +
		obsRuleHCL("drop_ping", "drop-ping", "traces", "") + fmt.Sprintf(`
resource "launchdarkly_observability_filter_settings" "web" {
  project_key        = "web"
  minute_rate_limits = %s
  rule_order = {
    logs = %s
  }
}
`, limits, logsOrder)
}

func TestObservabilityFilterSettings_orderDriftImport(t *testing.T) {
	fake, factories := newObsFakeHarness(t)
	addr := "launchdarkly_observability_filter_settings.web"
	healthz := "launchdarkly_observability_filter_rule.drop_healthz.key"
	debug := "launchdarkly_observability_filter_rule.sample_debug.key"
	forward := obsSettingsConfig(fmt.Sprintf("[%s, %s]", healthz, debug), "{ logs = 100000 }")
	reversed := obsSettingsConfig(fmt.Sprintf("[%s, %s]", debug, healthz), "{ logs = 50000, traces = 2000 }")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		CheckDestroy: checkFake(fake, "web", func(rules map[string]obsFakeRule, order map[string][]string, limits map[string]*int64) error {
			for p, v := range limits {
				if v != nil {
					return fmt.Errorf("rate limit for %s not cleared on destroy: %d", p, *v)
				}
			}
			if _, ok := rules["ui-added"]; !ok || len(rules) != 1 {
				return fmt.Errorf("expected only the UI-created rule to remain, got %v", rules)
			}
			if len(order["logs"]) != 0 || len(order["traces"]) != 0 {
				return fmt.Errorf("expected empty orders after destroy, got %v", order)
			}
			return nil
		}),
		Steps: []resource.TestStep{
			{
				Config: forward,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "id", "web"),
					resource.TestCheckResourceAttr(addr, "minute_rate_limits.logs", "100000"),
					resource.TestCheckNoResourceAttr(addr, "minute_rate_limits.traces"),
					resource.TestCheckResourceAttr(addr, "rule_order.logs.#", "2"),
					resource.TestCheckResourceAttr(addr, "rule_order.logs.0", "drop-healthz"),
					resource.TestCheckNoResourceAttr(addr, "rule_order.traces"),
					expectOrder(fake, "web", "logs", "drop-healthz", "sample-debug"),
					// traces is not listed, so its order stays the server's.
					expectOrder(fake, "web", "traces", "drop-ping"),
				),
			},
			{
				Config: reversed,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "rule_order.logs.0", "sample-debug"),
					resource.TestCheckResourceAttr(addr, "minute_rate_limits.traces", "2000"),
					expectOrder(fake, "web", "logs", "sample-debug", "drop-healthz"),
				),
			},
			{
				// A rule created in the UI joins the logs order. Terraform
				// owns that order, so the plan shows drift and the apply
				// takes the UI rule back out (the rule itself is untouched).
				PreConfig: func() {
					fake.uiCreateRule("web", obsFakeRule{Key: "ui-added", Name: "Added in UI", ProductType: "logs", Query: "level=debug", Operation: "exclude", Rate: 1, Enabled: true})
				},
				Config: reversed,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeTestCheckFunc(
					expectOrder(fake, "web", "logs", "sample-debug", "drop-healthz"),
					checkFake(fake, "web", func(rules map[string]obsFakeRule, _ map[string][]string, _ map[string]*int64) error {
						if _, ok := rules["ui-added"]; !ok {
							return fmt.Errorf("UI rule should still exist")
						}
						return nil
					}),
				),
			},
			{
				// Import adopts every product with a rule order; traces is
				// unmanaged in the config, hence the ignore.
				ResourceName:            addr,
				ImportState:             true,
				ImportStateId:           "web",
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"rule_order.traces"},
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 imported state, got %d", len(states))
					}
					if got := states[0].Attributes["rule_order.traces.0"]; got != "drop-ping" {
						return fmt.Errorf("import should adopt the traces order, got %q", got)
					}
					return nil
				},
			},
		},
	})
}

func TestObservabilityFilterSettings_unknownRuleKey(t *testing.T) {
	_, factories := newObsFakeHarness(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config:      "resource \"launchdarkly_observability_filter_settings\" \"web\" {\n  project_key = \"web\"\n  rule_order  = { logs = [\"nope\"] }\n}\n",
				ExpectError: regexp.MustCompile(`unknown rule key "nope"`),
			},
		},
	})
}

func TestObservabilityFilterSettings_projectMissing(t *testing.T) {
	fake, factories := newObsFakeHarness(t)
	config := `resource "launchdarkly_observability_filter_settings" "gone" { project_key = "gone" }`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{Config: config},
			{
				// The project disappears: Read gets a 404 and the settings
				// leave state, so Terraform plans to create them again.
				PreConfig: func() {
					fake.mu.Lock()
					fake.missingProjects["gone"] = true
					fake.mu.Unlock()
				},
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestObservabilityFilterRulesDataSource(t *testing.T) {
	fake, factories := newObsFakeHarness(t)
	fake.uiCreateRule("web", obsFakeRule{Key: "a-logs", Name: "A", ProductType: "logs", Query: "q", Operation: "exclude", Rate: 0.5, Enabled: true})
	fake.uiCreateRule("web", obsFakeRule{Key: "b-metrics", Name: "B", ProductType: "metrics", Query: "q", Operation: "include", Rate: 1, Attributes: []string{"host"}})
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: `
data "launchdarkly_observability_filter_rules" "all" { project_key = "web" }
data "launchdarkly_observability_filter_rules" "metrics" {
  project_key  = "web"
  product_type = "metrics"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.launchdarkly_observability_filter_rules.all", "id", "web"),
					resource.TestCheckResourceAttr("data.launchdarkly_observability_filter_rules.all", "rules.#", "2"),
					resource.TestCheckResourceAttr("data.launchdarkly_observability_filter_rules.metrics", "id", "web/metrics"),
					resource.TestCheckResourceAttr("data.launchdarkly_observability_filter_rules.metrics", "rules.#", "1"),
					resource.TestCheckResourceAttr("data.launchdarkly_observability_filter_rules.metrics", "rules.0.key", "b-metrics"),
					resource.TestCheckResourceAttr("data.launchdarkly_observability_filter_rules.metrics", "rules.0.operation", "include"),
					resource.TestCheckResourceAttr("data.launchdarkly_observability_filter_rules.metrics", "rules.0.attributes.0", "host"),
				),
			},
		},
	})
}

func TestObservabilityRequestHeadersAndURL(t *testing.T) {
	var got *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	defer server.Close()
	client, err := newClient("api-secret", server.URL, false, DEFAULT_HTTP_TIMEOUT_S, 1)
	require.NoError(t, err)

	_, err = listObservabilityFilterRules(context.Background(), client, "web", "logs")
	require.NoError(t, err)
	require.Equal(t, "/api/v2/projects/web/observability/filter-rules", got.URL.Path)
	require.Equal(t, "logs", got.URL.Query().Get("productType"))
	require.Equal(t, []string{"beta"}, got.Header.Values("LD-API-Version"))
	require.Equal(t, "api-secret", got.Header.Get("Authorization"))
	require.Contains(t, got.Header.Get("User-Agent"), "launchdarkly-terraform-provider/")

	require.Equal(t, "https://app.launchdarkly.com/api/v2/projects/p/observability/filter-settings", observabilityURL("app.launchdarkly.com", "projects", "p", "observability", "filter-settings"))
	require.Equal(t, "https://app.launchdarkly.com/api/v2/projects/p", observabilityURL("", "projects", "p"))
}

func TestObservabilityErrorDiagnostics(t *testing.T) {
	var diags diag.Diagnostics
	addObservabilityError(&diags, "Failed", &observabilityAPIError{StatusCode: 409, Code: observabilityCodeVersionConflict, Message: "stale"}, "")
	require.Contains(t, diags[0].Detail(), "version conflict")
	require.Contains(t, diags[0].Detail(), "terraform plan")
	require.Contains(t, diags[0].Detail(), "409 Conflict (optimistic_locking_error): stale")

	diags = nil
	addObservabilityError(&diags, "Failed", &observabilityAPIError{StatusCode: 409, Code: observabilityCodeKeyTaken, Message: "key taken"}, "key hint")
	require.True(t, strings.HasPrefix(diags[0].Detail(), "key hint"))

	// A create can't conflict on version, so its 409 is "key taken" even if
	// the server labels it optimistic_locking_error.
	diags = nil
	addObservabilityError(&diags, "Failed", &observabilityAPIError{StatusCode: 409, Code: observabilityCodeVersionConflict, Message: "key taken"}, "key hint")
	require.True(t, strings.HasPrefix(diags[0].Detail(), "key hint"))

	diags = nil
	addObservabilityError(&diags, "Failed", &observabilityAPIError{StatusCode: 403, Code: "forbidden", Message: "observability filter settings require a member API access token"}, "")
	require.Contains(t, diags[0].Detail(), "Service tokens are not supported")
	require.Contains(t, diags[0].Detail(), "require a member API access token")

	diags = nil
	addObservabilityError(&diags, "Failed", &observabilityAPIError{StatusCode: 400, Code: "invalid_request", Message: "bad query"}, "key hint")
	require.Equal(t, "400 Bad Request (invalid_request): bad query", diags[0].Detail())
}

// The fake must treat a missing beta header the way gonfalon does (403), and
// the provider must surface it with the credential hint.
func TestObservabilityMissingBetaHeaderIs403(t *testing.T) {
	fake := newObsFake()
	server := httptest.NewServer(fake)
	defer server.Close()
	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/v2/projects/web/observability/filter-settings", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "api-x")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// An unknown value must never reach the API: an unknown rule order would be
// sent as an authoritative empty list, and an unknown limit as unlimited.
func TestObservabilitySettingsPayloadRejectsUnknown(t *testing.T) {
	ctx := context.Background()
	r := &ObservabilityFilterSettingsResource{}
	nullLimits := map[string]attr.Value{}
	nullOrders := map[string]attr.Value{}
	for _, p := range observabilityProductTypes {
		nullLimits[p] = types.Int64Null()
		nullOrders[p] = types.ListNull(types.StringType)
	}
	with := func(m map[string]attr.Value, k string, v attr.Value) map[string]attr.Value {
		out := map[string]attr.Value{}
		for kk, vv := range m {
			out[kk] = vv
		}
		out[k] = v
		return out
	}
	model := func(limits, orders map[string]attr.Value) ObservabilityFilterSettingsResourceModel {
		return ObservabilityFilterSettingsResourceModel{
			ProjectKey:       types.StringValue("web"),
			MinuteRateLimits: types.ObjectValueMust(observabilityRateLimitAttrTypes, limits),
			RuleOrder:        types.ObjectValueMust(observabilityRuleOrderAttrTypes, orders),
		}
	}
	cases := map[string]ObservabilityFilterSettingsResourceModel{
		"unknown order list":    model(nullLimits, with(nullOrders, LOGS, types.ListUnknown(types.StringType))),
		"unknown order element": model(nullLimits, with(nullOrders, LOGS, types.ListValueMust(types.StringType, []attr.Value{types.StringValue("a"), types.StringUnknown()}))),
		"unknown limit":         model(with(nullLimits, LOGS, types.Int64Unknown()), nullOrders),
		"unknown objects": {
			ProjectKey:       types.StringValue("web"),
			MinuteRateLimits: types.ObjectUnknown(observabilityRateLimitAttrTypes),
			RuleOrder:        types.ObjectUnknown(observabilityRuleOrderAttrTypes),
		},
	}
	for name, m := range cases {
		var diags diag.Diagnostics
		r.settingsPayload(ctx, m, &diags)
		require.True(t, diags.HasError(), name)
		require.Contains(t, diags[0].Summary(), "unknown", name)
	}

	var diags diag.Diagnostics
	payload := r.settingsPayload(ctx, model(with(nullLimits, LOGS, types.Int64Value(5)), with(nullOrders, LOGS, types.ListValueMust(types.StringType, []attr.Value{}))), &diags)
	require.False(t, diags.HasError())
	require.Equal(t, int64(5), *payload.MinuteRateLimits[LOGS])
	require.Nil(t, payload.MinuteRateLimits[TRACES])
	require.Equal(t, map[string][]string{LOGS: {}}, payload.RuleOrder)
}
