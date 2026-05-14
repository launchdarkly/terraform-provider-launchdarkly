// Package statecompat contains the wire-compatibility harness used to
// guarantee that v2.29 SDKv2-produced state files round-trip without a
// spurious plan diff once the corresponding resource migrates to
// terraform-plugin-framework.
//
// Historically this lived in a sub-package because terraform-plugin-testing's
// helper/resource and terraform-plugin-sdk/v2/helper/resource both register
// a `sweep` flag in init(), and importing both into one test binary panics.
// Phase 5.1a swapped the root pkg onto terraform-plugin-testing, so the
// constraint is gone — the sub-package remains only because the captured
// fixtures and reconstruction logic are cohesive here.
//
// Per-resource fixtures land under launchdarkly/testdata/state-fixtures/
// captured by scripts/capture-state-fixtures/capture.sh and scanned by
// scripts/capture-state-fixtures/scan.sh on every CI run. See
// .claude/MIGRATION_PLAN_NON_BREAKING.md §Phase 0.5 + §Phase 0.9b for
// the rolling-gate contract.
package statecompat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// FixturesDir is the canonical relative path (from the launchdarkly
// package) where committed fixtures live. Sub-package tests resolve it
// against repo root via filepath.Join("..", "testdata", "state-fixtures",
// fixtureName).
const FixturesDir = "testdata/state-fixtures"

// Case describes a single round-trip assertion.
//
//   - HCLConfig: the .tf source whose plan should match the fixture
//     once the new provider takes over.
//   - FixtureFile: filename under <repo>/launchdarkly/testdata/state-fixtures/.
//     The harness validates the file is well-formed JSON; the contents
//     themselves are exercised via the ExternalProviders apply step.
//   - PreviousVersion: SDKv2-only provider version (e.g. "2.29.0") used
//     to seed legacy-encoded state via the upstream registry.
//   - ProtoV5ProviderFactories: factory map for the in-tree
//     framework-served provider. Pass the same value
//     launchdarkly/provider_test.go uses for acceptance tests.
//   - PreCheck: invoked before the test runs; usually wraps
//     testAccPreCheck.
type Case struct {
	HCLConfig                string
	FixtureFile              string
	PreviousVersion          string
	ProtoV5ProviderFactories map[string]func() (tfprotov5.ProviderServer, error)
	PreCheck                 func()
}

// Run is the single entry point Phase 2-4 migration tests call to
// assert wire-compatibility against a captured fixture.
//
// Contract:
//  1. Fixture validated as JSON (catches dropped or truncated files
//     before terraform-plugin-testing panics on parse).
//  2. ExternalProviders pins the previous SDKv2 release; the harness
//     applies HCLConfig against it, producing legacy-encoded state.
//  3. A second TestStep switches to ProtoV5ProviderFactories pointed
//     at the in-tree framework-served provider. ExpectEmptyPlan
//     asserts zero diff — the round-trip success signal.
//
// Skipped when TF_ACC is unset: step two configures the live provider
// against a real LD account, matching `make testacc` semantics.
func Run(t *testing.T, c Case) {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("state-compat harness requires TF_ACC=1 (provider configures against live LD)")
	}
	if c.PreviousVersion == "" {
		t.Fatalf("statecompat.Case.PreviousVersion must be set (e.g. \"2.29.0\")")
	}
	if c.FixtureFile == "" {
		t.Fatalf("statecompat.Case.FixtureFile must be set")
	}
	if c.ProtoV5ProviderFactories == nil {
		t.Fatalf("statecompat.Case.ProtoV5ProviderFactories must be supplied")
	}

	path := filepath.Join("..", FixturesDir, c.FixtureFile)
	if err := AssertFixtureIsJSON(path); err != nil {
		t.Fatalf("fixture %s failed validation: %s", path, err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: c.PreCheck,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"launchdarkly": {
						Source:            "launchdarkly/launchdarkly",
						VersionConstraint: c.PreviousVersion,
					},
				},
				Config: c.HCLConfig,
			},
			{
				ProtoV5ProviderFactories: c.ProtoV5ProviderFactories,
				Config:                   c.HCLConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// AssertFixtureIsJSON parses the fixture as JSON so dropped or
// truncated files surface as a clear test failure rather than a
// downstream terraform-plugin-testing panic.
//
// Exported so tooling (e.g. future Go-side fixture validators) can
// reuse the same definition of "well-formed".
func AssertFixtureIsJSON(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read fixture: %w", err)
	}
	var parsed any
	if err := json.Unmarshal(b, &parsed); err != nil {
		return fmt.Errorf("fixture is not valid JSON: %w", err)
	}
	return nil
}
