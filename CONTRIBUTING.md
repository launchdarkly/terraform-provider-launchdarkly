# Contributing

Thanks for your interest in contributing to the LaunchDarkly Terraform provider.

## Quick links

- **Architecture, build / test commands, and conventions**: see [`CLAUDE.md`](./CLAUDE.md) at the repository root. That file is the day-to-day source of truth.
- **Internal runbook (LaunchDarkly employees)**: linked from `.github/pull_request_template.md`.

## Adding a new resource

Resources and data sources are built on [`terraform-plugin-framework`](https://github.com/hashicorp/terraform-plugin-framework). Register new ones in `launchdarkly/plugin_provider.go` (`Resources()` / `DataSources()`).

The shared utility surface is:

- [`launchdarkly/framework_helpers.go`](./launchdarkly/framework_helpers.go) — `*Client` extraction, set/list conversions, optional-attr helpers.
- [`launchdarkly/framework_validators.go`](./launchdarkly/framework_validators.go) — `keyValidator`, `idValidator`, `tagValidator`, `opValidator`, etc.
- [`launchdarkly/framework_json_helpers.go`](./launchdarkly/framework_json_helpers.go) — JSON validators / plan modifiers for stringified-JSON attributes.
- [`launchdarkly/framework_schema_compat.go`](./launchdarkly/framework_schema_compat.go) — Crossplane-Upjet runtime-schema-stripping shim. Use only on attributes that may be stripped by an embedder (typically `Deprecated:` ones).
- [`launchdarkly/statecompat/`](./launchdarkly/statecompat/) — wire-compat regression harness. See [`scripts/capture-state-fixtures/README.md`](./scripts/capture-state-fixtures/README.md) for the fixture workflow if you need to assert state-shape invariants across releases.

Conventions: attribute names are constants in `launchdarkly/keys.go` (`//gofmts:sort` block — never inline a string literal). Match existing patterns in `resource_*_framework.go` files; data-sources mirror as `data_source_*_framework.go`. The internal `terraform-provider-add-resource` skill encodes the full step-by-step checklist (schema, CRUD, helpers, docs, tests, registration).

## Build / test / generate

```bash
make build          # fmtcheck + go install into $GOPATH/bin
make fmt            # gofmts (constant-sort) + gofmt
make fmtcheck       # CI-friendly fmt verification
make test           # unit tests, chunked via xargs
make testacc        # full acceptance suite (TF_ACC=1, real LD account required)
make generate       # regenerate docs/ + integration_configs_generated.go
```

`make generate` requires `LAUNCHDARKLY_ACCESS_TOKEN` and a real `terraform` binary on `PATH`. CI fails if `make generate` produces a diff that isn't committed.

## Local dev override

`make build` installs into `$GOPATH/bin`. Point a `~/.terraformrc` `dev_overrides` block at that directory to consume the local binary against scratch configs in `local-testing/`. The internal `terraform-provider-local-testing` skill captures the full workflow.
