# Set-hash parity inventory

> Source: `.claude/MIGRATION_PLAN_NON_BREAKING.md` §Phase 0.9a.
> Status: bootstrap docs landed. Per-resource parity fixtures land
> incrementally per §Phase 0.9b alongside each Phase 2-4 migration PR.
> Gates: Phase 1 promotion to `main`; each Phase 2/3/4 promotion
> requires fixtures for the resources migrated in that phase.

## Why this matters

`terraform-plugin-sdk/v2` lets schemas attach a custom `SchemaSetFunc`
(`Set: someHashFunc`) that turns each element into an `int`. The SDK
uses that integer as the canonical identity of the element inside a
`*schema.Set`. Two elements with the same hash are considered the same
element; ordering inside a set is determined by hash value, not source
order.

`terraform-plugin-framework` has no equivalent — framework Sets
(`types.Set`) compare by element value (full structural equality of the
encoded `tftypes.Value`). A custom hash function is therefore not
directly portable; what matters is that the **framework representation
preserves wire-equivalent semantics for whichever cases the SDKv2 hash
function was collapsing or distinguishing**.

The migration goal: every SDKv2 set whose hash function ignores
inner-list ordering (or some other property) must, after the migration,
still treat element-permutations as the same element so existing state
files don't diff.

## Inventory

| Site | Hash function | Element shape | Framework representation | Round-trip risks |
|---|---|---|---|---|
| `custom_properties_helper.go:21` (`custom_properties` block on `launchdarkly_feature_flag`) | `customPropertyHash` (`:117`) | `{Key string, Name string, Value []string}` | `types.Set` of `types.Object` with `key: String`, `name: String`, `value: List<String>` | `value` list reordering. SDKv2 hash sorts `Value` before stringifying so `["a","b"]` and `["b","a"]` hash identical. Framework's value comparison treats the lists as different. **Mitigation**: serialize `value` as sorted on write to state. Test: capture fixture with reordered `value`; assert zero diff. |
| `policies_helper.go:14` (`policy` block on `launchdarkly_custom_role`, deprecated) | `policyHash` (`:101`) | `{Resources []string, Actions []string, Effect string}` | `types.Set` of `types.Object` with `resources: List<String>`, `actions: List<String>`, `effect: String` | `Resources` and `Actions` list reordering. SDKv2 doesn't pre-sort — `policyHash` stringifies as-is. So SDKv2 actually distinguishes element-order in those lists, which means framework already matches that semantic. **Mitigation**: none required, but verify via fixture. Deprecated block — Phase 2.7 (`launchdarkly_custom_role`) carries it forward via `DeprecationMessage`. |
| `tags_helper.go:14` (`tags` attribute on every resource that has one) | `schema.HashString` | `string` | `types.Set` of `String` | None — framework Set on `String` is order-independent by definition. **Test**: capture fixture with reordered tags; assert zero diff. |
| `resource_launchdarkly_access_token.go:64` (`custom_roles` set) | `schema.HashString` | `string` | `types.Set` of `String` | None. Same as tags. |
| `resource_launchdarkly_team_member.go:60` (custom_roles on member) | `schema.HashString` | `string` | `types.Set` of `String` | None. |
| `resource_launchdarkly_view_links.go:72` (`flag_keys` set) | `schema.HashString` | `string` | `types.Set` of `String` | None. |
| `data_source_launchdarkly_team_member.go:42` (custom_roles on member data source) | `schema.HashString` | `string` | `types.Set` of `String` | None. |

## Migration strategy per shape

### Custom hash funcs (`customPropertyHash`, `policyHash`)

These are the highest-risk sites because the SDKv2 hash function
encoded a normalization that the framework Set wouldn't otherwise
apply.

For `customPropertyHash`:

- Decision: **normalize on write to state**. Sort the `value` list
  before constructing the framework `types.Set` element. The user's
  config is parsed into the same canonical form, so reorders in the
  config produce no plan diff.
- Phase 4.3 (`launchdarkly_feature_flag`) implements this.
- Parity test: synthetic fixture with `value = ["foo", "bar"]` and
  another with `value = ["bar", "foo"]` — both produce the same
  framework Set after the migration.

For `policyHash`:

- Decision: **none required**. SDKv2 itself didn't normalize the
  `Resources` / `Actions` lists (it stringified the struct verbatim),
  so element-order is meaningful in both. Framework reproduces that
  semantic by default.
- Phase 2.7 (`launchdarkly_custom_role`) carries the deprecated
  `policy` block forward.
- Parity test: synthetic fixture with `resources = ["proj/a",
  "proj/b"]` — assert zero diff after migration.

### `schema.HashString` (5 sites)

- Decision: **none required**. Framework `types.Set` of `String`
  compares by value, not hash; permutations always collapse to the
  same Set.
- Parity tests live per-resource alongside Phase 2-4 migration PRs.

## Per-phase parity gate

Promotion of each phase to `main` (per the rolling gate in
`MIGRATION_PLAN_NON_BREAKING.md` §Phase 0.9b) requires:

- **Phase 2**: parity tests for every resource in Phase 2 that uses a
  Set (custom hash or HashString). Specifically:
  - 2.6 `launchdarkly_team_member` (custom_roles HashString)
  - 2.7 `launchdarkly_custom_role` (policy via policyHash)
  - 2.8 `launchdarkly_access_token` (custom_roles HashString)
  - 2.9 `launchdarkly_view` (any Set-shaped attrs)
- **Phase 3**: every resource here with a Set:
  - 3.7 `launchdarkly_team` (custom_roles / members)
  - 3.8 `launchdarkly_view_links` (flag_keys HashString)
- **Phase 4**: the high-risk cases:
  - 4.1 `launchdarkly_project` (tags HashString)
  - 4.2 `launchdarkly_segment` (rules / clauses — Set-of-string fields)
  - 4.3 `launchdarkly_feature_flag` (custom_properties via
    customPropertyHash + tags HashString)
  - 4.4 `launchdarkly_feature_flag_environment` (rules + clauses,
    Set-of-string fields)

## What "parity test" means concretely

A parity test is a `launchdarkly/statecompat/`-based round-trip
assertion that:

1. Applies a synthetic config against the pinned v2.29 SDKv2-only
   provider to produce a legacy-encoded state file.
2. Re-runs the same config against the in-tree framework-served
   provider.
3. Plancheck `ExpectEmptyPlan` asserts zero diff.

For sites where ordering matters, the test is run twice — once with
the "natural" element order, once with a permuted order — and both
must produce zero diff. Failures here are the canary that the
framework representation is dropping or reordering Set elements.

## Refresh policy

Re-run this audit:

- Whenever a new resource adds a Set-shaped attribute.
- Whenever a SDKv2 Set's hash function changes (rare, but check
  `git log launchdarkly/*_helper.go` before each phase).
- When framework releases change the structural equality semantics of
  `types.Set` (would invalidate the no-test-required assumption for
  `HashString` sites).
