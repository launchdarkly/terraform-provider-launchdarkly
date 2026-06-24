---
name: terraform-provider-block-to-nested-attrs
description: Migrate LaunchDarkly Terraform provider HCL configs between block syntax (provider v2.x and earlier) and nested-attribute syntax (v3.x+). Use when a user upgrades the launchdarkly provider to v3 and hits "Unsupported block type" / "Missing required argument" plan errors, when downgrading from v3 to v2.x, when porting a v2 example to v3 syntax (or vice versa), or when the user mentions "block to nested attribute", "= [{...}]", "v3 plan errors", or pastes errors that reference `inline_roles`, `statements`, `policy`, `policy_statements`, `environments`, `default_client_side_availability`, `rules`, `clauses`, `variations`, `client_side_availability`, `defaults`, `fallthrough`, `prerequisites`, `targets`, `context_targets`, `urls`, `role_attributes`, `messages`, `boolean_defaults`, `included_contexts`, `excluded_contexts`, `instructions`, `segments`, or `approval_settings`.
compatibility: Works on any directory containing `.tf` files that use `launchdarkly_*` resources. No external tools required beyond a working `terraform` CLI for validation.
metadata:
  author: ffeldberg
  version: "1.0.0"
---

# LaunchDarkly Terraform Provider: Block ↔ Nested Attribute Migration

The LaunchDarkly Terraform provider v3.0.0 finished the migration from `terraform-plugin-sdk/v2` to `terraform-plugin-framework`. Every former `schema.Block` is now a `*NestedAttribute` (`ListNestedAttribute` / `SetNestedAttribute`). HCL that worked against v2.x using block syntax (`name { ... }`) fails to parse against v3 with `Unsupported block type` or `Missing required argument`. The fix is mechanical: change `name { ... }` to `name = [{ ... }]`.

This skill enumerates every affected attribute, gives the exact rewrite, and lists the gotchas that bite during migration.

## When to use

- User runs `terraform plan` against v3 provider and gets `Unsupported block type` for any of the attributes in the mapping table below.
- User asks to "convert blocks to nested attributes" / "migrate config to v3" / "downgrade to v2 syntax".
- User pastes an HCL example using `name { ... }` and asks for the v3 equivalent (or vice versa).

## Do NOT use

- For provider source code changes (use `terraform-provider-add-resource` instead).
- For non-LaunchDarkly providers — the mappings here are specific to `launchdarkly_*` resources.

## Core rule

Every former block is now a **list-of-objects** (or set-of-objects). HCL syntax for both is `name = [{ ... }, { ... }]`. Even when the schema allows max 1 element (e.g. `fallthrough`, `default_client_side_availability`), it is still a list — wrap the single object in `[ ]`.

```hcl
# v2 (block)              # v3 (nested attribute)
foo {                     foo = [{
  bar = "baz"               bar = "baz"
}                         }]

foo { x = 1 }             foo = [
foo { x = 2 }               { x = 1 },
                            { x = 2 },
                          ]
```

To go v3 → v2, do the inverse: strip `= [` / `]`, change `},` separators to a new `foo {` per element.

## Mapping table

Every attribute that changed from block → nested attribute in v3. All use `= [{ ... }]` syntax. Underlying type column is informational — it does not change HCL syntax (List and Set both render as `[{...}]`).

| Resource | Attribute | Underlying type | Notes |
|---|---|---|---|
| `launchdarkly_access_token` | `inline_roles` | List | Replaces deprecated `policy_statements`. |
| `launchdarkly_access_token` | `policy_statements` | List | Deprecated; prefer `inline_roles`. |
| `launchdarkly_audit_log_subscription` | `statements` | List | Required. |
| `launchdarkly_webhook` | `statements` | List | Optional. |
| `launchdarkly_custom_role` | `policy` | Set | Deprecated; prefer `policy_statements`. |
| `launchdarkly_custom_role` | `policy_statements` | List | |
| `launchdarkly_relay_proxy_configuration` | `policy` | List | Required. |
| `launchdarkly_project` | `default_client_side_availability` | List (max 1) | |
| `launchdarkly_project` | `environments` | List | Required; min 1. |
| `launchdarkly_project.environments[*]` | `approval_settings` | List (max 1) | Nested inside each environment block. |
| `launchdarkly_environment` | `approval_settings` | List (max 1) | Same shape as inline-in-project. |
| `launchdarkly_segment` | `included_contexts` | List | |
| `launchdarkly_segment` | `excluded_contexts` | List | |
| `launchdarkly_segment` | `rules` | List | |
| `launchdarkly_segment.rules[*]` | `clauses` | List | Nested inside each rule. |
| `launchdarkly_flag_trigger` | `instructions` | List (max 1) | |
| `launchdarkly_view_links` | `segments` | Set | Wrap `environment_id` values in `nonsensitive()` to avoid set-hash churn — see `view.tf` in `local-testing/full-account-v2.29.original` for an example. |
| `launchdarkly_metric` | `urls` | List | |
| `launchdarkly_team` | `role_attributes` | Set | |
| `launchdarkly_team_member` | `role_attributes` | Set | |
| `launchdarkly_ai_config_variation` | `messages` | List | |
| `launchdarkly_flag_templates` | `boolean_defaults` | List (max 1) | |
| `launchdarkly_feature_flag_environment` | `prerequisites` | List | |
| `launchdarkly_feature_flag_environment` | `targets` | Set | |
| `launchdarkly_feature_flag_environment` | `context_targets` | Set | |
| `launchdarkly_feature_flag_environment` | `rules` | List | |
| `launchdarkly_feature_flag_environment.rules[*]` | `clauses` | List | Nested inside each rule. |
| `launchdarkly_feature_flag_environment` | `fallthrough` | List (max 1) | |
| `launchdarkly_feature_flag` | `client_side_availability` | List (max 1) | |
| `launchdarkly_feature_flag` | `variations` | List | **Required (min 1) in v3, including for boolean flags.** See gotcha §1. |
| `launchdarkly_feature_flag` | `defaults` | List (max 1) | |

If an attribute on a `launchdarkly_*` resource is not listed here, it was either already a primitive (no migration needed) or it was a `config` map (which uses `= { ... }` map syntax — not list-of-objects — and stayed the same across versions).

## Gotchas — v2 → v3

1. **`launchdarkly_feature_flag.variations` is now required for every flag, including booleans.** v2 inferred boolean variations; v3 requires them explicitly. Add:
   ```hcl
   variations = [
     { value = "true" },
     { value = "false" },
   ]
   ```
   `value` is a string — `"true"` / `"false"`, not the bare bool. The provider parses it per `variation_type`. The `migrate-tf-syntax` tool now synthesizes this automatically for flags whose `variation_type` is the literal `"boolean"` (value-only). Add variation `name`/`description` by hand only if they were set outside Terraform — the first apply rewrites them from config and clears any the config omits.

2. **`policy` on `launchdarkly_custom_role` and `policy_statements` on `launchdarkly_access_token` are deprecated** but still parse. Plan emits `Attribute Deprecated` warnings. Migrating to the newer name (`policy_statements` / `inline_roles`) is recommended but not required for the v2→v3 cutover itself.

3. **`launchdarkly_view_links.segments` uses set semantics.** If `environment_id` is sourced from a data source field marked `Sensitive` (e.g. `data.launchdarkly_environment.x.client_side_id`), the set hash will be unstable across plans. Wrap the value in `nonsensitive(...)` to stabilize the hash. Without this you get perpetual "segments updated" drift.

4. **Single-instance lists still need brackets.** `default_client_side_availability`, `fallthrough`, `defaults`, `boolean_defaults`, `client_side_availability`, `approval_settings`, and `instructions` are all max-1 lists. They still require `= [{ ... }]`, not `= { ... }`. A bare object map will fail with a type error.

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
- Synthesizes value-only `variations` for literal-`"boolean"` `launchdarkly_feature_flag`s (see gotcha §1), but not their `name`/`description`, and not for multivariate flags or non-literal `variation_type` — it warns on those.
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

  default_client_side_availability = [{
    using_environment_id = true
    using_mobile_key     = false
  }]

  environments = [
    {
      key   = "production"
      name  = "Production"
      color = "EF4444"
    },
    {
      key   = "staging"
      name  = "Staging"
      color = "F59E0B"

      approval_settings = [{
        required          = true
        min_num_approvals = 1
      }]
    },
  ]
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

inside the `terraform-provider-launchdarkly` repo. A `types.List` / `types.Set` field declaration paired with a `*NestedAttribute` schema entry means list-of-objects → use `= [{...}]`. A `types.Object` field means single object → use `= {...}` (none exist in v3.0.0 but watch for new ones in later releases).
