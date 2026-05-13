# Contributing

Thanks for your interest in contributing to the LaunchDarkly Terraform provider.

## Quick links

- **Architecture, build / test commands, and conventions**: see [`CLAUDE.md`](./CLAUDE.md) at the repository root. That file is the day-to-day source of truth.
- **Internal runbook (LaunchDarkly employees)**: linked from `.github/pull_request_template.md`.
- **Migration plan (SDKv2 -> terraform-plugin-framework)**: [`.claude/MIGRATION_PLAN_NON_BREAKING.md`](./.claude/MIGRATION_PLAN_NON_BREAKING.md). Per-phase execution plans live under [`.claude/plans/migration/`](./.claude/plans/migration/).

## Adding a new resource

New resources go on the **terraform-plugin-framework** code path (not SDKv2 — the SDKv2 surface is being retired). Use the SDKv2 -> framework convention documented in [`CLAUDE.md`](./CLAUDE.md) under "Framework schema conventions for SDKv2-parity migrations" — the cheat sheet in [`launchdarkly/framework_schema_reference.go`](./launchdarkly/framework_schema_reference.go) maps every SDKv2 schema concept to its framework equivalent.

The shared utility surface for framework resources is:

- [`launchdarkly/framework_helpers.go`](./launchdarkly/framework_helpers.go) — `*Client` extraction, set/list conversions, optional-attr helpers.
- [`launchdarkly/framework_validators.go`](./launchdarkly/framework_validators.go) — `keyValidator`, `idValidator`, `tagValidator`, `opValidator`, etc.
- [`launchdarkly/framework_schema_compat.go`](./launchdarkly/framework_schema_compat.go) — Upjet runtime-schema-stripping shim. Use only on deprecated attributes; rationale in [`docs/migration-schema-compat-upjet.md`](./docs/migration-schema-compat-upjet.md).
- [`launchdarkly/statecompat/`](./launchdarkly/statecompat/) — wire-compat regression harness for migration PRs. See [`scripts/capture-state-fixtures/README.md`](./scripts/capture-state-fixtures/README.md) for the fixture workflow.

The internal `terraform-provider-add-resource` skill encodes the full Steps 0-10 checklist (covers schema, CRUD, helpers, docs, tests, registration). The skill is being refreshed for framework conventions — track that work separately.

## Migration PRs (SDKv2 -> framework)

If your PR migrates an existing resource or data source from `terraform-plugin-sdk/v2` to `terraform-plugin-framework`, use the migration checklist in `.github/pull_request_template.md`. Key gates:

1. **Base branch is `moonshots/terraform-plugin-framework`** — not `main`. Phase batches promote to `main` once the integration branch has soaked the cumulative change.
2. **Branch name follows `moonshots/tpf/<phase-id>-<slug>`** (e.g. `moonshots/tpf/2.3-ai-tool`).
3. **Block-style schemas (`schema.Blocks`)** for any structure that was a block in SDKv2. No block-to-nested-attribute conversion.
4. **State-compat fixture** committed under `launchdarkly/testdata/state-fixtures/` and exercised from `launchdarkly/statecompat/`. The harness asserts a v2.29-applied state file produces zero plan diff after the migration.
5. **No `SchemaVersion` bump** — the wire format stays put. Existing `StateUpgraders` port to framework `UpgradeState` with identical shape transformations.
6. **Conventional Commit prefix `refactor:`** — migrations are non-breaking by construction. Never `feat!:` or `BREAKING CHANGE:`.

See [`MIGRATION_PLAN_NON_BREAKING.md`](./.claude/MIGRATION_PLAN_NON_BREAKING.md) for the full per-PR checklist and phase sequencing.

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
