# Contributing

Thanks for your interest in improving the LaunchDarkly Terraform provider!

This file is the entry point for contributors. For deeper context:

- [DEVELOPMENT.md](./DEVELOPMENT.md) — build the provider, run tests, use a local override.
- [AGENTS.md](./AGENTS.md) — architecture overview, conventions, and guardrails. Aimed at AI coding agents but useful for any new contributor.

## Quickstart

```sh
git clone git@github.com:launchdarkly/terraform-provider-launchdarkly.git
cd terraform-provider-launchdarkly
make build       # compile into $GOPATH/bin
make test-unit   # fast unit tests, no LaunchDarkly API calls
```

If you use [asdf](https://asdf-vm.com) or [mise](https://mise.jdx.dev), `.tool-versions` pins the Go and golangci-lint versions CI uses.

## Pull requests

### Title format

**PR titles must follow [Conventional Commits](https://www.conventionalcommits.org/).** This is enforced by the `lint-pr-title` workflow and consumed by [release-please](https://github.com/googleapis/release-please) to drive `CHANGELOG.md` and version bumps. Examples:

- `feat: add launchdarkly_foo resource`
- `fix(segment): handle nil unbounded context`
- `docs: clarify ai_config_variation usage`
- `chore: bump go to 1.25.5`

`feat:` produces a minor version bump, `fix:` a patch, and any commit body containing `BREAKING CHANGE:` produces a major bump.

### Before opening a PR

1. `make fmt` (runs `gofmts` then `gofmt`).
2. `make vet` and `make errcheck`.
3. `make test-unit` (or `make test` for the full non-acceptance suite).
4. If you touched anything generated — `examples/`, `templates/`, `launchdarkly/integration_configs_generated.go`, or anything `tfplugindocs` reads — run `make generate` and commit the resulting changes. CI runs `make generate` and fails if it produces a diff.
5. If you touched a resource, update or add a matching example under `examples/resources/launchdarkly_<name>/` so the generated docs stay accurate.

### Acceptance tests

`make testacc` runs the full acceptance suite (`TF_ACC=1`). **These hit the real LaunchDarkly API and create/destroy real resources.** They require a `LAUNCHDARKLY_ACCESS_TOKEN` for an enterprise-tier account. CI runs them automatically against a project sandbox — you usually don't need to run them locally unless you're debugging a specific failure. Use `TESTARGS="-run TestAccFoo"` to scope to a single test.

If you don't have an enterprise account, open the PR with unit tests; a maintainer will run acceptance tests against the sandbox.

## Pre-commit hooks

The repo ships a `.pre-commit-config.yaml`. To install:

```sh
pip install pre-commit
pre-commit install
```

This runs `gofmts`, `golangci-lint`, and `make generate` on each commit so you catch problems before pushing. The hook versions are pinned to match CI — if you bump one, bump both (`.pre-commit-config.yaml` and `.github/workflows/test.yml`).

## Code conventions

See [AGENTS.md § Conventions to follow](./AGENTS.md#conventions-to-follow). The short version:

- Add new schema attribute names as constants in [`launchdarkly/keys.go`](./launchdarkly/keys.go); never inline string literals.
- Wrap LaunchDarkly API calls in `client.withConcurrency(ctx, ...)` so the provider-wide rate-limit semaphore applies.
- Don't hand-edit generated files (`launchdarkly/integration_configs_generated.go`, anything in `docs/`). Edit the source (`scripts/codegen` or `templates/` + `examples/`) and run `make generate`.
- Prefer the plugin-framework path (`launchdarkly/plugin_provider.go`) for new resources; SDKv2 is the legacy surface area.

## Reporting issues

Bug reports and feature requests live on [GitHub Issues](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues). For LaunchDarkly product questions unrelated to Terraform, see <https://launchdarkly.com/support>.
