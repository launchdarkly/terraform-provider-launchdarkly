# AGENTS.md

Guidance for AI coding agents (Claude, Cursor, Copilot, Codex, etc.) working in this repository. Humans should also find this useful as a quick orientation; for the full developer setup see [DEVELOPMENT.md](./DEVELOPMENT.md).

## What this repo is

The official [Terraform](https://www.terraform.io) provider for [LaunchDarkly](https://launchdarkly.com). It exposes LaunchDarkly resources (projects, environments, feature flags, segments, teams, AI configs, integrations, etc.) as Terraform resources and data sources, talking to the LaunchDarkly REST API via [`launchdarkly/api-client-go/v22`](https://github.com/launchdarkly/api-client-go).

- **Language:** Go (see `.go-version` for the exact version).
- **Module:** `github.com/launchdarkly/terraform-provider-launchdarkly`.
- **Published to:** the Terraform Registry as `launchdarkly/launchdarkly`.

## Architecture at a glance

The provider is a **muxed dual-provider** assembled in [`main.go`](./main.go):

```go
providers := []func() tfprotov5.ProviderServer{
    launchdarkly.Provider().GRPCProvider,                            // SDKv2 (legacy)
    providerserver.NewProtocol5(launchdarkly.NewPluginProvider(version)()), // plugin-framework (modern)
}
muxServer, _ := tf5muxserver.NewMuxServer(ctx, providers...)
```

- Most resources use [`hashicorp/terraform-plugin-sdk/v2`](https://github.com/hashicorp/terraform-plugin-sdk) — registered in [`launchdarkly/provider.go`](./launchdarkly/provider.go).
- Newer resources use [`hashicorp/terraform-plugin-framework`](https://github.com/hashicorp/terraform-plugin-framework) — registered in [`launchdarkly/plugin_provider.go`](./launchdarkly/plugin_provider.go). This is the path forward; new resources should generally go here unless there's a strong reason otherwise.
- Both providers share the same configuration block (`access_token` / `oauth_token` / `api_host` / `http_timeout`) and the same internal `*Client`.

## Repo layout

```
main.go                          # Entry point, mux of SDKv2 + plugin-framework providers
launchdarkly/                    # All provider source code (flat package)
  provider.go                    #   SDKv2 provider + resource/data-source registry
  plugin_provider.go             #   plugin-framework provider + resource registry
  config.go                      #   *Client, retry/backoff, rate-limit + concurrency control
  keys.go                        #   String constants for every schema attribute name
  resource_launchdarkly_*.go     #   CRUD per LaunchDarkly entity
  data_source_launchdarkly_*.go  #   Data sources per entity
  *_helper.go                    #   Schema + API<->TF translation helpers per concept
  *_test.go                      #   Unit + acceptance tests (acceptance gated by TF_ACC=1)
  integration_configs_generated.go # GENERATED — do not hand-edit
  tests/                         #   Acceptance tests for plugin-framework resources
docs/                            # GENERATED Terraform Registry docs (via tfplugindocs)
templates/                       # Templates consumed by tfplugindocs
examples/                        # HCL examples used during doc generation
usage-examples/                  # User-facing examples
scripts/
  codegen/                       # Generator for integration_configs_generated.go
  *.sh                           # gofmt/errcheck checks etc.
tools/tools.go                   # Go tool dependency tracking
GNUmakefile                      # build / test / testacc / generate / fmt / vet / errcheck
.goreleaser.yml                  # Cross-platform release builds
release-please-config.json       # Automated changelog + version bumps
```

## Conventions to follow

When making changes, match the existing patterns rather than introducing new ones.

1. **Use the constants in `keys.go`** for every schema attribute name. If you need a new attribute, add a constant there (the file is alphabetised; a `//gofmts:sort` directive enforces this) and reference it everywhere — never inline raw strings like `"variation"`.
2. **File naming.** For a new LaunchDarkly entity `foo`, create:
   - `resource_launchdarkly_foo.go` (+ `_test.go`)
   - `data_source_launchdarkly_foo.go` (+ `_test.go`) if a data source makes sense
   - `foo_helper.go` (+ `_test.go`) for shared schema/translation logic
3. **Register new resources/data sources** in `launchdarkly/provider.go` (SDKv2) or `launchdarkly/plugin_provider.go` (plugin-framework). Don't forget both the resource/data-source map *and* the resource/data-source factory function.
4. **API calls go through the shared `*Client`** in `config.go`. Wrap calls in `client.withConcurrency(ctx, func() error { ... })` so we respect the provider-wide concurrency semaphore and don't trip LaunchDarkly's rate limits.
5. **404 handling.** There are two API clients on `*Client`:
   - `client.ld` — standard, treats 404 as terminal.
   - `client.ld404Retry` — retries 404s with exponential backoff. Use **only** when eventual consistency makes a transient 404 expected (see comments referencing `sc-218015`).
6. **Don't hand-edit generated files.** `launchdarkly/integration_configs_generated.go` is regenerated by `make generate` from upstream [`launchdarkly/integration-framework`](https://github.com/launchdarkly/integration-framework). Likewise `docs/` is regenerated by `tfplugindocs` from `templates/` + `examples/`.
7. **Examples and templates drive the docs.** When you add or change a resource, add/update the corresponding HCL example under `examples/` and (if needed) the template under `templates/`, then run `make generate` to regenerate `docs/`.
8. **Formatting/linting.** `gofmt`, `gofmts` (sort directives), `go vet`, and `errcheck` are all enforced. Run `make fmt && make vet && make errcheck` before committing.

## Build, test, generate

From [`GNUmakefile`](./GNUmakefile):

| Command | What it does |
|---|---|
| `make build` | Compiles the provider into `$GOPATH/bin` with the version embedded via `-ldflags`. |
| `make test` | Unit tests (`go test`, `parallel=4`). Safe to run anywhere. |
| `make testacc` | **Acceptance tests** — sets `TF_ACC=1` and hits the real LaunchDarkly API. Requires `LAUNCHDARKLY_ACCESS_TOKEN` (and optionally `LAUNCHDARKLY_API_HOST`) plus an enterprise account. **Don't run blindly from an agent loop** — these create and destroy real resources. |
| `make generate` | Regenerates `integration_configs_generated.go` (via `scripts/codegen`) and the registry docs (via `tfplugindocs`). Run after upstream integration-framework changes or after editing examples/templates. |
| `make fmt` / `make fmtcheck` | Run `gofmts` + `gofmt`. `fmtcheck` is also enforced by pre-commit and CI. |
| `make vet` / `make errcheck` | Static analysis. |

To use a locally built provider, add a `dev_overrides` block to `~/.terraformrc` pointing at `$GOPATH/bin` — see [DEVELOPMENT.md](./DEVELOPMENT.md) for the exact snippet.

## Things agents commonly get wrong here

- **Don't write to `docs/` directly.** Edit `templates/` and `examples/`, then run `make generate`.
- **Don't write to `launchdarkly/integration_configs_generated.go`.** Update `scripts/codegen` or the upstream integration-framework instead.
- **Don't add a new attribute string-literally.** Add a constant in `keys.go` first.
- **Don't bypass `withConcurrency`/the retry policies.** Direct `http.Client` calls won't respect rate limiting.
- **Don't assume acceptance tests will pass in CI** without LaunchDarkly credentials. They won't.
- **Don't pick the SDKv2 path "to be consistent" without thinking.** New resources should generally use the plugin-framework path (`plugin_provider.go`); SDKv2 is the legacy surface area.
- **Don't add backwards-compatibility shims** for schema changes without checking the existing migration patterns; Terraform state migrations have a specific shape (`StateUpgraders` for SDKv2, schema versions for plugin-framework).

## Where to look when stuck

- Provider configuration / auth / HTTP behaviour → [`launchdarkly/config.go`](./launchdarkly/config.go), [`launchdarkly/provider.go`](./launchdarkly/provider.go), [`launchdarkly/plugin_provider.go`](./launchdarkly/plugin_provider.go).
- A representative SDKv2 resource → [`launchdarkly/resource_launchdarkly_project.go`](./launchdarkly/resource_launchdarkly_project.go) + [`launchdarkly/project_helper.go`](./launchdarkly/project_helper.go).
- A representative plugin-framework resource → [`launchdarkly/resource_team_role_mapping.go`](./launchdarkly/resource_team_role_mapping.go) + [`launchdarkly/tests/resource_team_role_mapping_test.go`](./launchdarkly/tests/resource_team_role_mapping_test.go).
- Flag rule / targeting plumbing (clauses, rules, rollouts, fallthroughs, prerequisites) → the various `*_helper.go` files in `launchdarkly/`.
- User-facing docs / resource examples → [Terraform Registry](https://registry.terraform.io/providers/launchdarkly/launchdarkly/latest/docs).
- LaunchDarkly REST API reference → <https://launchdarkly.com/docs/api>.
