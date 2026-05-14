# v2.37.0 — Internal SDK cutover (release notes draft)

## Summary

Internal-only: the provider implementation completes its migration from
`terraform-plugin-sdk/v2` to `terraform-plugin-framework`. No HCL or
`.tfstate` changes required by users. Schema shape, attribute names,
defaults, and deprecation messages preserved verbatim from v2.36.

## ⚠️ Minimum Terraform requirement

This release **raises the minimum supported Terraform CLI to `>= 1.0`**.
The provider now serves Terraform plugin protocol version 6 exclusively;
earlier CLI versions (TF 0.13 – 0.15) cannot negotiate v6 and will fail
at `terraform init`.

If you are on TF 0.15 or earlier, upgrade to Terraform CLI `>= 1.0`
before upgrading this provider. Stay on v2.36.x if you cannot upgrade
the CLI.

## Internal changes

- Provider implementation fully migrated to
  `github.com/hashicorp/terraform-plugin-framework`.
- `github.com/hashicorp/terraform-plugin-sdk/v2` dropped as a direct
  require in `go.mod`. (Retained as an indirect dependency because the
  acceptance-test harness pulls it transitively; the released binary
  does not link it.)
- `github.com/hashicorp/terraform-plugin-mux` removed entirely — there
  is no longer a mux server in `main.go`.
- Wire protocol flipped from version 5 → version 6.

## Compatibility

| Surface | Status |
|---|---|
| Existing HCL configs | Apply unchanged |
| Existing `.tfstate` files | Apply unchanged, zero plan diff |
| Resource type names | Unchanged |
| Import IDs | Unchanged |
| Schema attribute names | Unchanged |
| Deprecated attribute carry-forward | Functional, still deprecated |

## Upgrading

```bash
# bump Terraform CLI if you are below 1.0
brew upgrade terraform   # or your equivalent install path

# bump the provider
terraform init -upgrade
terraform plan   # expected: no changes
```

If `terraform plan` shows a non-empty diff against an unchanged v2.36
state, please file an issue with the resource type and a redacted
state-fixture snippet so we can chase the regression.

## Downstream consumers

- **Crossplane Upjet**: the framework-side
  `framework_schema_compat.go` shim absorbs the deprecated-attribute
  stripping pattern Upjet performs against the runtime schema.
- **terraform-modules**: no module-side changes required.
