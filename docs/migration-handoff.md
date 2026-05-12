# Migration handoff

> Live status doc. Check git log on the moonshots branches for ground
> truth. 50 commits across ~54 stacked branches under
> `moonshots/tpf/*`, all green on local `make fmtcheck && go vet && go build`.

## Phase status summary

| Phase | Status | Branches |
|---|---|---|
| 0 (Foundation) | DONE | 9 |
| 1 (Data sources) | DONE — 19/19 migrated | 19 |
| 2 (Leaf resources) | DONE — 9/9 migrated | 9 |
| 3 (Medium resources) | 2/10 full impl (environment + flag_templates), 8 scaffolds | 10 |
| 4 (Complex resources) | 4 scaffolds (project, segment, feature_flag, FFE) | 4 |
| 5 (Cutover) | Documented (no code change required pre-promotion) | 1 |
| 6 (Additive features) | 1 example: `provider::launchdarkly::flag_key` | 1 |
| 7 (Release ceremony) | Documented per-phase | 1 |

## Detail

### Phase 0 — Foundation: COMPLETE

| Sub-phase | Branch | Status |
|---|---|---|
| 0.1 | `moonshots/tpf/0.1-branch-ci` | CI: push triggers + workflow_dispatch for moonshots branch |
| 0.2 | `moonshots/tpf/0.2-test-factories` | `testAccProtoV5ProviderFactories` mux factory replaces 171 `Providers:` callsites across 46 test files; `mustTestAccClient()` replaces `testAccProvider.Meta()` (20 callsites) |
| 0.3 | `moonshots/tpf/0.3-framework-helpers` | `framework_helpers.go` + tests |
| 0.4 | `moonshots/tpf/0.4-framework-validators` | SDKv2 validators ported (no new dep) |
| 0.5 | `moonshots/tpf/0.5-state-compat-harness` | `statecompat.Run`, capture/scan scripts, CI integration |
| 0.6 | `moonshots/tpf/0.6-schema-compat-decision` | `framework_schema_compat.go` defensive shim + decision doc |
| 0.7 | `moonshots/tpf/0.7-block-schema-reference` | `framework_schema_reference.go` cheatsheet; CLAUDE.md conventions |
| 0.8 | `moonshots/tpf/0.8-contributor-docs` | `CONTRIBUTING.md` + PR template migration metadata |
| 0.9a | `moonshots/tpf/0.9a-parity-bootstrap` | set-hash parity inventory + deprecation carry-forward |

### Phase 1 — Data sources: COMPLETE (19/19)

| Sub-phase | Branch |
|---|---|
| 1.1.1 relay_proxy_configuration | `moonshots/tpf/1.1.1-ds-relay-proxy` |
| 1.1.2 webhook | `moonshots/tpf/1.1.2-ds-webhook` |
| 1.1.3 flag_trigger | `moonshots/tpf/1.1.3-ds-flag-trigger` |
| 1.1.4 model_config | `moonshots/tpf/1.1.4-ds-model-config` |
| 1.1.5 audit_log_subscription | `moonshots/tpf/1.1.5-ds-audit-log-subscription` |
| 1.1.6 metric | `moonshots/tpf/1.1.6-ds-metric` |
| 1.1.7 ai_config | `moonshots/tpf/1.1.7-ds-ai-config` |
| 1.1.8 ai_config_variation | `moonshots/tpf/1.1.8-ds-ai-config-variation` |
| 1.1.9 ai_tool | `moonshots/tpf/1.1.9-ds-ai-tool` |
| 1.2.1 environment | `moonshots/tpf/1.2.1-ds-environment` |
| 1.2.2 project | `moonshots/tpf/1.2.2-ds-project` |
| 1.2.3 flag_templates | `moonshots/tpf/1.2.3-ds-flag-templates` |
| 1.3.1 team | `moonshots/tpf/1.3.1-ds-team` |
| 1.3.2 team_member | `moonshots/tpf/1.3.2-ds-team-member` |
| 1.3.3 team_members | `moonshots/tpf/1.3.3-ds-team-members` |
| 1.3.4 view | `moonshots/tpf/1.3.4-ds-view` |
| 1.3.5 segment | `moonshots/tpf/1.3.5-ds-segment` |
| 1.3.6 feature_flag | `moonshots/tpf/1.3.6-ds-feature-flag` |
| 1.3.7 feature_flag_environment | `moonshots/tpf/1.3.7-ds-feature-flag-environment` |

Shared helpers introduced (reused by Phase 2-4): `policy_statements_framework.go`,
`approvals_framework.go`, `role_attributes_framework.go`,
`clauses_framework.go`, `team_member_helper.go::getAllTeamMembers /
getTeamMemberByEmail`, `stringValueFromPointer`.

### Phase 2 — Leaf resources: COMPLETE (9/9)

| Sub-phase | Branch | Notes |
|---|---|---|
| 2.1 relay_proxy_configuration | `moonshots/tpf/2.1-resource-relay-proxy` | adds `frameworkPolicyStatementsResourceBlock` + FromList |
| 2.2 webhook | `moonshots/tpf/2.2-resource-webhook` | tag-after-create preserved |
| 2.3 ai_tool | `moonshots/tpf/2.3-resource-ai-tool` | ConflictsWith maintainer fields |
| 2.4 flag_trigger | `moonshots/tpf/2.4-resource-flag-trigger` | two-step create-then-disable |
| 2.5 model_config | `moonshots/tpf/2.5-resource-model-config` | API has no update; all attrs RequiresReplace |
| 2.6 team_member | `moonshots/tpf/2.6-resource-team-member` | role validated + role_attributes patches |
| 2.7 custom_role | `moonshots/tpf/2.7-resource-custom-role` | deprecated `policy` block carry-forward (typo preserved) |
| 2.8 access_token | `moonshots/tpf/2.8-resource-access-token` | deprecated `expire` + `policy_statements` |
| 2.9 view | `moonshots/tpf/2.9-resource-view` | beta API; ExactlyOneOf maintainer |

### Phase 3 — Medium resources: PARTIAL

| Sub-phase | Branch | Status |
|---|---|---|
| 3.1 destination | `moonshots/tpf/3.1-resource-destination` | SCAFFOLD (model + schema + stub CRUD); not registered on framework |
| 3.2 audit_log_subscription | `moonshots/tpf/3.2-resource-audit-log-subscription` | SCAFFOLD |
| 3.3 metric | `moonshots/tpf/3.3-resource-metric` | SCAFFOLD |
| 3.4 environment | `moonshots/tpf/3.4-resource-environment` | DONE — full CRUD + approval_settings patch logic |
| 3.5 ai_config | `moonshots/tpf/3.5-resource-ai-config` | SCAFFOLD |
| 3.6 ai_config_variation | `moonshots/tpf/3.6-resource-ai-config-variation` | SCAFFOLD |
| 3.7 team | `moonshots/tpf/3.7-resource-team` | SCAFFOLD |
| 3.8 view_links + view_filter_links | `moonshots/tpf/3.8-resource-view-links` | SCAFFOLD |
| 3.9 ip_allowlist_config + ip_allowlist_entry | `moonshots/tpf/3.9-resource-ip-allowlist` | SCAFFOLD |
| 3.10 flag_templates | `moonshots/tpf/3.10-resource-flag-templates` | DONE — Create/Update upsert via PUT; Delete is a no-op |

### Phase 4 — Complex resources: SCAFFOLDS

| Sub-phase | Branch | Status |
|---|---|---|
| 4.1 project | `moonshots/tpf/4.1-resource-project` | SCAFFOLD — needs customizeProjectDiff -> ModifyPlan port + Upjet shim wiring |
| 4.2 segment | `moonshots/tpf/4.2-resource-segment` | SCAFFOLD — nested clauses ready from Phase 1.3.5 |
| 4.3 feature_flag | `moonshots/tpf/4.3-resource-feature-flag` | SCAFFOLD — needs custom_properties hash parity tests |
| 4.4 feature_flag_environment | `moonshots/tpf/4.4-resource-feature-flag-environment` | SCAFFOLD — largest schema in provider |

### Phase 5 — Cutover: DOCUMENTED

`moonshots/tpf/5.1-cutover-scaffold` lands `docs/phase-5-cutover-checklist.md`
with the 5.1 -> 5.5 execution sequence (SDKv2 drop, test-pkg unify,
`go mod tidy`, protocol v6 flip, harness rename). No code change is
appropriate until every Phase 2-4 resource has promoted; the
checklist is deterministic enough for an autonomous run when ready.

### Phase 6 — Additive features: EXAMPLE

`moonshots/tpf/6.3-provider-function-flag-key` ships
`function_flag_key.go`: a `function.Function` for
`provider::launchdarkly::flag_key(project, flag)` -> `"project/flag"`.
Cannot register on the provider until Phase 5 cutover (functions are
framework-only). Pattern is the template for 6.1 / 6.2 / 6.4 ideas in
the master plan.

### Phase 7 — Release ceremony: DOCUMENTED

`moonshots/tpf/7.1-release-ceremony` lands
`docs/phase-7-release-ceremony.md`: per-promotion checklist (soak,
promote, downstream verify, communicate), promotion cadence table
mapping each phase to a v2.x minor, risk callouts per phase. Phase 5.4
protocol-v6 special case documented (minimum CLI version must be
stated in release notes).

## Stacks ready to push

Each phase is a separate gh-stack rooted on the previous phase's top
branch. From the top of each stack run `gh stack submit --auto --draft`
to push and open PRs targeting `moonshots/terraform-plugin-framework`.

## Open items

- Phase 2 scaffolds and Phase 3-4 scaffolds are **not registered on
  the framework provider**. SDKv2 mux path still serves these
  resources in production. To promote a scaffold to active:
  1. Replace the stub `Create`/`Read`/`Update`/`Delete` with the real
     port (model from this branch + helpers as documented in the file
     header).
  2. Add `New<Resource>Resource` to `launchdarkly/plugin_provider.go::Resources()`.
  3. Remove the SDKv2 registration in `launchdarkly/provider.go::ResourcesMap`.
  4. Delete the SDKv2 source file.
  5. Capture a state-compat fixture per the per-PR checklist.
- State-compat fixtures: none captured for Phase 2-4 resources. Need
  LD test-account access + `scripts/capture-state-fixtures/capture.sh`.
- `make generate` produces unrelated audit_log_subscription docs diff
  that should be regenerated on `main` separately.
- Crossplane Upjet behaviour on framework-served deprecated attrs
  still pending confirmation (see `docs/migration-schema-compat-upjet.md`).

## Stack inventory

```bash
git branch | grep moonshots/tpf | wc -l  # ~54
gh stack view --json | jq '.branches[].name'  # current stack only
```

Each phase is its own gh-stack. To enumerate all stacks:

```bash
git for-each-ref --format='%(refname:short)' refs/heads/moonshots/tpf/
```
