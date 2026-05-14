# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common commands

Go version pinned in `.go-version`: **1.25.8** (bumped from 1.25.5 in Phase 2 after the `terraform-plugin-sdk/v2` v2.40.1 upgrade required it; `scripts/codegen/go.mod` and parent `go.mod` are aligned at `go 1.25.8`). Provider package: `launchdarkly` (PKG_NAME in the makefile).

- `make build` — `fmtcheck` then `go install` with `-ldflags` injecting the short git rev as `version`. Binary lands in `$GOPATH/bin`.
- `make fmt` — runs `gofmts -w` (the constant-sorting linter from `github.com/ashanbrown/gofmts`, *not* `gofumpt`) **then** `gofmt -w`. Order matters: `gofmts` rewrites `//gofmts:sort` blocks (see `launchdarkly/keys.go`) and the second pass normalizes the result.
- `make fmtcheck` — invoked by `build`/`test`. Auto-installs `gofmts@v0.2.0` if missing.
- `make test` — unit tests, `-timeout=90s -parallel=4`, xargs in chunks of 4 packages.
- `make testacc` — full acceptance suite. Sets `TF_ACC=1`. Hits **real** LaunchDarkly via `LAUNCHDARKLY_ACCESS_TOKEN`; requires enterprise account. 120m timeout.
- `make testacc-with-retry` — runs `testacc` once, retries once on failure. CI uses this.
- `TESTARGS="-run TestAccProject_Create" make testacc` — single acceptance test. CI matrix entries (e.g. `TestAccFeatureFlag_`) are `go test -run` prefixes, not suite names.
- `make generate` — installs the `codegen` tool, then runs `go generate ./launchdarkly/... .`. Regenerates `launchdarkly/integration_configs_generated.go` from `https://app.launchdarkly.com/api/v2/integration-manifests` **and** rebuilds `docs/` via `tfplugindocs`. Requires `LAUNCHDARKLY_ACCESS_TOKEN` and a real `terraform` binary on PATH (no wrapper).
- `make vet` / `make errcheck` — also wired into CI.

CI gate at `.github/workflows/test.yml`: build + lint (`golangci-lint v1.64.8`) + `make generate` diff check + matrix of `TestAcc*` prefixes. The `generate` and `test` jobs are **skipped on fork PRs** — a maintainer must add the `safe-to-test` label to trigger `test-fork-pr.yml`. New pushes strip the label (see `remove-safe-to-test-label.yml`), forcing re-review.

## Local dev override

`make build` installs into `$GOPATH/bin`; point a `~/.terraformrc` `dev_overrides` block at that directory to consume the local binary. `terraform init` is skipped under dev overrides — re-run `make build` between provider changes. `local-testing/` holds scratch `.tf` files; treat its `terraform.tfstate*` files as throwaway, **never commit them**.

## Architecture

### Mux: two providers in one binary

`main.go` serves a `tf5muxserver` combining:

1. **`Provider()` in `launchdarkly/provider.go`** — terraform-plugin-sdk **v2** (the legacy SDK). Hosts the bulk of resources/data sources via the `ResourcesMap` / `DataSourcesMap` registration pattern.
2. **`NewPluginProvider(version)` in `launchdarkly/plugin_provider.go`** — terraform-plugin-**framework**. Currently hosts only `launchdarkly_team_role_mapping` (`resource_team_role_mapping.go`). New resources that need the framework's richer plan-modifier / nested-attribute model go here.

Both share the same `Client` and the same provider schema attributes (`access_token`, `oauth_token`, `api_host`, `http_timeout`); the framework version reads descriptions from `providerSchema()` to keep them aligned. The CLI sees a single provider address `registry.terraform.io/launchdarkly/launchdarkly`.

When adding a resource, decide SDKv2 vs framework based on schema complexity; do **not** register the same resource on both.

### Client (`launchdarkly/config.go`)

`newClient` builds **two** `ldapi.APIClient` instances against `github.com/launchdarkly/api-client-go/v22`:

- `client.ld` — standard retry policy.
- `client.ld404Retry` — *also* retries 404s with exponential backoff. Use **sparingly** (a 404 normally means deletion). Tracked under `sc-218015`.

Other invariants:
- `APIVersion = "20240415"` is sent as `LD-API-Version` on every request *except* when `apiVersion == "beta"` (the generated client passes the header per-request for beta endpoints; setting a default would duplicate it).
- `DEFAULT_MAX_CONCURRENCY = 1`. The client wraps API calls in `c.withConcurrency(...)` using a `semaphore.Weighted`. Tests can pass a higher concurrency via `baseNewClient`.
- 429 backoff respects `X-RateLimit-Reset` (millis since epoch). Negative durations are flipped + jittered — this is intentional, LD sometimes returns headers that compute to negatives.
- `newBetaClient` is the same constructor with `apiVersion = "beta"` — use it for endpoints not yet in the stable API surface (e.g. views).
- OAuth and personal/service tokens both use the `ApiKey` header in v22; the `oauth` flag is currently a no-op for header construction (kept for clarity).

### Resource conventions (SDKv2)

- File layout: `resource_launchdarkly_<name>.go` + `<name>_helper.go` (CRUD glue) + `resource_launchdarkly_<name>_test.go`. Data sources mirror: `data_source_launchdarkly_<name>.go`.
- All terraform attribute names are `const` strings declared in `launchdarkly/keys.go` inside a `//gofmts:sort` block — **never inline a string literal for a schema key**. The constant name must equal its value. Adding a new key requires inserting it into that sorted block; `make fmt` will re-sort if you misplace it. (Phase 5.3 decision: `keys.go` is retained post-cutover. Framework `tfsdk:` tags hold the wire identifier; the Go-side constants give cross-file consistency. Don't introduce string-literal schema keys in new framework code either.)
- `removeInvalidFieldsForDataSource` in `helper.go` strips `Default`, validation, diff suppression etc. from a schema map so the same nested schema can be reused for data sources (Terraform forbids those on computed-only attrs).
- Patches use `patchReplace` / `patchAdd` / `patchRemove` helpers wrapping `ldapi.PatchOperation`; prefer these over hand-rolled structs.
- `handleLdapiErr` unwraps `*ldapi.GenericOpenAPIError` to surface the response body — wrap raw API errors before returning to Terraform.

### Framework schema conventions for SDKv2-parity migrations

The provider is incrementally migrating from `terraform-plugin-sdk/v2` to `terraform-plugin-framework` per `.claude/MIGRATION_PLAN_NON_BREAKING.md`. Sub-phase branches use the naming convention `moonshots/tpf/<phase-id>-<slug>` (e.g. `moonshots/tpf/2.3-ai-tool`) and PR back into the long-lived `moonshots/terraform-plugin-framework` integration branch.

Rules for resources / data sources migrated from SDKv2:

- **Use `schema.Schema.Blocks`** (with `ListNestedBlock`, `SetNestedBlock`, `SingleNestedBlock`) for any SDKv2 nested structure that was a block in HCL. Nested attributes (`schema.ListNestedAttribute`, etc.) are a different user-facing config surface — switching breaks existing configs.
- Worked example + SDKv2 → framework cheatsheet: `launchdarkly/framework_schema_reference.go`.
- **Preserve every `Required`/`Optional`/`Computed`/`ForceNew`/`Default`/`ConflictsWith`/`ExactlyOneOf`/`Deprecated:` flag verbatim.** ForceNew becomes `stringplanmodifier.RequiresReplace()` (or the type-specific equivalent). Defaults become `<type>default.StaticX(...)`. Deprecated → `DeprecationMessage:` carries the SDKv2 string verbatim — see `docs/v2-deprecations-carryforward.md`.
- **Don't bump `SchemaVersion`** during a migration PR — the wire format stays put. If the SDKv2 resource has `StateUpgraders`, port the chain to framework `UpgradeState` returning the same shape transformations.
- **State-compat fixture required** for every Phase 2-4 PR. Drop the fixture under `launchdarkly/testdata/state-fixtures/`, captured via `scripts/capture-state-fixtures/capture.sh`, and exercise it from `launchdarkly/statecompat/`. See `scripts/capture-state-fixtures/README.md`.

Shared utility surface (Phase 0 deliverables):

- `launchdarkly/framework_helpers.go` — `*Client` extraction, set/list conversions, optional-attr helpers.
- `launchdarkly/framework_validators.go` — `keyValidator`, `idValidator`, `tagValidator`, `opValidator`, `keyAndLengthValidator`.
- `launchdarkly/framework_schema_compat.go` — Upjet runtime-schema-stripping shim for deprecated attrs (decision in `docs/migration-schema-compat-upjet.md`).
- `launchdarkly/statecompat/` — wire-compat regression harness (`statecompat.Run`).

**Block-style schemas for resources migrated from SDKv2; nested attributes only for genuinely new resources.** Drift from this convention in a migration PR is a breaking change disguised as internal cleanup — flag it in review.

### Upjet / embedded-schema compatibility (`launchdarkly/schema_compat.go`)

The provider is embedded by Crossplane's Upjet, which sometimes **strips deprecated attributes from the runtime schema**. Reads/writes against those keys then fail with very specific SDK errors:

- `Invalid address to set: []string{"<attr>"}` (from `ResourceData.Set`)
- `: invalid key: <attr>` (from `ResourceDiff.SetNew`)

`resourceDataSetSkipMissingKey` and `resourceDiffSetNewSkipMissingKey` are the only sanctioned way to swallow these. **Don't** add generic "ignore error" wrappers around `d.Set` — the matchers in `isOmittedEmbeddedSchemaAttrErr` are intentionally narrow so unrelated errors surface. When you remove or deprecate an attribute, route writes through these helpers instead of deleting the references, so embedded users don't break.

### Codegen for audit log subscriptions

`launchdarkly/integration_configs_generated.go` is generated from LaunchDarkly's integration-manifests API by `scripts/codegen/`. Touch the generator (in `scripts/codegen/manifestgen/`) — not the generated file. CI fails if `make generate` produces a diff that isn't committed. If you change `launchdarkly_audit_log_subscription`'s config field mapping you almost certainly need to run `make generate` and commit the result.

### Docs

`docs/` is generated by `tfplugindocs` from `templates/` + schema descriptions. Edit `templates/` (or schema `Description` fields) — **not** `docs/*.md` — then `make generate`.

## Skills to use

- **`terraform-provider-add-resource`** (LD internal) — full Steps 0-10 checklist for adding a new resource. Use whenever scaffolding a new `launchdarkly_*` resource or data source; it covers schema, CRUD, helpers, docs, tests, and registration.
- **`terraform-provider-local-testing`** (LD internal) — playbook for exercising the locally built provider against real LD (dev override flow, `local-testing/` scratch configs, state hygiene). Use before claiming a change works end-to-end.
- **`terraform-skill`** (https://github.com/antonbabenko/terraform-skill) — general Terraform development/review skill. Useful for HCL authorship, provider schema design review, plan/apply diagnostics, and second-opinion review on resource ergonomics.

## Releases

`release-please-config.json` drives a draft release-PR workflow (`release-type: go`, tags include `v`). `CHANGELOG.md` and `.release-please-manifest.json` are owned by the bot; don't hand-edit. Conventional Commit prefixes (`feat:`, `fix:`, `chore:`, etc.) determine the bump. `lint-pr-title.yml` enforces the format on PRs.
