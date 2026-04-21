# OpenAPI Provider Generation (v1)

This repository includes an OpenAPI-driven generation pipeline for framework/mux resources, metadata, and acceptance test scaffolding.

## Goals

- Keep generated framework artifacts reproducible.
- Keep generated acceptance tests aligned with existing `launchdarkly/tests` patterns.
- Validate configured CRUD operation mappings against live OpenAPI.
- Support staged rollout from scaffolded generic resources to full parity implementations.

## Inputs

- Overlay config (maintainer-owned): `templates/openapi-provider-gen/config.json`
- Discovered catalog (machine-generated): `templates/openapi-provider-gen/catalog.auto.json`
- Source OpenAPI document: `https://app.launchdarkly.com/api/v2/openapi.json`

## Generation Commands

### 1. Refresh discovered catalog from OpenAPI

```bash
go run ./scripts/openapi-provider-gen \
  --overlay ./templates/openapi-provider-gen/config.json \
  --discover-catalog-out ./templates/openapi-provider-gen/catalog.auto.json \
  --discover-only
```

### 2. Generate provider/test artifacts from overlay + catalog

```bash
go run ./scripts/openapi-provider-gen \
  --overlay ./templates/openapi-provider-gen/config.json \
  --catalog ./templates/openapi-provider-gen/catalog.auto.json \
  --template-dir ./templates/openapi-provider-gen \
  --out-dir ./launchdarkly \
  --tests-out-dir ./launchdarkly/tests
```

This is also wired into `go generate` via [`launchdarkly/plugin_provider.go`](/Users/fabianfeldberg/code/launchdarkly/terraform-provider-launchdarkly/launchdarkly/plugin_provider.go).

## Config Contract

Top-level fields:

- `version`: schema version (`v1`)
- `provider.name`
- `provider.openapi_url`
- `framework.resources[]`

Per-resource fields:

- `terraform_name`
- `framework_type_name`
- `constructor`
- `implementation`
- `register_framework`
- `enabled`: controls provider registration eligibility
- `experimental`: registration gated behind env var
- `rollout_phase`
- `modify_plan_hook`: optional framework `ModifyPlan` escape hatch
- `identity_fields`
- `mutable_fields`
- `import_ignore`
- `operations` (`create/read/update/delete` with `method` + `path`)
- `test.enabled`
- `test.scenario` (`team`, `project`, `team_role_mapping`, `generic`)
- `test.fixture` (required for runnable generic tests; otherwise generic test skips)

## Generated Outputs

- [`launchdarkly/plugin_provider_gen.go`](/Users/fabianfeldberg/code/launchdarkly/terraform-provider-launchdarkly/launchdarkly/plugin_provider_gen.go)
- [`launchdarkly/openapi_provider_metadata_gen.go`](/Users/fabianfeldberg/code/launchdarkly/terraform-provider-launchdarkly/launchdarkly/openapi_provider_metadata_gen.go)
- [`launchdarkly/openapi_generated_resources_gen.go`](/Users/fabianfeldberg/code/launchdarkly/terraform-provider-launchdarkly/launchdarkly/openapi_generated_resources_gen.go)
- [`launchdarkly/tests/generated_openapi_acceptance_test.go`](/Users/fabianfeldberg/code/launchdarkly/terraform-provider-launchdarkly/launchdarkly/tests/generated_openapi_acceptance_test.go)

Do not edit generated files manually.

## Experimental Registration Gate

Resources with `enabled: true` and `experimental: true` register only when:

```bash
export LD_TERRAFORM_EXPERIMENTAL_GENERATED_RESOURCES=true
```

Non-experimental enabled resources always register.

## Escape Hatch for Manual Plan Logic

Use `modify_plan_hook` per resource to call a handwritten helper from generated resources.

Example:

```json
{
  "framework_type_name": "generated_project",
  "modify_plan_hook": "generatedProjectModifyPlan"
}
```

Then implement in handwritten Go:

```go
func generatedProjectModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
    // custom plan normalization / validation
}
```

## Acceptance Test Contract

Generated acceptance tests use:

- `resource.Test` with `ProtoV5ProviderFactories: testAccFrameworkMuxProviders(...)`
- `testAccPreCheck`
- lifecycle test steps (`basic`, optional `update`)
- import verification (`ImportState`, `ImportStateVerify`)
- configurable `ImportStateVerifyIgnore`

## Current Parity Surface

Registered generated resources for direct comparison:

- `launchdarkly_generated_team`
- `launchdarkly_generated_project`
- `launchdarkly_generated_team_role_mapping`

Manual framework resource preserved for comparison:

- `launchdarkly_team_role_mapping`

SDKv2 manual resources remain unchanged.

Example configs:

- [`examples/resources/launchdarkly_generated_team/resource.tf`](/Users/fabianfeldberg/code/launchdarkly/terraform-provider-launchdarkly/examples/resources/launchdarkly_generated_team/resource.tf)
- [`examples/resources/launchdarkly_generated_project/resource.tf`](/Users/fabianfeldberg/code/launchdarkly/terraform-provider-launchdarkly/examples/resources/launchdarkly_generated_project/resource.tf)
- [`examples/resources/launchdarkly_generated_team_role_mapping/resource.tf`](/Users/fabianfeldberg/code/launchdarkly/terraform-provider-launchdarkly/examples/resources/launchdarkly_generated_team_role_mapping/resource.tf)

## How To Generate All Resources

1. Refresh catalog from OpenAPI (`--discover-only` command above).
2. Promote resources in overlay by adding overrides with:
   - matching `framework_type_name`
   - `enabled: true`
   - desired `implementation` (`generic` for scaffold, or specialized template)
   - optional `experimental`/`modify_plan_hook`/test metadata.
3. Run `go generate ./launchdarkly/plugin_provider.go`.
4. Inspect generated outputs and run targeted tests.

Recommended rollout: keep newly promoted resources `experimental: true` first, then remove experimental flag after parity validation.

## CI + Drift

`make generate` runs in [`.github/workflows/test.yml`](/Users/fabianfeldberg/code/launchdarkly/terraform-provider-launchdarkly/.github/workflows/test.yml) and fails on generated drift.

## Agent Workflows

- [`.github/workflows/agent-self-review.yml`](/Users/fabianfeldberg/code/launchdarkly/terraform-provider-launchdarkly/.github/workflows/agent-self-review.yml)
  - Trigger: `pull_request` (`opened`, `synchronize`, `reopened`)
  - Advisory only (comment output)
  - No `pull_request_target`
  - Pinned actions, minimal permissions
- [`.github/workflows/agent-manual-test.yml`](/Users/fabianfeldberg/code/launchdarkly/terraform-provider-launchdarkly/.github/workflows/agent-manual-test.yml)
  - Trigger: `workflow_dispatch`
  - Inputs: `test_regex`, `resource`, `scenario`, `dry_run`, optional `pr_number`
  - Runs targeted test command and publishes artifact + advisory analysis
  - No automatic code writeback
