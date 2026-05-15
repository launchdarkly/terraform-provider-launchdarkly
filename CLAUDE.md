# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common commands

Go version pinned in `.go-version`: **1.25.8** (`scripts/codegen/go.mod` and parent `go.mod` are aligned at `go 1.25.8`). Provider package: `launchdarkly` (PKG_NAME in the makefile).

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

### Provider entrypoint

`main.go` serves a single terraform-plugin-framework provider on protocol v6 via `providerserver.Serve`. The provider implementation is `launchdarkly.NewPluginProvider(version)` in `launchdarkly/plugin_provider.go`, which registers every resource and data source. The CLI sees the provider address `registry.terraform.io/launchdarkly/launchdarkly`.

`launchdarkly/provider.go` is a constants-only file holding the env-var names and provider-schema attribute keys (`access_token`, `oauth_token`, `api_host`, `http_timeout`). There is no SDKv2 provider; the migration from `terraform-plugin-sdk/v2` finished in Phase 5 (see `.claude/MIGRATION_PLAN_NON_BREAKING.md` and `.claude/migration-archive/` for historical context).

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

### Resource / data-source conventions

- File layout: `resource_<name>_framework.go` + (optionally) `<name>_helper.go` (CRUD glue, type-id parsing, etc.) + `resource_launchdarkly_<name>_test.go`. Data sources mirror: `data_source_<name>_framework.go`.
- All terraform attribute names are `const` strings declared in `launchdarkly/keys.go` inside a `//gofmts:sort` block — **never inline a string literal for a schema key**. The constant name must equal its value. Adding a new key requires inserting it into that sorted block; `make fmt` will re-sort if you misplace it. Framework `tfsdk:` tags hold the wire identifier; the Go-side constants give cross-file consistency.
- Patches use `patchReplace` / `patchAdd` / `patchRemove` helpers (in `launchdarkly/helper.go`) wrapping `ldapi.PatchOperation`; prefer these over hand-rolled structs.
- `handleLdapiErr` unwraps `*ldapi.GenericOpenAPIError` to surface the response body — wrap raw API errors before returning to Terraform.
- All nested-object structures are framework `*NestedAttribute` (`ListNestedAttribute` / `SetNestedAttribute`). The provider has **zero** `schema.Blocks` declarations as of v3.0.0 — the block→nested-attribute migration completed alongside the SDKv2→framework cutover. User HCL uses `name = { ... }` / `name = [{ ... }, ...]` syntax everywhere; legacy `name { ... }` block syntax does not parse against current schemas.

Shared framework utility surface:

- `launchdarkly/framework_helpers.go` — `*Client` extraction, set/list conversions, optional-attr helpers (e.g. `stringValueOrNullFromPointer`, `setFromStringSliceOrNull`).
- `launchdarkly/framework_validators.go` — `keyValidator`, `idValidator`, `tagValidator`, `opValidator`, `keyAndLengthValidator`.
- `launchdarkly/framework_json_helpers.go` — JSON validators / plan modifiers (`jsonStringValidator`, `jsonNormalizePlanModifier`).
- `launchdarkly/framework_schema_compat.go` — Crossplane-Upjet defensive shim; see Upjet section below.
- `launchdarkly/statecompat/` — wire-compat regression harness (`statecompat.Run`). Fixtures live under `launchdarkly/testdata/state-fixtures/`; capture flow under `scripts/capture-state-fixtures/`.

### Crossplane / Upjet embedded-schema compatibility (`launchdarkly/framework_schema_compat.go`)

This provider is embedded by Crossplane's Upjet, which historically strips deprecated attributes from the runtime schema. With SDKv2 this produced two error shapes that needed swallowing on writes to those attributes; the framework analogue lives in `framework_schema_compat.go` as `isOmittedFrameworkAttrDiag` + helpers, matching framework's `AttributeError` diagnostic shape.

Use the helpers only on attributes that may be stripped by an embedder (typically `Deprecated:` ones). Matchers are intentionally narrow so unrelated errors still surface — don't broaden them without rationale. Background: `.claude/migration-archive/schema-compat-upjet.md`.

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
