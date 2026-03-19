# Branch Comparison: AI Config Terraform Provider Implementations

**Branch A**: `ffeldberg/test-skill-on-ai-configs-resources` (this branch, Claude Code + `terraform-provider-add-resource` skill)
**Branch B**: `devin/1773079106-add-ai-config-resource` (Devin)

---

## 1. Scope

| | Branch A | Branch B |
|---|---|---|
| Resources | 4 (`ai_config`, `ai_config_variation`, `model_config`, `ai_tool`) | 1 (`ai_config`) |
| Data Sources | 3 (`ai_config`, `model_config`, `ai_tool`) | 1 (`ai_config`) |

Branch A covers the full AI Configs domain. Branch B only implements the top-level config — users cannot manage prompts, model parameters, or tool schemas through Terraform.

## 2. Architecture

Both branches use the generated SDK client (`client.ld.AIConfigsApi`), the standard `terraform-plugin-sdk/v2`, `withConcurrency` for rate limiting, and the `*_helper.go` shared-read pattern. No meaningful architectural difference for the shared `ai_config` resource.

Branch A additionally provides `json_helper.go` with reusable JSON validation/diff-suppression utilities used across variation, model config, and tool resources.

## 3. Schema Design (`ai_config`)

| Field | Branch A | Branch B |
|---|---|---|
| `mode` | Default `"completion"`, validates `["completion","agent"]` | No default, validates `["agent","completion","judge"]` |
| `maintainer_id` / `maintainer_team_key` | `ConflictsWith` enforced | No `ConflictsWith` |
| `creation_date` | Computed | Missing |
| `variations` | Computed list of `{key, name, variation_id}` | Missing |

Branch B includes `"judge"` as a valid mode (may be more API-complete). Branch A enforces mutual exclusion on maintainer fields and exposes more computed data.

## 4. Variation Handling

- **Branch A**: Full separate `launchdarkly_ai_config_variation` resource with messages, model, tools, instructions, state, import support.
- **Branch B**: Not implemented at all.

## 5. Test Coverage

### Branch A (8 test functions across 7 files)
- `TestAccAIConfig_CreateAndUpdate` — basic create/update/import with name, description, tags
- `TestAccAIConfigVariation_CreateAndUpdate` — create/update messages, import
- `TestAccModelConfig_CreateAndImport` — create + import (no update, ForceNew)
- `TestAccAITool_CreateAndUpdate` — create/update schema_json, import
- 3 data source tests with `_noMatchReturnsError` + `_exists` each

### Branch B (7 test functions across 2 files)
- `TestAccAiConfig_BasicCreateAndUpdate` — basic create/update/import
- `TestAccAiConfig_WithMode` — explicit mode=completion test
- `TestAccAiConfig_RemoveOptionalFields` — removal of description/tags reverts to defaults
- `TestAccAiConfig_WithMaintainer` — maintainer_id with real team_member
- `TestAccAiConfig_WithTeamMaintainer` — maintainer_team_key with real team
- `TestAccAiConfig_WithEvaluationMetric` — evaluation_metric_key + is_inverted with real metric
- `TestAccDataSourceAiConfig_Basic` — data source read

**Branch B tests `ai_config` much more thoroughly** — it covers mode, maintainer variants, evaluation metrics, and optional field removal. Branch A's `ai_config` test only covers the basic flow but has broader coverage across 4 resource types.

## 6. CI Integration

- **Branch A**: Added `TestAccAIConfig_`, `TestAccAIConfigVariation`, `TestAccAITool`, `TestAccModelConfig` to `.github/workflows/test.yml` matrix.
- **Branch B**: No CI changes. Its test names (`TestAccAiConfig_*`) wouldn't match any existing matrix entry, so **tests would silently not run in CI**.

## 7. Docs & Examples

- **Branch A**: Examples in `examples/resources/` and `examples/data-sources/` for all resources. Docs auto-generated via `make generate`.
- **Branch B**: Examples + full generated docs in `docs/resources/ai_config.md` and `docs/data-sources/ai_config.md` with two HCL examples (completion + agent modes).

Both branches have docs/examples. Branch B's inline examples are slightly richer (showing both modes).

## 8. Code Quality

| Aspect | Branch A | Branch B |
|---|---|---|
| Go naming | `resourceAIConfig` (idiomatic — acronyms all-caps) | `resourceAiConfig` (non-standard) |
| `Exists` function | Included on all resources | Omitted |
| Tags handling | `stringsFromResourceData` (consistent with codebase) | `interfaceSliceToStringSlice` (works but different) |
| Maintainer read | Reads both `MaintainerMember` and `AiConfigsMaintainerTeam` from API | Acknowledges team key gap, uses `ImportStateVerifyIgnore` |
| ForceNew on data sources | Conditional `ForceNew: !isDataSource` | Hardcoded, relies on `removeInvalidFieldsForDataSource` |

## 9. What Each Branch is Missing

### Branch A gaps
1. Shallow `ai_config` test coverage — no tests for `mode`, maintainers, evaluation metrics, optional field removal
2. `mode` validation missing `"judge"` (if API supports it)
3. No `ai_config_variation` data source

### Branch B gaps
1. No `ai_config_variation`, `model_config`, or `ai_tool` resources/data sources
2. No CI workflow updates (tests won't run)
3. No `ConflictsWith` on maintainer fields
4. No `Exists` function
5. Non-idiomatic Go naming (`Ai` vs `AI`)
6. Missing `creation_date` and `variations` computed fields

## Overall Assessment

| Dimension | Winner |
|---|---|
| Scope (4 resources vs 1) | **Branch A** |
| `ai_config` test depth | **Branch B** |
| CI integration | **Branch A** |
| Docs/examples | Tie |
| Code quality / conventions | **Branch A** (slight) |
| Overall completeness | **Branch A** |

**Recommendation**: Use Branch A as the base (4x resource coverage, CI integration, better conventions), but port Branch B's deeper `ai_config` test scenarios (mode, maintainers, evaluation metrics, optional field removal) and consider adding `"judge"` to mode validation.
