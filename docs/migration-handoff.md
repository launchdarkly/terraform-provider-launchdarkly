# Migration handoff

> Live status doc. Each session updates this; check git log on the
> moonshots branches for ground truth.

## Phase 0 — Foundation: COMPLETE

Stack: `moonshots/tpf/0.1-branch-ci` → … → `moonshots/tpf/0.9a-parity-bootstrap`.
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

## Phase 1 — Data Source Migration: COMPLETE (19/19)

Stack: `moonshots/tpf/1.1.4-ds-model-config` → ... → `moonshots/tpf/1.3.7-ds-feature-flag-environment`.

| Sub-phase | Data source | Branch |
|---|---|---|
| 1.1.4 | model_config | `moonshots/tpf/1.1.4-ds-model-config` |
| 1.1.1 | relay_proxy_configuration | `moonshots/tpf/1.1.1-ds-relay-proxy` |
| 1.1.2 | webhook | `moonshots/tpf/1.1.2-ds-webhook` |
| 1.1.3 | flag_trigger | `moonshots/tpf/1.1.3-ds-flag-trigger` |
| 1.1.5 | audit_log_subscription | `moonshots/tpf/1.1.5-ds-audit-log-subscription` |
| 1.1.6 | metric | `moonshots/tpf/1.1.6-ds-metric` |
| 1.1.7 | ai_config | `moonshots/tpf/1.1.7-ds-ai-config` |
| 1.1.8 | ai_config_variation | `moonshots/tpf/1.1.8-ds-ai-config-variation` |
| 1.1.9 | ai_tool | `moonshots/tpf/1.1.9-ds-ai-tool` |
| 1.2.1 | environment | `moonshots/tpf/1.2.1-ds-environment` |
| 1.2.2 | project | `moonshots/tpf/1.2.2-ds-project` |
| 1.2.3 | flag_templates | `moonshots/tpf/1.2.3-ds-flag-templates` |
| 1.3.1 | team | `moonshots/tpf/1.3.1-ds-team` |
| 1.3.2 | team_member | `moonshots/tpf/1.3.2-ds-team-member` |
| 1.3.3 | team_members | `moonshots/tpf/1.3.3-ds-team-members` |
| 1.3.4 | view | `moonshots/tpf/1.3.4-ds-view` |
| 1.3.5 | segment | `moonshots/tpf/1.3.5-ds-segment` |
| 1.3.6 | feature_flag | `moonshots/tpf/1.3.6-ds-feature-flag` |
| 1.3.7 | feature_flag_environment | `moonshots/tpf/1.3.7-ds-feature-flag-environment` |

Shared framework helpers introduced during Phase 1 (reused by the
upcoming resource migrations in Phases 2-4):

- `policy_statements_framework.go` — block schema + value converter
  for the `policy` / `statements` block.
- `approvals_framework.go` — `approval_settings` block schema and
  ApprovalSettings -> framework value converter.
- `role_attributes_framework.go` — `role_attributes` set block.
- `clauses_framework.go` — clauses ListNestedBlock + Clause slice
  converter (consumed by segment + feature_flag_environment).
- `team_member_helper.go` — extracted `getTeamMemberByEmail` +
  `getAllTeamMembers` so the legacy SDKv2 data source files could be
  deleted while preserving the helpers for the resource side.
- `stringValueFromPointer` (in `data_source_team_framework.go`) — used
  across resources whose ldapi fields are `*string`.

## Phase 2 — Leaf Resource Migration: STARTED (2 of 9)

Stack: `moonshots/tpf/2.5-resource-model-config` → `moonshots/tpf/2.3-resource-ai-tool`.

| Sub-phase | Resource | Status |
|---|---|---|
| 2.5 | `launchdarkly_model_config` | done — full Create/Read/Update/Delete/ImportState; API has no update so every attr carries `RequiresReplace`; Delete preserves "still in use" guidance |
| 2.3 | `launchdarkly_ai_tool` | done — full Create/Read/Update (PATCH-style)/Delete; ConflictsWith between maintainer_id and maintainer_team_key implemented as `resource.ConfigValidator` |
| 2.1 | `launchdarkly_relay_proxy_configuration` | pending |
| 2.2 | `launchdarkly_webhook` | pending |
| 2.4 | `launchdarkly_flag_trigger` | pending |
| 2.6 | `launchdarkly_team_member` | pending |
| 2.7 | `launchdarkly_custom_role` | pending — carry `policy` deprecation forward |
| 2.8 | `launchdarkly_access_token` | pending — carry `expire` + `policy_statements` deprecations forward |
| 2.9 | `launchdarkly_view` | pending |

**State-compat fixtures**: NONE of the Phase 2 migrations have a
fixture captured under `launchdarkly/testdata/state-fixtures/` yet —
this requires running `scripts/capture-state-fixtures/capture.sh`
against a real LD test account, which the autonomous-session execution
could not perform. Per the per-PR checklist in
MIGRATION_PLAN_NON_BREAKING.md §Per-PR, fixtures must be captured
before each Phase 2 PR is promoted to `main`.

## Phase 3 — Medium-complexity Resource Migration: COMPLETE (12/12)

Branch: `moonshots/tpf/phase-3` (worked on locally; merges directly to
the integration trunk rather than via sub-branches per phase).

| Sub-phase | Resource | Status |
|---|---|---|
| 3.1 | `launchdarkly_destination` | done — kind enum + 5-vendor config converter; mparticle `user_identities` JSON + `user_identity` drift suppression; obfuscated-secret preservation (api_key/secret/write_key/policy_key) |
| 3.2 | `launchdarkly_audit_log_subscription` | done — integration_key enum from `SUBSCRIPTION_CONFIGURATION_FIELDS` + extras; snake→camel/kebab config conversion; secret pass-through from prior state; statements via `frameworkPolicyStatementsResourceBlock` |
| 3.3 | `launchdarkly_metric` | done — full `customizeMetricDiff` ported to `ModifyPlan` (kind→required-fields, percentile gating, defaults for unit_aggregation_type/analysis_type, include_units_without_events default by analysis_type) |
| 3.4 | `launchdarkly_environment` | done — `applyApprovalPatch` preserves SDKv2 exclusivity rules (required vs required_approval_tags; launchdarkly service_kind vs auto_apply); environmentExists / environmentExistsInProject retained as package-level helpers for still-SDKv2 project/segment/FFE consumers |
| 3.5 | `launchdarkly_ai_config` | done — ConflictsWith maintainer_id / maintainer_team_key implemented as `ConfigValidator`; `is_inverted` requires `evaluation_metric_key`; delete-retry on transient 400 ("Could not delete AI config") |
| 3.6 | `launchdarkly_ai_config_variation` | done — versioned reads pick highest Items[] Version (per memory note); JSON-equivalence plan modifier suppresses spurious model diffs; ModelConfigKey/Model ConfigValidator; tool_keys preserved when API returns empty (API gap); strip_empty_model defaults parity |
| 3.7 | `launchdarkly_team` | done — patch-with-instructions Update (`updateName`, `updateDescription`, `addMembers`/`removeMembers`, `addPermissionGrants`/`removePermissionGrants`, `addCustomRoles`/`removeCustomRoles`, `replaceRoleAttributes`); members + custom roles + maintainers paginated; `interfaceToArr` + `makeAddAndRemoveArrays` retained in `team_helper.go` for still-SDKv2 `resource_team_role_mapping.go` |
| 3.8 | `launchdarkly_view_links` + `launchdarkly_view_filter_links` | done — both beta API; explicit-link drift detection for view_links; filter-based with optional `reconcile_on_apply` triggering `ModifyPlan` to mark `resolved_at` unknown; AtLeastOneOf / RequiredWith implemented as `ConfigValidator`; `difference` + `differenceSegmentIdentifiers` retained in `view_helper.go` for still-SDKv2 feature_flag/segment consumers |
| 3.9 | `launchdarkly_ip_allowlist_config` + `launchdarkly_ip_allowlist_entry` | done — singleton config resource (Delete resets to defaults rather than removing server-side); entry uses RequiresReplace on ip_address (matches ForceNew); BETA API via custom client with `LD-API-Version: beta` header |
| 3.10 | `launchdarkly_flag_templates` | done — Create/Update upsert collapse into PUT /flag-defaults; Delete is a no-op (templates always exist); CSA passed through from current API state to avoid conflict with launchdarkly_project ownership of CSA |

All 10 SDKv2 source files deleted. Helpers (`difference`, `differenceSegmentIdentifiers`, `interfaceToArr`, `makeAddAndRemoveArrays`, `destinationImportIDtoKeys`, `CUSTOM_METRIC_DEFAULT_SUCCESS_CRITERIA`, `shouldRetryAIConfigDelete`) retained / relocated where Phase 4 SDKv2 code still depends on them. SDKv2 test files preserved — they continue to run through the mux factory (Phase 0.2).

Build/vet/fmt all clean on the worktree. Acceptance tests not yet
run; state-fixture configs prepared under
`scripts/capture-state-fixtures/configs/` and ready for capture.

## State-fixture status (Phase 2 + Phase 3)

Synthetic v2.29-pinned configs land under `scripts/capture-state-fixtures/configs/`. Capture flow per `reference_ld_state_compat_capture.md`:

```bash
make build  # so $GOPATH/bin/terraform-provider-launchdarkly exists
TF_CLI_CONFIG_FILE=/tmp/terraformrc-noop \
  LAUNCHDARKLY_ACCESS_TOKEN=<test-account> \
  LAUNCHDARKLY_API_HOST=app.launchdarkly.com \
  ./scripts/capture-state-fixtures/capture.sh <fixture-name>
```

Configs prepared for Phase 3 (capture pending against a live LD test account):

- `destination_basic.tf`
- `audit_log_subscription_basic.tf`
- `metric_custom_basic.tf`
- `metric_pageview_basic.tf`
- `environment_basic.tf`
- `ai_config_basic.tf`
- `ai_config_variation_basic.tf`
- `team_basic.tf`
- `view_links_basic.tf`
- `view_filter_links_basic.tf`
- `ip_allowlist_config_basic.tf`
- `ip_allowlist_entry_basic.tf`
- `flag_templates_basic.tf`

## Phases 4-7: NOT STARTED

| Phase | Scope | Blocker |
|---|---|---|
| 4.1 | `launchdarkly_project` (398 LOC + `customizeProjectDiff`) | high-risk; needs careful ModifyPlan port and fixtures for IIS / CSA edge cases |
| 4.2 | `launchdarkly_segment` (419 LOC) | nested rules + clauses; existing shared helpers from Phase 1.3.5 reused |
| 4.3 | `launchdarkly_feature_flag` (475 LOC) | variations + customPropertyHash parity; deprecated `include_in_snippet` carry-forward |
| 4.4 | `launchdarkly_feature_flag_environment` (327 LOC + sprawling helpers) | deepest schema; CustomizeDiff -> ModifyPlan |
| 5 | SDKv2 drop, test-pkg unification, protocol v6 cutover | unblocks after Phase 4 lands |
| 6 | Additive features (write-only attrs, ephemeral resources, provider functions, actions) | post-cutover, ongoing |
| 7 | Release ceremony | rolling per phase |

## How to continue

### Land what's done

From the top of each stack run `gh stack submit --auto --draft`. Each
sub-phase becomes one PR targeting `moonshots/terraform-plugin-framework`.
The stacks are independent (each phase rooted on the previous phase's
top branch) so the order is:

1. Phase 0 stack (9 PRs)
2. Phase 1 stack (19 PRs)
3. Phase 2 stack (2 PRs)
4. Phase 3 (single branch `moonshots/tpf/phase-3`; no sub-stack — all 12 resource migrations land as one PR against the integration trunk)

Promotion to `main` follows the soak-then-batch rhythm in
MIGRATION_PLAN_NON_BREAKING.md §Phase 7.

### Resume Phase 2 / Phase 3

The model_config + ai_tool commits are the template. For each remaining
leaf resource:

1. Read the SDKv2 resource file + its `*_helper.go`.
2. Create `resource_<name>_framework.go` mirroring the pattern:
   - Model struct with `tfsdk` tags matching `keys.go`.
   - `Schema()` with the same Required/Optional/Computed flags;
     ForceNew → RequiresReplace plan modifier; Deprecated →
     DeprecationMessage verbatim. Per
     `feedback_framework_default_computed.md`: any attribute with a
     framework `Default` must also be `Computed: true`, even when the
     SDKv2 schema isn't — this is the one forced exception to the
     non-breaking-parity rule.
   - Create/Read/Update/Delete/ImportState ports of the SDKv2 funcs.
   - `readIntoModel` helper shared between Create and Read.
   - `CustomizeDiff` → `ModifyPlan`; `ConflictsWith` /
     `AtLeastOneOf` / `RequiredWith` → `ConfigValidator`;
     `DiffSuppressFunc` → `planmodifier.String` (see
     `jsonEquivalentPlanModifier` in
     `resource_ai_config_variation_framework.go` for the JSON
     equivalence pattern).
3. Register on framework, remove from SDKv2 `ResourcesMap`, delete
   SDKv2 file, build/vet/fmt, commit. If the SDKv2 file carried
   package-level helpers that Phase 4 SDKv2 code still references,
   move them to a `*_helper.go` file rather than deleting them.
4. Capture state-fixture via `scripts/capture-state-fixtures/capture.sh`
   against a test LD account; assert via the
   `launchdarkly/statecompat/` harness. See
   `reference_ld_state_compat_capture.md` for the `TF_CLI_CONFIG_FILE`
   override (bypasses local `dev_overrides` so v2.29 SDKv2 produces
   the legacy-shaped state).

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
- The fork PR workflow (`.github/workflows/test-fork-pr.yml`) was
  inspected during Phase 0.1 and confirmed not to filter on base ref
  — it should accept fork PRs against the moonshots integration
  branch without modification, but this was not verified end-to-end.

### Stack inventory

31 branches under `moonshots/tpf/*` plus the integration trunk
`moonshots/terraform-plugin-framework`. Verify with:

```bash
git branch | grep moonshots/tpf | wc -l
gh stack view --json | jq '.branches[].name'
```

### Crossplane coordination still open

`docs/migration-schema-compat-upjet.md` notes the open question for
the Crossplane provider-launchdarkly maintainers: does Upjet's
runtime-schema-stripping behaviour reproduce on framework-served
schemas? The defensive shim ships either way; this only affects whether
`schema_compat.go` (the SDKv2 shim) can retire in Phase 5.2.
