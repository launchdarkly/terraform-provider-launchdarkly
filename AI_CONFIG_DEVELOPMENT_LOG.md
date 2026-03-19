# AI Config Resources - Development Log

## Overview

Added Terraform provider support for LaunchDarkly AI Configs: 4 new resources, 3 data sources, acceptance tests, examples, CI updates, and auto-generated docs.

## Prompts Used

### 1. Initial request
> Please use the terraform provider add resource skill in order to add support for AI Config resources and data sources to the provider. Use our public facing documentation etc. as needed to inform your work https://launchdarkly.com/docs/home/ai-configs https://launchdarkly.com/docs/api/ai-configs

### 2. Scope clarification (asked by Claude)
Claude asked which resources to implement and whether variations should be inline or separate. Response:
> Implement all of these, when looking at variations it'd be best to follow existing patterns for our flag resource which also supports variations. I believe separate resources make the most sense, but make sure to compare against both the flag resource and potential guidance from our docs.

### 3. Build command correction
> It looks like you didn't run make fmt and make generate - are those mentioned in the skill for creating a new resource in the provider? I also can't see any new tests, we want both unit acceptance tests (including updates to the CI workflow to ensure acceptance tests are run) - is that mentioned in the skill?

### 4. Skill discovery
> There is a skill for adding a new resource to the provider - it's called terraform-provider-add-resource - please double check

This pointed Claude to the correct `terraform-provider-add-resource` skill, which defined the full checklist (Steps 0-10) including `make fmt`, `make generate`, example files, acceptance tests, and CI matrix updates.

### 5. Development log request
> Can you give me a writeup on the development process we've followed...

### 6. Cross-branch comparison
> Can you compare the solution you've come up with on this branch to another implementation on the branch devin/1773079106-add-ai-config-resource and compare/contrast/assess each implementation?

Claude compared both branches across 9 dimensions (scope, architecture, schema design, variation handling, test depth, CI, docs, code quality, completeness). See `IMPLEMENTATION_COMPARISON.md` for the full analysis.

### 7. Address identified gaps
> Can you address the gaps in our branch that you identified?

Added 5 new `ai_config` test scenarios ported from the comparison findings: `TestAccAIConfig_WithMode`, `TestAccAIConfig_WithMaintainer`, `TestAccAIConfig_WithTeamMaintainer`, `TestAccAIConfig_WithEvaluationMetric`, `TestAccAIConfig_RemoveOptionalFields`. Verified `"judge"` mode doesn't exist in the Go API client — skipped.

### 8. CI lint fix
> I've just checked the CI lint logs and there's only one error... func `validateJsonString` is unused

Removed the unused `validateJsonString()` function and its `validation` import from `json_helper.go`.

### 9. CI InternalValidate fix
> Another issue from CI... resource launchdarkly_model_config: No Update defined, must set ForceNew on: []string{"tags"} / data source launchdarkly_model_config: provider is a reserved field name

Two fixes: (1) Added `ForceNew: true` to the `tags` field on `model_config` by overriding the `tagsSchema()` result. (2) Renamed the reserved `provider` attribute to `model_provider` across `keys.go`, helper, resource, tests, and examples.

### 10. Local testing
> Now please build the current version of the provider and attempt to test it using the local-testing folder

Built provider with `make build`, created `local-testing/ai_config_test.tf` with all 4 resources + 3 data sources + outputs. `terraform plan` succeeded — all resources planned correctly. Apply requires a live API token.

## Skill Used: `terraform-provider-add-resource`

The `terraform-provider-add-resource` skill (v1.0.0, from `launchdarkly-agent-skills`) defines a 10-step process for adding new resources to the provider:

| Step | What |
|------|------|
| 0 | Understand the API surface (OpenAPI spec, Go client types) |
| 1 | Add schema constants to `keys.go` (with `//gofmts:sort`) |
| 2 | Create helper file (`<name>_helper.go`) with shared schema + read logic |
| 3 | Create resource file with CRUD + Exists + Import |
| 4 | Create data source file (thin wrapper around shared read) |
| 5 | Register in `provider.go` ResourcesMap/DataSourcesMap |
| 6 | Create `examples/resources/` and `examples/data-sources/` with `resource.tf`, `import.sh`, `data-source.tf` |
| 7 | Doc templates (optional - `make generate` auto-generates from schema) |
| 8 | Write acceptance tests (create/import/update steps, CheckDestroy) |
| 9 | Add test cases to `.github/workflows/test.yml` matrix |
| 10 | Run `make fmt`, `make build`, `make generate` |

The skill also includes a `references/patterns.md` with full code templates for helpers, resources, data sources, and tests.

## What Was Built

### Resources
- `launchdarkly_ai_config` — core AI Config (key, name, mode, description, tags)
- `launchdarkly_ai_config_variation` — variation as separate child resource (messages, model, tools, instructions)
- `launchdarkly_model_config` — project-level model configuration (ForceNew only, no PATCH API)
- `launchdarkly_ai_tool` — project-level AI tool definitions (JSON schema, custom params)

### Data Sources
- `launchdarkly_ai_config`, `launchdarkly_model_config`, `launchdarkly_ai_tool`

### Key Design Decisions
- **Variations as separate resources** rather than inline blocks, following the `feature_flag` / `feature_flag_environment` pattern
- **Standard API client** (`client.ld.AIConfigsApi`) — the v22 Go client doesn't require beta headers for AI Config endpoints
- **Model config is immutable** — no PATCH API exists, so all fields are `ForceNew: true`
- **JSON fields** (`model`, `params`, `schema_json`, `custom_parameters`) use `TypeString` with semantic diff suppression to avoid false diffs from key ordering

### Files Changed/Created
- 11 new Go source files (helpers, resources, data sources)
- 7 new test files
- 11 new example files (`resource.tf`, `import.sh`, `data-source.tf`)
- Modified: `keys.go`, `provider.go`, `test.yml`
- Auto-generated: `docs/resources/*.md`, `docs/data-sources/*.md`
