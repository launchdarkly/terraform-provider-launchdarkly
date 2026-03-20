# AI Config Resources — Review Fix Plan

Status: **COMPLETE — all issues implemented**

---

## P0 — Bug

### 1. `GetOk` on bool field silently drops `true→false` updates

**Location**: `resource_launchdarkly_ai_config.go`

**Problem**: `d.GetOk()` returns `ok=false` for Go zero-values (empty string, false, 0). When a user changes `is_inverted` from `true` to `false`, `HasChange` fires but `GetOk` returns `ok=false`, so the patch is sent without the field. The update is silently dropped — state and API disagree.

Same pattern exists for `EVALUATION_METRIC_KEY` in update (can't unset to empty string) and `IS_INVERTED` in create (explicit `false` is ignored).

**Fix**:

In **update** (lines 139-153), replace the `GetOk` guard with `d.Get()` for both fields:

```go
// IS_INVERTED — use d.Get() because GetOk returns ok=false for bool(false)
if d.HasChange(IS_INVERTED) {
    isInverted := d.Get(IS_INVERTED).(bool)
    patch.IsInverted = &isInverted
    hasChanges = true
}

// EVALUATION_METRIC_KEY — use d.Get() so users can unset to ""
if d.HasChange(EVALUATION_METRIC_KEY) {
    evaluationMetricKey := d.Get(EVALUATION_METRIC_KEY).(string)
    patch.EvaluationMetricKey = &evaluationMetricKey
    hasChanges = true
}
```

In **create** (lines 65-73), same fix — use `d.Get()` so explicitly setting `false`/empty works:

```go
if v, ok := d.GetOk(EVALUATION_METRIC_KEY); ok {
    evaluationMetricKey := v.(string)
    post.EvaluationMetricKey = &evaluationMetricKey
}

// IS_INVERTED: only send if evaluation_metric_key is also set,
// since is_inverted is meaningless without a metric
if _, hasMetric := d.GetOk(EVALUATION_METRIC_KEY); hasMetric {
    isInverted := d.Get(IS_INVERTED).(bool)
    post.IsInverted = &isInverted
}
```

**Files changed**: `resource_launchdarkly_ai_config.go`

---

## P1 — Functional

### 9. `ExpectNonEmptyPlan` in `WithToolKeys` test — resource doesn't converge

**Location**: `resource_launchdarkly_ai_config_variation_test.go:332-365`

**Problem**: Both steps use `ExpectNonEmptyPlan: true`, meaning the resource never reaches a clean plan. Users will see persistent diffs on every `terraform plan` when using `tool_keys`.

**Root cause**: This is a version propagation issue. The AI Config Variation API creates a new version on every PATCH. The read function already selects the highest version from `Items[]`, but if the GET is called before the new version is indexed, `Items[]` only contains the old version — so the read returns stale data, creating a diff.

**Decision**: **(A) Short sleep before Read** — add a brief delay (~1-2s) after POST/PATCH in Create and Update, before calling Read, to allow the new version to propagate. Then remove `ExpectNonEmptyPlan` from tests.

The sleep goes in the resource Create/Update functions (not in Read itself, since Read shouldn't have arbitrary sleeps). Add it between the API call and the `return resourceAIConfigVariationRead(...)` call:

```go
// Brief pause to allow the new variation version to propagate before reading.
// The API creates a new version on each write; the GET endpoint may not
// immediately return the latest version due to eventual consistency.
time.Sleep(2 * time.Second)

return resourceAIConfigVariationRead(ctx, d, metaRaw)
```

**Files changed**: `resource_launchdarkly_ai_config_variation.go`, `resource_launchdarkly_ai_config_variation_test.go`

---

## P2 — Missing Coverage / Validation

### 6. No `ai_config_variation` data source

**Problem**: Data sources exist for `ai_config`, `ai_tool`, and `model_config`, but not for `ai_config_variation`. Users can't reference an existing variation without managing it as a resource.

**Fix**: Create the data source following the existing pattern:
- Refactor `aiConfigVariationSchema()` to accept `isDataSource bool`, matching the pattern used by `baseAIConfigSchema`, `baseAIToolSchema`, and `baseModelConfigSchema`. When `isDataSource=true`: lookup fields (`project_key`, `config_key`, `key`) stay Required, all other fields become Computed, and `removeInvalidFieldsForDataSource` is applied.
- Create `data_source_launchdarkly_ai_config_variation.go` — thin wrapper calling a shared read function.
- The existing `aiConfigVariationRead` needs a minor refactor to accept `isDataSource bool` and set the ID for data sources (same as the other 3 read functions).
- Create `data_source_launchdarkly_ai_config_variation_test.go` — error test + exists test.
- Register in `provider.go` under `DataSourcesMap`.
- Add example in `examples/data-sources/launchdarkly_ai_config_variation/data-source.tf`.
- Add `TestAccDataSourceAIConfigVariation` to CI matrix in `.github/workflows/test.yml` (or confirm it's covered by existing patterns).
- Run `make generate` for docs.

**Files changed**: `ai_config_variation_helper.go`, `resource_launchdarkly_ai_config_variation.go`, new `data_source_launchdarkly_ai_config_variation.go`, new `data_source_launchdarkly_ai_config_variation_test.go`, `provider.go`, new example file, generated docs

### 8. No `ConflictsWith` on `model` vs `model_config_key` (variation)

**Problem**: Both `MODEL` and `MODEL_CONFIG_KEY` are `Optional` with no mutual exclusion. A user setting both gets an opaque API error.

**Fix**: Add `ConflictsWith` to both fields in `ai_config_variation_helper.go`:

```go
MODEL: {
    ...
    ConflictsWith: []string{MODEL_CONFIG_KEY},
},
MODEL_CONFIG_KEY: {
    ...
    ConflictsWith: []string{MODEL},
},
```

**Files changed**: `ai_config_variation_helper.go`

---

## P3 — Improvements

### 10. `is_inverted` without `evaluation_metric_key` is meaningless

**Problem**: Setting `is_inverted = true` without `evaluation_metric_key` is semantically invalid but accepted.

**Fix**: Add a `CustomizeDiff` function on the `ai_config` resource. `RequiredWith` isn't suitable because setting `evaluation_metric_key` without `is_inverted` IS valid. The diff function only errors if `is_inverted = true` AND `evaluation_metric_key` is empty:

```go
CustomizeDiff: customdiff.All(
    func(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
        isInverted := diff.Get(IS_INVERTED).(bool)
        metricKey, _ := diff.GetOk(EVALUATION_METRIC_KEY)
        if isInverted && (metricKey == nil || metricKey.(string) == "") {
            return fmt.Errorf("is_inverted requires evaluation_metric_key to be set")
        }
        return nil
    },
),
```

**Files changed**: `resource_launchdarkly_ai_config.go`

### 2. Test configs duplicate project definitions (all 4 test files + 3 data source test files)

**Problem**: Every test config const embeds a full `launchdarkly_project` resource block. ~200 lines of pure repetition.

**Decision**: **(A) New shared `ai_test_helpers_test.go` file** — usable by all AI config/tool/variation/model test files.

**Fix**: Create `launchdarkly/ai_test_helpers_test.go` with:

```go
func withAITestProject(projectKey, resource string) string {
    return fmt.Sprintf(`
resource "launchdarkly_project" "test" {
    key  = "%s"
    name = "AI Config Test Project"
    environments {
        name  = "Test Environment"
        key   = "test-env"
        color = "000000"
    }
}

%s`, projectKey, resource)
}
```

Then refactor all test configs to use this helper. Test configs that need additional resources (team, model_config, ai_tool) concatenate them in the resource string passed to the helper.

Also move the shared cooldown function (issue #12) and serial test file comment (issue #3) here.

**Files changed**: New `ai_test_helpers_test.go`, all 4 resource test files, all 3 data source test files

### 7. No test for inline `model` JSON on variations

**Problem**: The `MODEL` field accepts inline JSON but no test exercises this path. The `jsonStringToMap`/`mapToJsonString`/`isEmptyModelMap` roundtrip is untested in acceptance tests.

**Fix**: Add a `TestAccAIConfigVariation_WithInlineModel` test:

```hcl
resource "launchdarkly_ai_config_variation" "test" {
    project_key = launchdarkly_project.test.key
    config_key  = launchdarkly_ai_config.test.key
    key         = "%s"
    name        = "Variation with inline model"
    model       = jsonencode({
        modelName  = "gpt-4"
        parameters = { temperature = 0.7 }
    })
    messages {
        role    = "system"
        content = "You are a helpful assistant."
    }
}
```

Checks verify `model` is set and roundtrips cleanly after import.

**Files changed**: `resource_launchdarkly_ai_config_variation_test.go`

### 12. Duplicated cooldown function

**Problem**: `aiConfigTestCooldown()` and `aiConfigVariationTestCooldown()` are identical.

**Fix**: Replace both with a single `aiTestCooldown()` in the new shared `ai_test_helpers_test.go` (issue #2). Update all call sites.

**Files changed**: `ai_test_helpers_test.go` (new), `resource_launchdarkly_ai_config_test.go`, `resource_launchdarkly_ai_config_variation_test.go`

---

## P4 — Style / Convention Nits

### 3. AI Config and Variation tests use `resource.Test` instead of `resource.ParallelTest`

**Problem**: Intentionally serial due to rate limits, but undocumented at the file level.

**Fix**: Add a comment on the shared `aiTestCooldown()` function in `ai_test_helpers_test.go` explaining the serial test choice:

```go
// aiTestCooldown adds a brief delay between AI config / variation tests.
// These tests use resource.Test (serial) instead of resource.ParallelTest
// because the AI Config API creates feature flags internally. The flag creation
// endpoint has a tight rate limit that returns 429, but the AI Config API handler
// translates this to a 400, bypassing the retry client. Serial execution with
// cooldown pauses avoids these transient failures.
func aiTestCooldown() {
    time.Sleep(2 * time.Second)
}
```

**Files changed**: `ai_test_helpers_test.go` (new — shared with issues #2 and #12)

### 4. `ValidateFunc` vs `ValidateDiagFunc` inconsistency

**Problem**: JSON fields use old-style `ValidateFunc` while other fields use `ValidateDiagFunc`.

**Decision**: **(A) Convert now** for consistency.

**Fix**: In `json_helper.go`, add a `ValidateDiagFunc`-compatible wrapper:

```go
func validateJsonStringDiagFunc() schema.SchemaValidateDiagFunc {
    return validation.ToDiagFunc(func(v interface{}, k string) ([]string, []error) {
        return validateJsonStringFunc(v, k)
    })
}
```

Then update all schema fields that currently use `ValidateFunc: validateJsonStringFunc` to use `ValidateDiagFunc: validateJsonStringDiagFunc()`. Similarly, convert `DiffSuppressFunc` usage to be consistent (DiffSuppressFunc doesn't have a Diag variant, so it stays as-is — only the validate needs converting).

Fields to update:
- `ai_config_variation_helper.go`: `MODEL` field
- `ai_tool_helper.go`: `SCHEMA_JSON` and `CUSTOM_PARAMETERS` fields (these use `emptyValueIfDataSource(validateJsonStringFunc, ...)` which also needs converting)
- `model_config_helper.go`: `PARAMS` and `CUSTOM_PARAMETERS` fields (same emptyValueIfDataSource pattern)

For the `emptyValueIfDataSource` calls: since `ValidateDiagFunc` is a function type (not an interface), `emptyValueIfDataSource` works the same way — it returns nil for data sources. Just change from `ValidateFunc: emptyValueIfDataSource(validateJsonStringFunc, isDataSource)` to `ValidateDiagFunc: emptyValueIfDataSource(validateJsonStringDiagFunc(), isDataSource)`.

**Files changed**: `json_helper.go`, `ai_config_variation_helper.go`, `ai_tool_helper.go`, `model_config_helper.go`

### 5. Provider registration ordering

**Problem**: New resources aren't sorted consistently within `ResourcesMap`.

**Fix**: Move all entries in `ResourcesMap` and `DataSourcesMap` to be alphabetically sorted. This is a one-line-per-entry reorder.

**Files changed**: `provider.go`

### 11. Add `CheckDestroy` for data source tests

**Problem**: Data source tests don't verify underlying resources are cleaned up.

**Decision**: Add `CheckDestroy` to all 3 existing data source tests + the new variation data source test.

**Fix**: Reuse the existing `testAccCheckAIConfigDestroy`, `testAccCheckAIToolDestroy`, `testAccCheckModelConfigDestroy` functions in the data source tests. For the variation data source test, reuse `testAccCheckAIConfigVariationDestroy`. These are already defined in the corresponding resource test files and accessible within the same package.

**Files changed**: `data_source_launchdarkly_ai_config_test.go`, `data_source_launchdarkly_ai_tool_test.go`, `data_source_launchdarkly_model_config_test.go`, new `data_source_launchdarkly_ai_config_variation_test.go`

---

## Implementation Order

1. **#1** — P0 bug fix (`GetOk` on bool) — immediate safety fix
2. **#8** — `ConflictsWith` on model/model_config_key — simple schema addition
3. **#10** — `CustomizeDiff` for is_inverted — simple validation addition
4. **#4** — ValidateFunc→ValidateDiagFunc conversion
5. **#5** — Sort provider.go entries
6. **#2 + #12 + #3** — Create shared test helper file, deduplicate cooldown, add comments (bundled since they share a file)
7. **#11** — Add CheckDestroy to data source tests
8. **#7** — Add inline model test
9. **#9** — Fix tool_keys convergence (sleep before read)
10. **#6** — Add variation data source (most work)
11. Run `make fmt && make generate` to finalize

---

## Checklist

- [x] #1 — Fix `GetOk` on bool/string zero-values in ai_config create+update
- [x] #8 — Add `ConflictsWith` on model/model_config_key
- [x] #10 — Add `CustomizeDiff` for is_inverted + evaluation_metric_key
- [x] #4 — Convert `ValidateFunc` → `ValidateDiagFunc` for JSON fields
- [x] #5 — Alphabetize provider.go resource/data source registration
- [x] #2 — Create `ai_test_helpers_test.go` with shared project helper
- [x] #12 — Deduplicate cooldown into shared helper
- [x] #3 — Add serial test rationale comment
- [x] #11 — Add CheckDestroy to data source tests
- [x] #7 — Add inline model acceptance test
- [x] #9 — Add sleep before read in variation create+update, remove ExpectNonEmptyPlan
- [x] #6 — Create ai_config_variation data source + tests + docs
- [x] Final — `make fmt && make generate`

---

## PR Comment Triage (from automated review)

19 comments on `docs/` files — all moot since docs are auto-generated by `make generate`.

2 comments on `GetOk` for `is_inverted` and `evaluation_metric_key` — **already fixed** by Issue #1.

6 new code issues identified below.

---

## P1 — Functional (from PR comments)

### 13. `aiConfigRead` doesn't always set `mode` / `evaluation_metric_key` — stale state drift

**Location**: `ai_config_helper.go:164-170`

**Problem**: `mode` and `evaluation_metric_key` are only set in state when the API returns non-nil pointers. If these fields are unset server-side (or removed by an out-of-band update), the prior state value persists, causing perpetual diffs.

**Fix**: Always set both fields, defaulting to `"completion"` for mode and `""` for evaluation_metric_key when nil:

```go
mode := "completion"
if aiConfig.Mode != nil {
    mode = *aiConfig.Mode
}
_ = d.Set(MODE, mode)

evaluationMetricKey := ""
if aiConfig.EvaluationMetricKey != nil {
    evaluationMetricKey = *aiConfig.EvaluationMetricKey
}
_ = d.Set(EVALUATION_METRIC_KEY, evaluationMetricKey)
```

**Files changed**: `ai_config_helper.go`

### 14. `aiConfigRead` doesn't clear opposite maintainer field — stale state drift

**Location**: `ai_config_helper.go:178-185`

**Problem**: Read only sets `maintainer_id` or `maintainer_team_key` when the corresponding union member is present, but never clears the opposite field. If a user changes from member maintainer to team maintainer (or vice versa), the old value persists in state.

**Fix**: Default both to empty string before setting the one returned by the API:

```go
_ = d.Set(MAINTAINER_ID, "")
_ = d.Set(MAINTAINER_TEAM_KEY, "")
maintainer := aiConfig.GetMaintainer()
if maintainer.MaintainerMember != nil {
    _ = d.Set(MAINTAINER_ID, maintainer.MaintainerMember.GetId())
}
if maintainer.AiConfigsMaintainerTeam != nil {
    _ = d.Set(MAINTAINER_TEAM_KEY, maintainer.AiConfigsMaintainerTeam.GetKey())
}
```

**Files changed**: `ai_config_helper.go`

### 15. `aiToolRead` doesn't clear opposite maintainer field — stale state drift

**Location**: `ai_tool_helper.go:141-148`

**Problem**: Same issue as #14 but in the AI tool read function.

**Fix**: Same pattern — default both to empty before setting:

```go
_ = d.Set(MAINTAINER_ID, "")
_ = d.Set(MAINTAINER_TEAM_KEY, "")
maintainer := tool.GetMaintainer()
if maintainer.MaintainerMember != nil {
    _ = d.Set(MAINTAINER_ID, maintainer.MaintainerMember.GetId())
}
if maintainer.AiConfigsMaintainerTeam != nil {
    _ = d.Set(MAINTAINER_TEAM_KEY, maintainer.AiConfigsMaintainerTeam.GetKey())
}
```

**Files changed**: `ai_tool_helper.go`

### 16. AI tool update can't unset `maintainer_id` / `maintainer_team_key`

**Location**: `resource_launchdarkly_ai_tool.go:113-123`

**Problem**: When a user removes `maintainer_id` or `maintainer_team_key` from config, `HasChange` is true but `GetOk` returns false (empty string is a zero-value), so the patch omits the field. The server-side maintainer is never cleared, causing persistent drift.

**Fix**: Use `d.Get()` instead of `d.GetOk()` so the empty string is sent to the API:

```go
if d.HasChange(MAINTAINER_ID) {
    maintainerId := d.Get(MAINTAINER_ID).(string)
    patch.MaintainerId = ldapi.PtrString(maintainerId)
}

if d.HasChange(MAINTAINER_TEAM_KEY) {
    maintainerTeamKey := d.Get(MAINTAINER_TEAM_KEY).(string)
    patch.MaintainerTeamKey = ldapi.PtrString(maintainerTeamKey)
}
```

**Note**: Same `GetOk` zero-value trap as Issue #1. The ai_config update (lines 128-142) has the same issue but needs investigation on whether the API accepts empty string to clear maintainers.

**Files changed**: `resource_launchdarkly_ai_tool.go`, potentially `resource_launchdarkly_ai_config.go`

---

## P2 — Design Decision (from PR comments)

### 17. Model config Delete silently ignores "still in use" error

**Location**: `resource_launchdarkly_model_config.go:124-130`

**Problem**: Delete catches the 400 "model config is still in use" error and returns success, dropping the resource from state even though it still exists. If `model_config_key` is set as a literal string (not a Terraform reference), the dependency graph won't order deletion correctly, orphaning the model config.

**Decision**: **(A) Return error with guidance.** When `model_config_key` is a Terraform reference, the graph already orders destruction correctly and this error never fires. The silent-ignore only masks a misconfigured dependency (literal string). Returning the error teaches the user to fix their config. Plan-time validation (Option C) is not feasible — `CustomizeDiff` doesn't run on destroy operations.

**Fix**: Remove the "still in use" catch block and let the error propagate:

```go
if strings.Contains(errMsg, "model config is still in use") {
    return diag.Errorf("failed to delete model config %q in project %q: still in use by one or more AI config variations. Use a Terraform resource reference for model_config_key (not a literal string) so Terraform can order destruction correctly, or delete referencing resources first.", modelConfigKey, projectKey)
}
```

**Files changed**: `resource_launchdarkly_model_config.go`

### 18. Variation Delete silently ignores "Cannot delete the last variation" error

**Location**: `resource_launchdarkly_ai_config_variation.go:192-201`

**Decision**: **(B) Keep current behavior.** Unlike #17, this is an inherent API constraint — the last variation can never be deleted. During `terraform destroy`, the dependency graph destroys the variation before the parent AI config. If we returned an error, `terraform destroy` would fail for any config with a single variation, with no user fix available. The cascade from the parent AI config delete handles cleanup. The edge case (removing a variation block while keeping the parent) is narrow and documentable. Plan-time validation (Option C) is not feasible — `CustomizeDiff` doesn't run on destroy operations.

**Files changed**: None (keep as-is)

---

## PR Comment Checklist

- [x] `is_inverted` GetOk bug — already fixed (Issue #1)
- [x] `evaluation_metric_key` GetOk in update — already fixed (Issue #1)
- [x] #13 — Always set `mode`/`evaluation_metric_key` in aiConfigRead
- [x] #14 — Clear opposite maintainer field in aiConfigRead
- [x] #15 — Clear opposite maintainer field in aiToolRead
- [x] #16 — Fix maintainer unset in ai_tool update + ai_config update (GetOk zero-value)
- [x] #17 — Return error for model config "still in use" delete
- [x] #18 — Keep current behavior (no change needed)
