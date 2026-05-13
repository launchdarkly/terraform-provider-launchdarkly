# Deprecated-attribute carry-forward

> Source: `.claude/MIGRATION_PLAN_NON_BREAKING.md` §Phase 0.9a.
> Status: bootstrap inventory. Refresh before any Phase 2-4 PR that
> touches a listed resource.

The SDKv2 -> terraform-plugin-framework migration is non-breaking by
construction. Every attribute marked `Deprecated:` in SDKv2 today must
carry forward into framework with the **same `DeprecationMessage:` text
verbatim**, the same functional behaviour, and the same conflict /
override semantics. Drop is a future-major-version conversation.

## Carry-forward inventory

| # | Attribute | Owning resource / data source | SDKv2 message (verbatim) | Migration phase |
|---|---|---|---|---|
| 1 | `include_in_snippet` | `launchdarkly_project` (`resource_launchdarkly_project.go:99`) | `"'include_in_snippet' is now deprecated. Please migrate to 'default_client_side_availability' to maintain future compatibility."` | 4.1 |
| 2 | `client_side_availability` | `launchdarkly_project` data source (`data_source_launchdarkly_project.go:30`) | `"'client_side_availability' is now deprecated. Please migrate to 'default_client_side_availability' to maintain future compatibility."` | 1.2 (data source migration) |
| 3 | `include_in_snippet` | `launchdarkly_feature_flag` (`feature_flags_helper.go:75`) | `"'include_in_snippet' is now deprecated. Please migrate to 'client_side_availability' to maintain future compatability."` (note: original retains the "compatability" typo) | 4.3 |
| 4 | `policy` (block) | `launchdarkly_custom_role` (via `policies_helper.go:16`) | `"'policy' is now deprecated. Please migrate to 'policy_statements' to maintain future compatability."` (typo preserved) | 2.7 |
| 5 | `policy_statements` | `launchdarkly_access_token` (`resource_launchdarkly_access_token.go:29`) | `"'policy_statements' is deprecated in favor of 'inline_roles'. This field will be removed in the next major release of the LaunchDarkly provider"` | 2.8 |
| 6 | `expire` | `launchdarkly_access_token` (`resource_launchdarkly_access_token.go:93`) | `"'expire' is deprecated and will be removed in the next major release of the LaunchDarkly provider"` | 2.8 |
| 7 | (legacy field) | `launchdarkly_metric` (`metrics_helper.go:66`) | `"No longer in use. This field will be removed in a future major release of the LaunchDarkly provider."` | 3.3 |

Read positions in `metrics_helper.go:66` (item 7) to identify the
specific deprecated field name when migrating Phase 3.3.

## Carry-forward rule

For each entry above, the framework migration PR for the owning
resource:

1. **Copies the deprecation string verbatim** into the framework
   schema's `DeprecationMessage:` field — including typos like
   "compatability". Users see this string in their `terraform plan`
   output; changing the wording is a user-visible behaviour change.
2. **Preserves the attribute's `Optional` / `Computed` flags** so
   configs that set the deprecated value continue to apply with no
   diff.
3. **Preserves any `ConflictsWith:` relationship** by porting to a
   framework `resource.ConfigValidator` (e.g.
   `resourcevalidator.Conflicting`) that exposes the same path
   combinations.
4. **Keeps the functional behaviour intact.** Deprecated isn't dead —
   the underlying API call still routes through the deprecated field
   the same way SDKv2 did.

## Verification

A migration PR for any resource above lands a state-compat fixture in
`launchdarkly/testdata/state-fixtures/` whose synthetic config sets the
deprecated attribute. The harness in `launchdarkly/statecompat/`
asserts:

- The deprecated value appears in plan output with the same warning.
- Zero plan diff after the framework swap.
- A second fixture that does *not* set the deprecated attribute also
  round-trips cleanly (regression test against accidentally requiring
  it).

## When does deprecation eligible-for-deletion arrive?

Not in this migration. Removal lands in a future major version (v3.x
or later) with an explicit upgrade guide. The non-breaking guarantee
in MIGRATION_PLAN_NON_BREAKING.md rules out deletion across v2.x.
