// Package state_compat_phase2 hosts the wire-compatibility regression
// tests for every Phase 2 leaf resource. It lives in its own sub-package
// because terraform-plugin-testing/helper/resource (used by
// launchdarkly/statecompat) and terraform-plugin-sdk/v2/helper/resource
// (used by the existing launchdarkly/* acceptance tests) both register a
// `sweep` flag in init(). Importing both into one test binary panics —
// statecompat.harness.go:7-11 documents the constraint.
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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-mux/tf5muxserver"

	"github.com/launchdarkly/terraform-provider-launchdarkly/launchdarkly"
	"github.com/launchdarkly/terraform-provider-launchdarkly/launchdarkly/statecompat"
)

const stateCompatProviderVersion = "2.29.0"

// protoV5Factories serves the same tf5muxserver as main.go. Defined
// here (not borrowed from launchdarkly/provider_test.go) because
// package-scoped test symbols don't cross package boundaries.
var protoV5Factories = map[string]func() (tfprotov5.ProviderServer, error){
	"launchdarkly": func() (tfprotov5.ProviderServer, error) {
		ctx := context.Background()
		return tf5muxserver.NewMuxServer(ctx,
			launchdarkly.Provider().GRPCProvider,
			providerserver.NewProtocol5(launchdarkly.NewPluginProvider("test")()),
		)
	},
}

const accessTokenEnvVar = "LAUNCHDARKLY_ACCESS_TOKEN"

func preCheck(t *testing.T) {
	t.Helper()
	if v := os.Getenv(accessTokenEnvVar); v == "" {
		t.Fatalf("%s env var must be set for state-compat tests", accessTokenEnvVar)
	}
}

// stateCompatCase pairs a fixture filename with the synthetic HCL
// config that produced it. The HCL string is loaded from
// scripts/capture-state-fixtures/configs/<config-path> so the replay
// step exercises the same source-of-truth config the fixture was
// captured from.
type stateCompatCase struct {
	fixtureName string
	configPath  string
}

func (c stateCompatCase) hcl(t *testing.T) string {
	t.Helper()
	abs := filepath.Join(repoRoot(t), "scripts", "capture-state-fixtures", "configs", c.configPath)
	b, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("read synthetic config %s: %s", abs, err)
	}
	return string(b)
}

func (c stateCompatCase) fixtureAbsPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "launchdarkly", statecompat.FixturesDir, c.fixtureName)
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
		shortName := c.fixtureName[:len(c.fixtureName)-len(".tfstate")]
		t.Skipf(
			"fixture %s not captured yet. Run:\n  LAUNCHDARKLY_ACCESS_TOKEN=<test-token> ./scripts/capture-state-fixtures/capture.sh %s",
			c.fixtureName, shortName,
		)
		return
	}
	statecompat.Run(t, statecompat.Case{
		HCLConfig:                c.hcl(t),
		FixtureFile:              c.fixtureName,
		PreviousVersion:          stateCompatProviderVersion,
		ProtoV5ProviderFactories: protoV5Factories,
		PreCheck:                 func() { preCheck(t) },
	})
}

func TestStateCompatAccessToken_Basic(t *testing.T) {
	runCase(t, stateCompatCase{
		fixtureName: "access_token_basic.tfstate",
		configPath:  "access_token_basic.tf",
	})
}

func TestStateCompatAccessToken_CustomRoles(t *testing.T) {
	runCase(t, stateCompatCase{
		fixtureName: "access_token_custom_roles.tfstate",
		configPath:  "access_token_custom_roles.tf",
	})
}

func TestStateCompatCustomRole_Basic(t *testing.T) {
	runCase(t, stateCompatCase{
		fixtureName: "custom_role_basic.tfstate",
		configPath:  "custom_role_basic.tf",
	})
}

func TestStateCompatTeamMember_Basic(t *testing.T) {
	runCase(t, stateCompatCase{
		fixtureName: "team_member_basic.tfstate",
		configPath:  "team_member_basic.tf",
	})
}

func TestStateCompatWebhook_Basic(t *testing.T) {
	runCase(t, stateCompatCase{
		fixtureName: "webhook_basic.tfstate",
		configPath:  "webhook_basic.tf",
	})
}

func TestStateCompatRelayProxyConfiguration_Basic(t *testing.T) {
	runCase(t, stateCompatCase{
		fixtureName: "relay_proxy_configuration_basic.tfstate",
		configPath:  "relay_proxy_configuration_basic.tf",
	})
}

func TestStateCompatAITool_Basic(t *testing.T) {
	runCase(t, stateCompatCase{
		fixtureName: "ai_tool_basic.tfstate",
		configPath:  "ai_tool_basic.tf",
	})
}

func TestStateCompatFlagTrigger_Basic(t *testing.T) {
	runCase(t, stateCompatCase{
		fixtureName: "flag_trigger_basic.tfstate",
		configPath:  "flag_trigger_basic.tf",
	})
}

func TestStateCompatModelConfig_Basic(t *testing.T) {
	runCase(t, stateCompatCase{
		fixtureName: "model_config_basic.tfstate",
		configPath:  "model_config_basic.tf",
	})
}

func TestStateCompatView_Basic(t *testing.T) {
	runCase(t, stateCompatCase{
		fixtureName: "view_basic.tfstate",
		configPath:  "view_basic.tf",
	})
}

// phase2Inventory enumerates every fixture the Phase 2 plan requires.
// TestStateCompatPhase2_Inventory below logs (but does not fail on) the
// list of fixtures still to capture, giving operators a single
// `go test`-driven checklist.
var phase2Inventory = []string{
	"access_token_basic.tfstate",
	"access_token_custom_roles.tfstate",
	"ai_tool_basic.tfstate",
	"custom_role_basic.tfstate",
	"flag_trigger_basic.tfstate",
	"model_config_basic.tfstate",
	"relay_proxy_configuration_basic.tfstate",
	"team_member_basic.tfstate",
	"view_basic.tfstate",
	"webhook_basic.tfstate",
}

func TestStateCompatPhase2_Inventory(t *testing.T) {
	fixturesDir := filepath.Join(repoRoot(t), "launchdarkly", statecompat.FixturesDir)
	var missing []string
	for _, name := range phase2Inventory {
		if _, err := os.Stat(filepath.Join(fixturesDir, name)); err != nil {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		return
	}
	msg := fmt.Sprintf("Phase 2 fixtures still to capture (%d of %d remaining):\n", len(missing), len(phase2Inventory))
	for _, f := range missing {
		short := f[:len(f)-len(".tfstate")]
		msg += fmt.Sprintf("  - ./scripts/capture-state-fixtures/capture.sh %s\n", short)
	}
	t.Log(msg)
}
