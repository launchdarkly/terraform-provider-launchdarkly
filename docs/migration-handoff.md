# Migration handoff — session 2026-05-12

> Live status doc. Each session updates this; check git log on the
> moonshots branches for ground truth.

## Phase 0 — Foundation: COMPLETE

Stack: `moonshots/tpf/0.1-branch-ci` → ... → `moonshots/tpf/0.9a-parity-bootstrap`.
9 branches, 9 commits, all green on local `make fmtcheck && go vet && go build`.

| Sub-phase | Branch | Status |
|---|---|---|
| 0.1 | `moonshots/tpf/0.1-branch-ci` | done — push triggers + workflow_dispatch for moonshots branch |
| 0.2 | `moonshots/tpf/0.2-test-factories` | done — `testAccProtoV5ProviderFactories` mux factory replaces 171 `Providers:` callsites across 46 test files; `mustTestAccClient()` replaces `testAccProvider.Meta()` (20 callsites) |
| 0.3 | `moonshots/tpf/0.3-framework-helpers` | done — `framework_helpers.go` + tests; team_role_mapping refactored to consume |
| 0.4 | `moonshots/tpf/0.4-framework-validators` | done — SDKv2 validation_helper.go ported to `framework_validators.go` (no new external deps; native `validator.String` interface) |
| 0.5 | `moonshots/tpf/0.5-state-compat-harness` | done — `statecompat.Run`, capture/scan/safe-placeholders scripts, CI integration, fixture-safety policy locked |
| 0.6 | `moonshots/tpf/0.6-schema-compat-decision` | done — `framework_schema_compat.go` ships defensively; decision doc in `docs/migration-schema-compat-upjet.md` |
| 0.7 | `moonshots/tpf/0.7-block-schema-reference` | done — `framework_schema_reference.go` worked example + cheatsheet; CLAUDE.md migration conventions section |
| 0.8 | `moonshots/tpf/0.8-contributor-docs` | done — `CONTRIBUTING.md` + PR template migration metadata |
| 0.9a | `moonshots/tpf/0.9a-parity-bootstrap` | done — set-hash parity inventory + deprecation carry-forward docs |

**Phase 0 promotion gate**: ready. Recommended next action is `gh stack
submit --auto --draft` from the 0.9a branch (with the maintainer's
review) so CI runs against the full stack on the moonshots integration
branch.

## Phase 1 — Data Source Migration: STARTED (1 of 19)

Stack: `moonshots/tpf/1.1.4-ds-model-config` (currently a 1-branch stack
rooted on `moonshots/tpf/0.9a-parity-bootstrap`).

| Sub-phase | Status |
|---|---|
| 1.1.4 `launchdarkly_model_config` | done — `data_source_model_config_framework.go`; SDKv2 file deleted; registered on framework provider; existing acceptance test compiles against the new factory |
| 1.1.1 `launchdarkly_relay_proxy_configuration` | pending |
| 1.1.2 `launchdarkly_webhook` | pending |
| 1.1.3 `launchdarkly_flag_trigger` | pending |
| 1.1.5 `launchdarkly_audit_log_subscription` | pending |
| 1.1.6 `launchdarkly_metric` | pending |
| 1.1.7 `launchdarkly_ai_config` | pending |
| 1.1.8 `launchdarkly_ai_config_variation` | pending |
| 1.1.9 `launchdarkly_ai_tool` | pending |
| 1.2.x (nested-shape) | pending — `environment`, `project`, `flag_templates`, `ai_config_variation` data source |
| 1.3.x (complex) | pending — `segment`, `feature_flag`, `feature_flag_environment`, `view` |

**Pattern is proven by 1.1.4.** The model_config migration commit is
the template for the remaining 18 data sources. Per-data-source effort
estimate (after pattern is internalised): 30-60 min for 1.1.x sources,
1-2 hours for 1.2.x, 2-4 hours for 1.3.x sources.

## Phases 2-7: NOT STARTED

Per the plan, these depend on Phase 1 being live and require:

- Phase 2 (9 leaf resources): each migration writes a state-compat
  fixture; effort ~2-4 days per resource for a human, hard to compress
  in an autonomous session because each needs a v2.29 fixture captured
  against a real LD test account.
- Phase 3 (10 medium resources): same risks; CustomizeDiff →
  ModifyPlan needs careful per-resource thought.
- Phase 4 (4 complex resources): project / segment / feature_flag /
  feature_flag_environment. 1-2 weeks per resource.
- Phase 5 (cutover): only viable once Phases 2-4 land.
- Phase 6 (additive features): post-cutover.
- Phase 7 (release): rolling per-phase soak.

**Honest scoping note**: Phases 2-7 are months of engineering work even
for an experienced human. Attempting them in autonomous batches risks
shipping unsafe migrations (state-fixture parity is the existence test
— without LD account access the harness can't run end-to-end).

## How to continue

### Land Phase 0
1. Review the 9 Phase 0 commits (`git log moonshots/tpf/0.9a-parity-bootstrap`).
2. From `moonshots/tpf/0.9a-parity-bootstrap`: `gh stack submit --auto
   --draft`. Each sub-phase becomes one PR targeting
   `moonshots/terraform-plugin-framework`.
3. Wait for CI to go green on each PR; merge in order
   (`gh stack` enforces base-branch chaining automatically).

### Resume Phase 1
1. `gh stack checkout moonshots/tpf/1.1.4-ds-model-config` (after Phase
   0 lands; or work in parallel with the moonshots branch updated).
2. For each pending data source: copy the model_config migration as a
   template:
   - Create `data_source_<name>_framework.go` with a
     `datasource.DataSource` implementation.
   - Add `Newxxx` to `plugin_provider.go::DataSources()`.
   - Remove entry from `provider.go::DataSourcesMap`.
   - Delete the old `data_source_launchdarkly_<name>.go`.
   - Run `go build ./...`, `go vet ./...`, `make fmtcheck`.
   - `gh stack add 1.1.N-ds-<name>` to push to the next branch in
     this stack.
   - Commit per data source.
3. Promote to `main` as v2.30.0 after Phase 1.3 closes.

### Schema-compat callout

If Crossplane Upjet responds confirming framework-side runtime
stripping behaves identically to SDKv2 (see
`docs/migration-schema-compat-upjet.md`), tighten the matchers in
`framework_schema_compat.go` against the live error shape. If they
confirm it does *not* affect framework, schedule `schema_compat.go`
deletion for Phase 5.2 and consider deleting
`framework_schema_compat.go` too.

### Open items recorded in commit log

- `make generate` produced a diff in audit_log_subscription docs +
  integration_configs_generated.go during Phase 0 work; this was
  reverted to keep the stack focused on migration changes. A
  follow-up commit on `main` (separate from migration work) should
  capture the regenerated integration manifests.
- The terraform-plugin-framework-validators dep was tried in Phase
  0.4 and rejected because it forced a grpc upgrade incompatible
  with terraform-plugin-sdk/v2. Re-evaluate after Phase 5.1 drops
  SDKv2.
