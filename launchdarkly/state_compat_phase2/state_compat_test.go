// Package state_compat_phase2 hosts the wire-compatibility regression
// tests for every Phase 2 leaf resource. Originally a sub-package because
// terraform-plugin-testing/helper/resource and terraform-plugin-sdk/v2/
// helper/resource both registered a `sweep` flag in init() and panicked
// when imported together; Phase 5.1a swapped the root pkg off SDKv2 so
// the constraint is gone. The sub-package remains for cohesion with the
// fixture-capture flow described below.
//
// Capture flow (run once per resource against a disposable LD test
// account):
//
//	LAUNCHDARKLY_ACCESS_TOKEN=<test-account-token> \
//	  ./scripts/capture-state-fixtures/capture.sh access_token_basic
//
// The captured fixture lands under launchdarkly/testdata/state-fixtures/
// and the matching test below stops skipping.
package state_compat_phase2

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"

	"github.com/launchdarkly/terraform-provider-launchdarkly/launchdarkly"
	"github.com/launchdarkly/terraform-provider-launchdarkly/launchdarkly/statecompat"
)

const stateCompatProviderVersion = "2.29.0"

// protoV5Factories serves the framework provider as v5, matching main.go's
// wire protocol. Defined here (not borrowed from launchdarkly/provider_test.go)
// because package-scoped test symbols don't cross package boundaries.
var protoV5Factories = map[string]func() (tfprotov5.ProviderServer, error){
	"launchdarkly": providerserver.NewProtocol5WithError(launchdarkly.NewPluginProvider("test")()),
}

const accessTokenEnvVar = "LAUNCHDARKLY_ACCESS_TOKEN"

func preCheck(t *testing.T) {
	t.Helper()
	if v := os.Getenv(accessTokenEnvVar); v == "" {
		t.Fatalf("%s env var must be set for state-compat tests", accessTokenEnvVar)
	}
}

// stateCompatCase keys both the fixture file and the synthetic HCL
// config off a single stem: <name>.tfstate under state-fixtures/ and
// <name>.tf under scripts/capture-state-fixtures/configs/. Keeping
// them in lockstep is the captured-then-replayed contract.
type stateCompatCase struct {
	name string
}

func (c stateCompatCase) hcl(t *testing.T) string {
	t.Helper()
	abs := filepath.Join(repoRoot(t), "scripts", "capture-state-fixtures", "configs", c.name+".tf")
	b, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("read synthetic config %s: %s", abs, err)
	}
	return string(b)
}

func (c stateCompatCase) fixtureFile() string { return c.name + ".tfstate" }

func (c stateCompatCase) fixtureAbsPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "launchdarkly", statecompat.FixturesDir, c.fixtureFile())
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, here, _, _ := runtime.Caller(0)
	// launchdarkly/state_compat_phase2/ -> launchdarkly/ -> repo root
	return filepath.Join(filepath.Dir(here), "..", "..")
}

func runCase(t *testing.T, c stateCompatCase) {
	t.Helper()
	if _, err := os.Stat(c.fixtureAbsPath(t)); err != nil {
		t.Skipf(
			"fixture %s not captured yet. Run:\n  LAUNCHDARKLY_ACCESS_TOKEN=<test-token> ./scripts/capture-state-fixtures/capture.sh %s",
			c.fixtureFile(), c.name,
		)
		return
	}
	statecompat.Run(t, statecompat.Case{
		HCLConfig:                c.hcl(t),
		FixtureFile:              c.fixtureFile(),
		PreviousVersion:          stateCompatProviderVersion,
		ProtoV5ProviderFactories: protoV5Factories,
		PreCheck:                 func() { preCheck(t) },
	})
}

func TestStateCompatAccessToken_Basic(t *testing.T) {
	runCase(t, stateCompatCase{name: "access_token_basic"})
}

func TestStateCompatAccessToken_CustomRoles(t *testing.T) {
	runCase(t, stateCompatCase{name: "access_token_custom_roles"})
}

func TestStateCompatCustomRole_Basic(t *testing.T) {
	runCase(t, stateCompatCase{name: "custom_role_basic"})
}

func TestStateCompatTeamMember_Basic(t *testing.T) {
	runCase(t, stateCompatCase{name: "team_member_basic"})
}

func TestStateCompatWebhook_Basic(t *testing.T) {
	runCase(t, stateCompatCase{name: "webhook_basic"})
}

func TestStateCompatRelayProxyConfiguration_Basic(t *testing.T) {
	runCase(t, stateCompatCase{name: "relay_proxy_configuration_basic"})
}

func TestStateCompatAITool_Basic(t *testing.T) {
	runCase(t, stateCompatCase{name: "ai_tool_basic"})
}

func TestStateCompatFlagTrigger_Basic(t *testing.T) {
	runCase(t, stateCompatCase{name: "flag_trigger_basic"})
}

func TestStateCompatModelConfig_Basic(t *testing.T) {
	runCase(t, stateCompatCase{name: "model_config_basic"})
}

func TestStateCompatView_Basic(t *testing.T) {
	runCase(t, stateCompatCase{name: "view_basic"})
}

// Phase 4 fixtures.

func TestStateCompatProject_Basic(t *testing.T) {
	runCase(t, stateCompatCase{name: "project_basic"})
}
func TestStateCompatProject_IIS(t *testing.T) {
	runCase(t, stateCompatCase{name: "project_iis"})
}
func TestStateCompatProject_CSA(t *testing.T) {
	runCase(t, stateCompatCase{name: "project_csa"})
}
func TestStateCompatProject_MultiEnv(t *testing.T) {
	runCase(t, stateCompatCase{name: "project_multi_env"})
}
func TestStateCompatProject_ViewAssociation(t *testing.T) {
	runCase(t, stateCompatCase{name: "project_view_association"})
}
func TestStateCompatProject_EnvApprovals(t *testing.T) {
	runCase(t, stateCompatCase{name: "project_env_approvals"})
}

func TestStateCompatSegment_Basic(t *testing.T) {
	runCase(t, stateCompatCase{name: "segment_basic"})
}
func TestStateCompatSegment_IncludedExcluded(t *testing.T) {
	runCase(t, stateCompatCase{name: "segment_included_excluded"})
}
func TestStateCompatSegment_Rules(t *testing.T) {
	runCase(t, stateCompatCase{name: "segment_rules"})
}
func TestStateCompatSegment_RuleRollout(t *testing.T) {
	runCase(t, stateCompatCase{name: "segment_rule_rollout"})
}
func TestStateCompatSegment_Contexts(t *testing.T) {
	runCase(t, stateCompatCase{name: "segment_contexts"})
}
func TestStateCompatSegment_Big(t *testing.T) {
	runCase(t, stateCompatCase{name: "segment_big"})
}

func TestStateCompatFeatureFlag_Boolean(t *testing.T) {
	runCase(t, stateCompatCase{name: "feature_flag_boolean"})
}
func TestStateCompatFeatureFlag_String(t *testing.T) {
	runCase(t, stateCompatCase{name: "feature_flag_string"})
}
func TestStateCompatFeatureFlag_Number(t *testing.T) {
	runCase(t, stateCompatCase{name: "feature_flag_number"})
}
func TestStateCompatFeatureFlag_JSON(t *testing.T) {
	runCase(t, stateCompatCase{name: "feature_flag_json"})
}
func TestStateCompatFeatureFlag_CustomProperties(t *testing.T) {
	runCase(t, stateCompatCase{name: "feature_flag_custom_properties"})
}
func TestStateCompatFeatureFlag_CSA(t *testing.T) {
	runCase(t, stateCompatCase{name: "feature_flag_csa"})
}
func TestStateCompatFeatureFlag_IIS(t *testing.T) {
	runCase(t, stateCompatCase{name: "feature_flag_iis"})
}
func TestStateCompatFeatureFlag_Defaults(t *testing.T) {
	runCase(t, stateCompatCase{name: "feature_flag_defaults"})
}
func TestStateCompatFeatureFlag_Tags(t *testing.T) {
	runCase(t, stateCompatCase{name: "feature_flag_tags"})
}
func TestStateCompatFeatureFlag_Deprecated(t *testing.T) {
	runCase(t, stateCompatCase{name: "feature_flag_deprecated"})
}

func TestStateCompatFFE_SimpleOn(t *testing.T) {
	runCase(t, stateCompatCase{name: "ffe_simple_on"})
}
func TestStateCompatFFE_RulesSingle(t *testing.T) {
	runCase(t, stateCompatCase{name: "ffe_rules_single"})
}
func TestStateCompatFFE_RulesMultiClause(t *testing.T) {
	runCase(t, stateCompatCase{name: "ffe_rules_multi_clause"})
}
func TestStateCompatFFE_RuleRollout(t *testing.T) {
	runCase(t, stateCompatCase{name: "ffe_rule_rollout"})
}
func TestStateCompatFFE_Targets(t *testing.T) {
	runCase(t, stateCompatCase{name: "ffe_targets"})
}
func TestStateCompatFFE_ContextTargets(t *testing.T) {
	runCase(t, stateCompatCase{name: "ffe_context_targets"})
}
func TestStateCompatFFE_Prerequisites(t *testing.T) {
	runCase(t, stateCompatCase{name: "ffe_prerequisites"})
}
func TestStateCompatFFE_FallthroughRollout(t *testing.T) {
	runCase(t, stateCompatCase{name: "ffe_fallthrough_rollout"})
}
func TestStateCompatFFE_OffVariation(t *testing.T) {
	runCase(t, stateCompatCase{name: "ffe_off_variation"})
}
func TestStateCompatFFE_FullyLoaded(t *testing.T) {
	runCase(t, stateCompatCase{name: "ffe_fully_loaded"})
}

// phase2Inventory keys the Phase 2 + Phase 4 fixture set.
// TestStateCompatPhase2_Inventory below logs (but does not fail on) the
// list of fixtures still to capture, giving operators a single
// `go test`-driven checklist. The sub-package's name stuck at
// state_compat_phase2 for backwards-compat — Phase 4 fixtures land
// here too because the sweep-flag isolation argument is identical.
var phase2Inventory = []string{
	"access_token_basic",
	"access_token_custom_roles",
	"ai_tool_basic",
	"custom_role_basic",
	"flag_trigger_basic",
	"model_config_basic",
	"relay_proxy_configuration_basic",
	"team_member_basic",
	"view_basic",
	"webhook_basic",
	// Phase 4 — project (6)
	"project_basic",
	"project_iis",
	"project_csa",
	"project_multi_env",
	"project_view_association",
	"project_env_approvals",
	// Phase 4 — segment (6)
	"segment_basic",
	"segment_included_excluded",
	"segment_rules",
	"segment_rule_rollout",
	"segment_contexts",
	"segment_big",
	// Phase 4 — feature_flag (10)
	"feature_flag_boolean",
	"feature_flag_string",
	"feature_flag_number",
	"feature_flag_json",
	"feature_flag_custom_properties",
	"feature_flag_csa",
	"feature_flag_iis",
	"feature_flag_defaults",
	"feature_flag_tags",
	"feature_flag_deprecated",
	// Phase 4 — feature_flag_environment (10)
	"ffe_simple_on",
	"ffe_rules_single",
	"ffe_rules_multi_clause",
	"ffe_rule_rollout",
	"ffe_targets",
	"ffe_context_targets",
	"ffe_prerequisites",
	"ffe_fallthrough_rollout",
	"ffe_off_variation",
	"ffe_fully_loaded",
}

func TestStateCompatPhase2_Inventory(t *testing.T) {
	fixturesDir := filepath.Join(repoRoot(t), "launchdarkly", statecompat.FixturesDir)
	var missing []string
	for _, name := range phase2Inventory {
		if _, err := os.Stat(filepath.Join(fixturesDir, name+".tfstate")); err != nil {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		return
	}
	msg := fmt.Sprintf("Phase 2 fixtures still to capture (%d of %d remaining):\n", len(missing), len(phase2Inventory))
	for _, name := range missing {
		msg += fmt.Sprintf("  - ./scripts/capture-state-fixtures/capture.sh %s\n", name)
	}
	t.Log(msg)
}
