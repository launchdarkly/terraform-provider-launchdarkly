---
name: terraform-provider-block-to-nested-attrs
description: Migrate LaunchDarkly Terraform provider HCL configs between block syntax (provider v2.x and earlier) and nested-attribute syntax (v3.x+). Use when a user upgrades the launchdarkly provider to v3 and hits "Unsupported block type" / "Missing required argument" plan errors, when downgrading from v3 to v2.x, when porting a v2 example to v3 syntax (or vice versa), or when the user mentions "block to nested attribute", "= [{...}]", "v3 plan errors", or pastes errors that reference `inline_roles`, `statements`, `policy`, `policy_statements`, `environments`, `default_client_side_availability`, `rules`, `clauses`, `variations`, `client_side_availability`, `defaults`, `fallthrough`, `prerequisites`, `targets`, `context_targets`, `urls`, `role_attributes`, `messages`, `boolean_defaults`, `included_contexts`, `excluded_contexts`, `instructions`, `segments`, or `approval_settings`.
compatibility: Works on any directory containing `.tf` files that use `launchdarkly_*` resources. No external tools required beyond a working `terraform` CLI for validation.
metadata:
  author: ffeldberg
  version: "3.0.0"
---

# LaunchDarkly Terraform Provider: Block ↔ Nested Attribute Migration

The LaunchDarkly Terraform provider v3.0.0 finished the migration from `terraform-plugin-sdk/v2` to `terraform-plugin-framework`. Every former `schema.Block` is now a nested attribute. HCL that worked against v2.x using block syntax (`name { ... }`) fails to parse against v3 with `Unsupported block type` or `Missing required argument`. The fix is mechanical, but the target syntax depends on the attribute's cardinality:

- **List/Set nested attributes** (genuinely plural, e.g. `variations`, `rules`, `statements`) → `name = [{ ... }]`.
- **Single nested attributes** (genuinely one object — `client_side_availability`, `defaults`, `default_client_side_availability`, `fallthrough`, `approval_settings`, `segment_approval_settings`, `instructions`, `boolean_defaults`) → `name = { ... }`. These were modeled as max-1 lists through the `3.0.0-beta` pre-releases and switched to single objects for GA (REL-14237), so the bracketless object form is the correct v3.0.0 syntax. **If you are on a `3.0.0-beta.N` pre-release, use the list form `= [{ ... }]` instead** — the object form is GA-only.
- **Map nested attributes** (keyed by a natural key — `launchdarkly_project.environments` by env `key`, `launchdarkly_feature_flag.custom_properties` by property `key`, `launchdarkly_ai_agent_graph.edges` by edge `key`) → `name = { "<key>" = { ... } }`. Each block's `key` value becomes the map key; the `key` attribute is also **kept inside** the object (Optional+Computed in v3, equals the map key) so `.environments["x"].key` references keep working (REL-14236). Reordering/adding/removing one entry no longer churns the others.
- **Plain map attribute** (`role_attributes` on `launchdarkly_team` / `launchdarkly_team_member`) → `role_attributes = { "<key>" = ["<values>", ...] }`. The `{key, values}` object collapses entirely: the map key is the role attribute key and the value is the string list, matching `launchdarkly_team_role_mapping`.

This skill enumerates every affected attribute, gives the exact rewrite, and lists the gotchas that bite during migration.

## When to use

- User runs `terraform plan` against v3 provider and gets `Unsupported block type` for any of the attributes in the mapping table below.
- User asks to "convert blocks to nested attributes" / "migrate config to v3" / "downgrade to v2 syntax".
- User pastes an HCL example using `name { ... }` and asks for the v3 equivalent (or vice versa).

## Do NOT use

- For provider source code changes (use `terraform-provider-add-resource` instead).
- For non-LaunchDarkly providers — the mappings here are specific to `launchdarkly_*` resources.

## Core rule

Most former blocks are now **list-of-objects** (or set-of-objects). HCL syntax for those is `name = [{ ... }, { ... }]`.

```hcl
# v2 (block)              # v3 (list/set nested attribute)
foo {                     foo = [{
  bar = "baz"               bar = "baz"
}                         }]

foo { x = 1 }             foo = [
foo { x = 2 }               { x = 1 },
                            { x = 2 },
                          ]
```

**Exception — single-object attributes** (`client_side_availability`, `defaults`, `default_client_side_availability`, `fallthrough`, `approval_settings`, `segment_approval_settings`, `instructions`) are `SingleNestedAttribute` in v3.0.0 GA. They take a **bare object, no brackets**:

```hcl
# v2 (block)                       # v3.0.0 GA (single nested attribute)
client_side_availability {         client_side_availability = {
  using_environment_id = true        using_environment_id = true
}                                  }
```

**Exception — one map attribute** (`launchdarkly_project.environments`) is a `MapNestedAttribute` in v3 (REL-14236), keyed by the environment `key`. Each block's `key` value becomes the map key; the `key` attribute stays inside the object (it equals the map key):

```hcl
# v2 (block)              # v3 (map nested attribute)
environments {            environments = {
  key   = "production"      "production" = {
  name  = "Production"        key   = "production"
}                             name  = "Production"
                            }
                          }
```

To go v3 → v2, do the inverse: for list attributes strip `= [` / `]` and split `},` separators into a new `foo {` per element; for the four single-object attributes just drop `= ` and the braces become a block (`foo = { ... }` → `foo { ... }`); for the `environments` map, emit one `environments { ... }` block per map entry (the `key` is already inside the object; if a hand-written map omitted it, re-inject `key = "<map key>"`).

## Mapping table

Every attribute that changed from block → nested attribute in v3. The **Type** column drives the syntax: `List` / `Set` render as `= [{...}]`; `Object` (the four single-nested attributes) renders as `= {...}` with no brackets; `Map` (`environments`) renders as `= { "<key>" = {...} }`.

| Resource | Attribute | Underlying type | Notes |
|---|---|---|---|
| `launchdarkly_access_token` | `inline_roles` | List | Replaces deprecated `policy_statements`. |
| `launchdarkly_access_token` | `policy_statements` | List | Deprecated; prefer `inline_roles`. |
| `launchdarkly_audit_log_subscription` | `statements` | List | Required. |
| `launchdarkly_webhook` | `statements` | List | Optional. |
| `launchdarkly_custom_role` | `policy` | Set | Deprecated; prefer `policy_statements`. |
| `launchdarkly_custom_role` | `policy_statements` | List | |
| `launchdarkly_relay_proxy_configuration` | `policy` | List | Required. |
| `launchdarkly_project` | `default_client_side_availability` | **Object** | v3.0.0 GA: `= { ... }`. Was List (max 1) through `3.0.0-beta.3`. |
| `launchdarkly_project` | `environments` | **Map** | Keyed by env `key`: `= { "<key>" = { key = "<key>", ... } }`. The `key` stays inside the object (Optional+Computed, equals the map key). Required, at least one entry; authoritative (an env removed from the map is deleted). Use `lifecycle { ignore_changes = [environments] }` to manage environments outside Terraform. Was an ordered List through the early v3 preview (REL-14236). |
| `launchdarkly_project.environments["<key>"]` | `approval_settings` | **Object** | Nested inside each environment map value. v3.0.0 GA: `= { ... }`. Was List (max 1) through the betas. |
| `launchdarkly_environment` | `approval_settings` | **Object** | Same shape as inline-in-project. v3.0.0 GA: `= { ... }`. |
| `launchdarkly_environment` | `segment_approval_settings` | **Object** | Net-new in v3 (beta approvals API). `= { ... }`. |
| `launchdarkly_segment` | `included_contexts` | List | |
| `launchdarkly_segment` | `excluded_contexts` | List | |
| `launchdarkly_segment` | `rules` | List | |
| `launchdarkly_segment.rules[*]` | `clauses` | List | Nested inside each rule. |
| `launchdarkly_flag_trigger` | `instructions` | **Object** | Required. v3.0.0 GA: `= { kind = "..." }`. Was List (max 1) through the betas. |
| `launchdarkly_view_links` | `segments` | Set | Wrap `environment_id` values in `nonsensitive()` to avoid set-hash churn — see `view.tf` in `local-testing/full-account-v2.29.original` for an example. |
| `launchdarkly_metric` | `urls` | List | |
| `launchdarkly_team` | `role_attributes` | **Plain map** | `= { "<key>" = ["<values>"] }` — the `{key, values}` object collapses to a map entry. Was Set through the betas. |
| `launchdarkly_team_member` | `role_attributes` | **Plain map** | Same shape as on `launchdarkly_team`. |
| `launchdarkly_ai_config_variation` | `messages` | List | |
| `launchdarkly_flag_templates` | `boolean_defaults` | **Object** | Required. v3.0.0 GA: `= { ... }`. Was List (max 1) through the betas. |
| `launchdarkly_feature_flag_environment` | `prerequisites` | List | |
| `launchdarkly_feature_flag_environment` | `targets` | Set | |
| `launchdarkly_feature_flag_environment` | `context_targets` | Set | |
| `launchdarkly_feature_flag_environment` | `rules` | List | |
| `launchdarkly_feature_flag_environment.rules[*]` | `clauses` | List | Nested inside each rule. |
| `launchdarkly_feature_flag_environment` | `fallthrough` | **Object** | Required. v3.0.0 GA: `= { ... }`. Was List (max 1) through `3.0.0-beta.3`. |
| `launchdarkly_feature_flag` | `client_side_availability` | **Object** | v3.0.0 GA: `= { ... }`. Was List (max 1) through `3.0.0-beta.3`. |
| `launchdarkly_feature_flag` | `variations` | List | **Required (min 1) in v3, including for boolean flags.** See gotcha §1. |
| `launchdarkly_feature_flag` | `custom_properties` | **Map** | Keyed by property `key`: `= { "<key>" = { name = ..., value = [...] } }`. The `key` stays inside the object (Optional+Computed, equals the map key). Was Set through the betas. |
| `launchdarkly_feature_flag` | `defaults` | **Object** | v3.0.0 GA: `= { ... }`. Was List (max 1) through `3.0.0-beta.3`. |
| `launchdarkly_ai_agent_graph` | `edges` | **Map** | Net-new in v3. Keyed by edge `key`: `= { "<key>" = { source_config = ..., target_config = ... } }`. |

If an attribute on a `launchdarkly_*` resource is not listed here, it was either already a primitive (no migration needed) or it was a `config` map (which uses `= { ... }` map syntax — not list-of-objects — and stayed the same across versions).

## Gotchas — v2 → v3

1. **`launchdarkly_feature_flag.variations` is now required for every flag, including booleans.** v2 inferred boolean variations; v3 requires them explicitly. Add:
   ```hcl
   variations = [
     { value = "true" },
     { value = "false" },
   ]
   ```
   `value` is a string — `"true"` / `"false"`, not the bare bool. The provider parses it per `variation_type`. The `migrate-tf-syntax` tool now synthesizes this automatically for flags whose `variation_type` is the literal `"boolean"` (value-only). Variation `name`/`description` set outside Terraform are preserved by the provider when the config omits them, so no manual step is needed.

2. **`policy` on `launchdarkly_custom_role` and `policy_statements` on `launchdarkly_access_token` are deprecated** but still parse. Plan emits `Attribute Deprecated` warnings. Migrating to the newer name (`policy_statements` / `inline_roles`) is recommended but not required for the v2→v3 cutover itself.

3. **`launchdarkly_view_links.segments` uses set semantics.** If `environment_id` is sourced from a data source field marked `Sensitive` (e.g. `data.launchdarkly_environment.x.client_side_id`), the set hash will be unstable across plans. Wrap the value in `nonsensitive(...)` to stabilize the hash. Without this you get perpetual "segments updated" drift.

4. **Single-object vs map.** Several shapes use brace-ish syntax — don't confuse them:
   - **Single objects** (no brackets, `= { ... }`): `client_side_availability`, `defaults` (feature_flag), `default_client_side_availability` (project), `fallthrough` (flag_environment), `approval_settings` (environment + project envs), `segment_approval_settings` (environment), `instructions` (flag_trigger), `boolean_defaults` (flag_templates). These are `SingleNestedAttribute` in v3.0.0 GA. A bracketed list here fails with a type error. (Through the `3.0.0-beta` pre-releases they were max-1 lists — if you target a beta pre-release, use brackets.)
   - **Maps of objects** (keyed object, `= { "<key>" = { ... } }`): `launchdarkly_project.environments`, `launchdarkly_feature_flag.custom_properties`, `launchdarkly_ai_agent_graph.edges`. The top-level keys are entry keys, each mapping to an object. A list `= [{ ... }]` here fails with `map of object required`. Reference elements as `environments["<key>"]`, never `environments[0]`.
   - **Plain maps** (`= { "<key>" = [ ... ] }`): `role_attributes` on team / team_member — the value is the string list directly, no inner object.

5. **`config` blocks on `launchdarkly_audit_log_subscription` and `launchdarkly_destination` were never blocks** — they have always been maps (`config = { ... }`). Do not wrap them in `[ ]`.

6. **Order matters inside Sets in source, not in state.** Don't rely on Set element order for stability — but the rewrite itself is order-independent.

## Automated path: `scripts/migrate-tf-syntax`

A deterministic Go tool ships in this repo at `scripts/migrate-tf-syntax/`. **Use it first** — manual rewrites are only the fallback for edge cases the tool can't reach.

```bash
# migrate-tf-syntax is its OWN Go module — run it with `go -C` and an ABSOLUTE -dir.
# (`go run ./scripts/migrate-tf-syntax` from the repo root fails: "main module does not contain package".)
DIR="$PWD/local-testing/your-config"

# Forward (v2 → v3) on a directory of .tf files.
go -C scripts/migrate-tf-syntax run . -dir "$DIR" -direction v2-to-v3

# Inverse (v3 → v2) for rollback.
go -C scripts/migrate-tf-syntax run . -dir "$DIR" -direction v3-to-v2

# Dry-run to stdout.
go -C scripts/migrate-tf-syntax run . -dir "$DIR" -direction v2-to-v3 -dry-run

# Custom mapping file (for non-LD providers or future LD releases).
go -C scripts/migrate-tf-syntax run . -dir "$DIR" -direction v2-to-v3 -mappings "$PWD/my-spec.json"
```

The default `mappings.json` ships embedded and mirrors the mapping table below. Update both the JSON and this table when a new release adds nested-attribute resources.

Tool limits — must be patched manually after running:
- Synthesizes value-only `variations` for literal-`"boolean"` `launchdarkly_feature_flag`s (see gotcha §1); the provider preserves any out-of-band variation `name`/`description`. It does not synthesize for multivariate flags or non-literal `variation_type` — it warns on those.
- `v3-to-v2` mode appends reversed blocks at the end of each resource body — attribute ordering shifts. `terraform fmt` will normalize whitespace but not order.

After running the tool, run `terraform validate` to confirm parse-clean, then `terraform plan` for semantic regressions.

## Manual workflow (fallback)

1. **Identify failing files.** Run `terraform validate` (faster than `plan`, surfaces parse errors). The errors enumerate file:line per offending block.
2. **For each error**, look up the attribute in the mapping table to confirm it is a list-of-objects.
3. **Rewrite** using the patterns:
   - Single block → `name = [{ ... }]`
   - Multiple repeated blocks → `name = [{ ... }, { ... }, ...]`
   - Nested blocks → recurse: a `rules` block containing `clauses` blocks becomes `rules = [{ clauses = [{...}], ... }]`.
4. **Re-run `terraform validate`** after each file. Iterate until clean.
5. **Check for v3-required attributes** (currently only `variations` on `launchdarkly_feature_flag`) and add them if missing — the parse error is `Missing required argument`, not `Unsupported block type`.
6. **`terraform plan`** to surface any remaining schema-level issues (deprecation warnings are fine; new required attributes are not).

## Workflow — v3 → v2 downgrade

Inverse direction. Rare, but supported (e.g., rolling back the provider):

1. For each attribute in the mapping table, convert `name = [{ a, b, c }, { d, e, f }]` to repeated `name { a; b; c } name { d; e; f }` blocks.
2. Single-element max-1 lists collapse cleanly: `fallthrough = [{ variation = 0 }]` → `fallthrough { variation = 0 }`.
3. Drop now-unneeded `variations` on boolean flags if the v2 schema rejected them (it accepted them, so usually keep).
4. Validate against the v2 provider binary (`terraform init` may need `-upgrade` to pull the older constraint).

## Example: full conversion

v2 input:
```hcl
resource "launchdarkly_project" "main" {
  key  = "demo"
  name = "Demo"

  default_client_side_availability {
    using_environment_id = true
    using_mobile_key     = false
  }

  environments {
    key   = "production"
    name  = "Production"
    color = "EF4444"
  }

  environments {
    key   = "staging"
    name  = "Staging"
    color = "F59E0B"

    approval_settings {
      required          = true
      min_num_approvals = 1
    }
  }
}
```

v3 equivalent:
```hcl
resource "launchdarkly_project" "main" {
  key  = "demo"
  name = "Demo"

  default_client_side_availability = {
    using_environment_id = true
    using_mobile_key     = false
  }

  environments = {
    "production" = {
      key   = "production"
      name  = "Production"
      color = "EF4444"
    }
    "staging" = {
      key   = "staging"
      name  = "Staging"
      color = "F59E0B"

      approval_settings = [{
        required          = true
        min_num_approvals = 1
      }]
    }
  }
}
```

## Reference fixtures

The repo holds a complete v2-shaped scratch config and a converted v3 sibling that exercise every resource in the mapping table:

- `local-testing/full-account-v2.29.original/` and `local-testing/full-account-v2.29/` — both currently in v3 syntax (post-migration). Use either as a canonical v3 example.
- `local-testing/full-account-v2.29-backup-*.zip` — the genuine v2.29 **block-syntax** snapshot (the real "before" reference). Extract it; the on-disk dirs above are no longer block syntax.

When in doubt about a specific resource's exact attribute name or nesting, grep these directories first before reaching into the provider source.

## Verifying without breaking state

`terraform validate` only checks HCL parse + schema shape; it does not hit the API. Safe to run repeatedly without auth. Use it as the inner loop. Run `terraform plan` only once at the end to confirm no semantic regressions.

For state-affecting changes (the actual upgrade), use the `terraform-provider-local-testing` skill — it covers the dev-override flow, scratch configs, and state hygiene rules that keep `local-testing/` clean.

## When the table is wrong

This skill's mapping table is a snapshot. If a future LD provider release adds, removes, or renames any nested-attribute resource, the source of truth is:

```bash
grep -nE 'tfsdk:"(<attr_name>)"' launchdarkly/*.go
```

inside the `terraform-provider-launchdarkly` repo. A `types.List` / `types.Set` field paired with a `ListNestedAttribute` / `SetNestedAttribute` schema entry means list-of-objects → use `= [{...}]`. A `types.Object` field paired with a `SingleNestedAttribute` means single object → use `= {...}`. A `types.Map` field paired with a `MapNestedAttribute` means a key-addressed map → use `= { "<key>" = {...} }`. As of v3.0.0 GA the `types.Object` attributes are `client_side_availability`, `defaults`, `default_client_side_availability`, and `fallthrough`, and the only `types.Map` nested attribute is `launchdarkly_project.environments`; watch for more in later releases.
