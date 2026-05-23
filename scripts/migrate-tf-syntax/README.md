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

# Custom mappings for another provider.
go run . -dir ./configs -direction v2-to-v3 -mappings ./aws-spec.json
```

Build a standalone binary:

```bash
go build -o ../../bin/migrate-tf-syntax .
```

## Mapping file format

`mappings.json` (embedded as default) maps resource type → list of attributes that switched from block to list-of-objects. Nested entries describe attributes inside an element that themselves migrated (e.g. `rules` contains `clauses`).

```json
{
  "launchdarkly_segment": [
    { "name": "included_contexts" },
    { "name": "rules", "nested": [{ "name": "clauses" }] }
  ]
}
```

The embedded default ships every attribute touched by LD provider v3.0.0. Pass `-mappings path.json` to override.

## Caveats

1. **`v3-to-v2` reorders attributes.** Reverse-converted blocks are appended at the end of the parent body; the original attribute position is lost. `terraform validate` still passes; `terraform fmt` cleans whitespace; diff readability suffers.
2. **No semantic upgrades.** The script converts syntax only. The v3 schema introduces new required attributes (notably `variations` on `launchdarkly_feature_flag` for every variation_type, including boolean). Add those by hand — the script does not synthesize values.
3. **Files only, not modules.** Operates on `*.tf` files in a single directory. No recursion. Module composition + `for_each` blocks unaffected (they aren't block syntax in the first place).
4. **`*.tfvars` and `terraform.tfstate` untouched.** State migration is a separate problem; rerun `terraform apply` after upgrading the provider.
5. **Comments preserved best-effort.** Block-internal comments survive forward conversion (they ride along in the body's token stream). Reverse conversion may shuffle them if they straddle attribute boundaries.

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
