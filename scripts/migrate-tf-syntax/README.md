# migrate-tf-syntax

Deterministic converter between block syntax (`name { ... }`) and list-of-objects nested-attribute syntax (`name = [{ ... }]`) for Terraform HCL. Built for the LaunchDarkly Terraform provider v2 → v3 cutover but driven by an external JSON mapping file, so it works for any provider that did the same `terraform-plugin-sdk/v2` → `terraform-plugin-framework` block-to-attribute migration.

## Usage

```bash
# v2 → v3: turn `name { ... }` blocks into `name = [{ ... }]` attributes.
go run . -dir ../../local-testing/full-account-v2.29.original -direction v2-to-v3

# v3 → v2: best-effort inverse (see "Caveats" below).
go run . -dir ./configs -direction v3-to-v2

# Preview without writing.
go run . -dir ./configs -direction v2-to-v3 -dry-run

# Convert local modules in the same pass (walks subdirectories, skips .terraform and .git).
go run . -dir ./configs -direction v2-to-v3 -recursive

# Custom mappings for another provider.
go run . -dir ./configs -direction v2-to-v3 -mappings ./aws-spec.json
```

Build a standalone binary:

```bash
go build -o ../../bin/migrate-tf-syntax .
```

## Mapping file format

`mappings.json` (embedded as default) maps resource type → object containing three optional sections:

- `blocks` — attributes that switched from block to list-of-objects nested attribute. Nested entries describe attributes inside an element that themselves migrated (e.g. `rules` contains `clauses`).
- `deprecations` — attributes removed from the v3 schema. Each entry has `name`, `action`, and (for some actions) `to`. Supported actions:
  - `drop` — remove the attribute outright (no replacement).
  - `rename` — move the value verbatim onto `to` (e.g. `policy_statements` → `inline_roles`). If `to` already exists in the config, the existing value wins and the deprecated attribute is dropped.
  - `iis_to_csa` — rewrite `include_in_snippet` into a `client_side_availability`-shaped nested attribute on `to`, preserving the original expression as `using_environment_id` and synthesizing `using_mobile_key` (`using_mobile_key` in the mapping overrides the default `false`).
  - `policy_to_policy_statements` — move the custom-role `policy` list onto `to`; the inner attribute names are identical so elements transfer verbatim.
- `ds_attr_rewrites` — data-source output attributes renamed in v3. The script rewrites every `data.<type>.<name>.<from>` reference across all files (data-source outputs are computed-only, so references are the only thing to migrate). `to` renames the terminal attribute; `to_expr` replaces it with a structurally different access path.

```json
{
  "launchdarkly_segment": {
    "blocks": [
      { "name": "included_contexts" },
      { "name": "rules", "nested": [{ "name": "clauses" }] }
    ]
  },
  "launchdarkly_metric": {
    "blocks": [{ "name": "urls" }],
    "deprecations": [{ "name": "is_active", "action": "drop" }]
  },
  "launchdarkly_project": {
    "ds_attr_rewrites": [{ "from": "client_side_availability", "to": "default_client_side_availability" }]
  }
}
```

Deprecation operations are one-way (v2→v3 only). The reverse direction reinstates block syntax but cannot re-create attributes the v3 schema no longer accepts.

The embedded default ships every attribute touched by LD provider v3.0.0. Pass `-mappings path.json` to override.

## Caveats

1. **`v3-to-v2` reorders attributes.** Reverse-converted blocks are appended at the end of the parent body; the original attribute position is lost. `terraform validate` still passes; `terraform fmt` cleans whitespace; diff readability suffers.
2. **No semantic upgrades.** The script converts syntax only. The v3 schema introduces new required attributes (notably `variations` on `launchdarkly_feature_flag` for every variation_type, including boolean). Add those by hand — the script does not synthesize values.
3. **Local modules need `-recursive`; remote modules are out of scope.** The default is the historical single-directory glob. `-recursive` walks subdirectories (skipping `.terraform` and `.git`) so locally vendored modules convert in the same pass. Registry- or git-sourced modules can't be rewritten from the consumer side — upgrade those modules at their source.
4. **`dynamic` blocks are not converted.** A `dynamic "variations" { ... }` generator needs a for expression (`variations = [for ... : { ... }]`) that only the author can write. The script warns with the file and resource address and skips the whole attribute — including static sibling blocks of the same name, since converting only those would leave an attribute and a dynamic block for the same name, which v3 rejects.
5. **`*.tfvars` and `terraform.tfstate` untouched.** State migration is a separate problem; rerun `terraform apply` after upgrading the provider.
6. **Comments preserved best-effort.** Block-internal comments survive forward conversion (they ride along in the body's token stream). Reverse conversion may shuffle them if they straddle attribute boundaries.

## When the mapping is wrong

If a future provider release adds or renames a nested attribute, regenerate the mapping by grepping the framework schema definitions. For LaunchDarkly:

```bash
grep -nE 'tfsdk:"[^"]+"' launchdarkly/resource_*_framework.go \
  | grep -E 'types\.(List|Set)\b'
```

A `types.List` / `types.Set` paired with a `*NestedAttribute` schema entry means list-of-objects; add it to `mappings.json`.

## Round-trip test

```bash
# Start from v3 source, downgrade, upgrade, expect identical (modulo formatting).
cp -r ./configs /tmp/rt && \
  go run . -dir /tmp/rt -direction v3-to-v2 && \
  go run . -dir /tmp/rt -direction v2-to-v3 && \
  terraform -chdir=/tmp/rt validate
```
