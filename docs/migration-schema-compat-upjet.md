# Schema-compat decision: SDKv2 -> terraform-plugin-framework

> Source: `.claude/MIGRATION_PLAN_NON_BREAKING.md` Phase 0.6.
> Status: defensive shim shipped; Crossplane investigation pending.

## Context

`launchdarkly/schema_compat.go` exists because Crossplane's Upjet — which
embeds this provider — sometimes strips deprecated attributes from the
runtime schema. Reads / writes against those keys then fail with
specific terraform-plugin-sdk/v2 error shapes:

- `Invalid address to set: []string{"<attr>"}` — emitted by
  `schema.ResourceData.Set` / `MapFieldWriter.WriteField` when the target
  attribute is absent from the runtime schema.
- `: invalid key: <attr>` — emitted by `schema.ResourceDiff.SetNew` via
  the same code path, with a slightly different wrapping.

`resourceDataSetSkipMissingKey` and `resourceDiffSetNewSkipMissingKey`
swallow exactly those two shapes (via `isOmittedEmbeddedSchemaAttrErr`)
and log at DEBUG. Anything else flows through unchanged so unrelated
errors still surface.

The migration question: **does this behaviour reproduce on framework-served
schemas?** If yes, we need an analogous shim before migrating any
deprecated-attribute-bearing resource (Phase 4.1 `launchdarkly_project`
with `include_in_snippet`, Phase 4.3 `launchdarkly_feature_flag` with the
same attribute, etc.). If no, the SDKv2 shim is dead code once
`schema_compat.go` retires in Phase 5.2.

## What changes in framework

Framework data accessors route writes through `fwschemadata.SetAtPath`
(see `terraform-plugin-framework@v1.9.0/internal/fwschemadata/
data_set_at_path.go`). When the supplied path resolves to nothing in the
schema, the function emits an `AttributeError` diagnostic:

- **Summary**: `"<Description> Write Error"` where `<Description>` is
  one of `"Config"`, `"Plan"`, or `"State"` — the diagnostics emitter
  tags the surface that's being written to.
- **Detail**: starts with `"An unexpected error was encountered trying
  to retrieve type information at a given path."` followed by the
  underlying error from `Schema.TypeAtTerraformPath`.

This shape is materially different from SDKv2's error strings, so the
SDKv2 matchers in `isOmittedEmbeddedSchemaAttrErr` cannot match it. A
framework-side matcher needs its own implementation.

## Investigation status

| Item | Status |
|---|---|
| Reproduced Upjet-strip on framework schema with a synthetic test rig | Not yet — requires building a minimal embedder that mirrors Upjet's runtime stripping. Filed as follow-up. |
| Filed issue with Crossplane provider-launchdarkly maintainers | Pending. Tracked in `MIGRATION_PLAN_NON_BREAKING.md` open decisions §1. |
| Confirmed framework's emitted diagnostic shape | Yes — read of fwschemadata source (see "What changes in framework" above). |
| Framework-side shim authored | Yes — see `launchdarkly/framework_schema_compat.go`. |

## Decision

**Ship the framework shim defensively in Phase 0.6.** Reasoning:

- The cost of having `framework_schema_compat.go` and discovering Upjet
  doesn't trigger it is small: ~80 LOC + 6 unit tests, scheduled for
  removal in Phase 5.2 if confirmed dead.
- The cost of *not* shipping it and discovering Upjet does trigger it is
  larger: every Phase 2-4 PR that migrates a deprecated-attribute-bearing
  resource breaks Crossplane consumers until we land a hotfix. Phase 4.1
  (`launchdarkly_project` with `include_in_snippet`) is the canonical
  blast radius.
- The shim is opt-in: callers route writes through
  `stateSetSkipMissingKey` / `planSetSkipMissingKey` only on deprecated
  attributes. Resources without deprecated attributes pay zero cost.

## Matcher discipline

`isOmittedFrameworkAttrDiag` matches three conditions simultaneously:

1. The diagnostic summary has the suffix `"Write Error"` — covers
   `"Config Write Error"`, `"Plan Write Error"`, `"State Write Error"`.
2. The diagnostic detail starts with the canonical "unexpected error
   trying to retrieve type information at a given path" prefix.
3. The diagnostic carries an attribute path equal to the supplied
   target path.

Loosening any of the three risks swallowing unrelated errors. Tighten
cautiously when Crossplane provides the actual repro.

## Phase 4 callers (forecast)

The following attributes use `resourceDataSetSkipMissingKey` /
`resourceDiffSetNewSkipMissingKey` in SDKv2 and will need the framework
analogue when their owning resource migrates:

| Attribute | Resource | SDKv2 call sites | Phase |
|---|---|---|---|
| `include_in_snippet` | `launchdarkly_project` | 3 in `resource_launchdarkly_project.go`, 2 in `project_helper.go` | 4.1 |
| `client_side_availability` | `launchdarkly_project` | 1 in `project_helper.go` | 4.1 |
| `include_in_snippet` | `launchdarkly_feature_flag` | 2 in `resource_launchdarkly_feature_flag.go` | 4.3 |
| `expire` | `launchdarkly_access_token` | 1 in `resource_launchdarkly_access_token.go` | 2.8 |
| `policy_statements` | `launchdarkly_access_token` | 1 in `resource_launchdarkly_access_token.go` | 2.8 |

The actual list is an inventory snapshot — refresh when the per-resource
migration PRs land. Audit with `grep -rn
"resourceDataSetSkipMissingKey\|resourceDiffSetNewSkipMissingKey"
launchdarkly/` before starting each phase.

## Crossplane coordination

Open question to file upstream (`crossplane-contrib/
provider-upjet-launchdarkly`):

1. Confirm whether Upjet's runtime-schema generation strips deprecated
   attributes from framework-served `provider.Provider.Schema()` the
   same way it does from SDKv2-served `*schema.Provider.Schema`. If
   yes, what's the precise error path?
2. If yes, share a minimal reproduction we can pin a regression test
   against (currently we're matching on diagnostic shape inferred from
   reading framework source, not from a live failure).
3. If no, confirm the SDKv2 shim's removal in Phase 5.2 won't break
   them.

Until that conversation closes, ship `framework_schema_compat.go` and
keep the dead-code-removal item open in `MIGRATION_PLAN_NON_BREAKING.md`
Phase 5.2.

## Revisit triggers

Revisit this doc when any of the following lands:

- Crossplane response on the runtime-stripping question.
- A new deprecated attribute is introduced on any framework-served
  resource.
- The framework's `data_set_at_path.go` wording changes (would break
  the matcher).
- Phase 5.2 SDKv2-drop cleanup — at which point either remove the
  framework shim entirely (if confirmed dead) or keep it and delete
  `schema_compat.go`.
