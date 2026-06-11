# Code Patterns Reference (plugin framework / v3 line)

This provider migrated to terraform-plugin-framework; inline SDKv2 templates were removed because they no longer compile here. **The patterns live in real files in `launchdarkly/` — read the exemplars below before writing code.** They are current idiom by definition; prefer copying their shape over inventing structure.

## Exemplar map

| Need | Read this file |
|---|---|
| Canonical compact resource (full CRUD, import, state upgrade, null-vs-empty reads) | `launchdarkly/resource_webhook_framework.go` (~300 lines) |
| Project-scoped resource with composite ID, RequiresReplace identity fields | `launchdarkly/resource_metric_framework.go` |
| Data source (separate schema, Required lookup keys, errors on 404) | `launchdarkly/data_source_metric_framework.go` |
| Resource-level `ModifyPlan` (cross-field / upgrade-preserving plan logic) | `launchdarkly/resource_access_token_framework.go` |
| JSON-string attribute (validator + normalize plan modifier) | `launchdarkly/resource_ai_tool_framework.go` |
| Nested attributes at scale (`ListNestedAttribute`, `SingleNestedAttribute`) | `launchdarkly/resource_feature_flag_framework.go` |
| State upgrade from v2.x wire format (`UpgradeState`, null-vs-empty fixes) | `launchdarkly/resource_webhook_framework.go`, `launchdarkly/resource_metric_upgrade.go` |
| Shared conversion helpers (use, don't reinvent) | `launchdarkly/framework_helpers.go` |
| Validators | `launchdarkly/framework_validators.go`, `launchdarkly/framework_json_helpers.go` |
| Patch helpers + API error unwrapping | `launchdarkly/helper.go` (`patchReplace`/`patchAdd`/`patchRemove`, `handleLdapiErr`, `isStatusNotFound`) |
| Acceptance test shape (steps, CheckDestroy, import verify) | `launchdarkly/resource_launchdarkly_metric_test.go` |
| Data source test shape (API-scaffolded fixtures) | `launchdarkly/data_source_launchdarkly_metric_test.go` |

## Skeleton (shape only — copy real code from the exemplars)

```go
var (
    _ resource.Resource                = &<Name>Resource{}
    _ resource.ResourceWithImportState = &<Name>Resource{}
)

type <Name>Resource struct{ client *Client }

type <Name>ResourceModel struct {
    ID         types.String `tfsdk:"id"`
    ProjectKey types.String `tfsdk:"project_key"`
    // ... types.* fields with tfsdk tags matching keys.go constants
}

func New<Name>Resource() resource.Resource { return &<Name>Resource{} }

func (r *<Name>Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_<name>"
}

func (r *<Name>Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    r.client = configureResourceClient(req, resp)
}
// Schema, Create, Read, Update, Delete, ImportState — copy shape from resource_webhook_framework.go
```

## API call wrapper

Every API call goes through the concurrency limiter:

```go
var out *ldapi.<Type>
var res *http.Response
err := r.client.withConcurrency(r.client.ctx, func() error {
    var e error
    out, res, e = r.client.ld.<Api>.Get<Name>(r.client.ctx, projectKey, key).Execute()
    return e
})
if err != nil {
    if isStatusNotFound(res) { /* resource read: null the ID + RemoveResource; data source: AddError */ }
    addLdapiError(&resp.Diagnostics, "Failed to get <name>", err)
    return
}
```

Beta endpoints: same shape with `.LDAPIVersion("beta")` chained before `.Execute()`; the provider `Client` also exposes a beta-configured client via `newBetaClient` plumbing — check `config.go`.

## Update via JSON patch

```go
patch := []ldapi.PatchOperation{
    patchReplace("/name", &name),
}
if !plan.Statements.Equal(state.Statements) {
    if len(stmts) > 0 {
        patch = append(patch, patchReplace("/statements", &stmts))
    } else {
        patch = append(patch, patchRemove("/statements"))
    }
}
```

Compare `plan.X.Equal(state.X)` to gate optional patch ops (the framework replacement for SDKv2 `d.HasChange`).

## Null-vs-empty on read (plan-apply consistency)

For **Optional** (non-Computed) attributes the read must distinguish "user omitted" (null) from "user wrote `[]`/`\"\"`" (empty), or Terraform fails with "Provider produced inconsistent result after apply". Pick from `framework_helpers.go`:

- `stringValueOrNullFromPointer` — Optional string from API pointer; empty → null.
- `setFromStringSliceOrNull` — Optional set; empty → null.
- `setFromStringSlicePreservingPlan` / `listFromStringSlicePreservingPlan` — Optional collection where explicit `attr = []` must survive as empty; pass the existing model value so the read preserves the planned shape.

See the read functions in `resource_webhook_framework.go` for all three in use.

## Version-advancement retry (versioned entities)

For APIs that create a new version per update (e.g. AI Config variations): record the entity's `version` before the write, then poll the read until the version advances, with a max-attempt cap. Never trust `items[0]` to be latest — select the highest `Version` from `Items[]`. Use a fixed `time.Sleep` only when no version field exists. Find the existing loop with `grep -rn "version" launchdarkly/resource_ai_config_variation_framework.go`.

## Transient delete retry

Some endpoints return non-deterministic 4xx on delete that succeed on retry. Only add a retry **after observing** the failure mode: match `res.StatusCode` plus a narrow body substring via `handleLdapiErr`, wrap in a bounded retry (~45s overall). Never add preemptively; a broad substring match hides real validation errors.

## Last-child cascade delete

On "cannot delete the last `<child>`" errors during destroy, log a warning and return cleanly — the parent's delete cascades. On "still in use", error with guidance to use Terraform references so the graph orders destruction. Always early-return on `isStatusNotFound(res)`.

## Acceptance test scaffold

Shape (full version in `resource_launchdarkly_metric_test.go`):

```go
func TestAcc<Name>_CreateUpdate(t *testing.T) {
    projectKey := acctest.RandStringFromCharSet(16, acctest.CharSetAlpha)
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        CheckDestroy:             testAcc<Name>Destroy,
        Steps: []resource.TestStep{
            { Config: create, Check: ... },
            { ResourceName: "launchdarkly_<name>.test", ImportState: true, ImportStateVerify: true },
            { Config: update, Check: ... },
        },
    })
}
```

- Sequential `resource.Test`, never `ParallelTest`.
- Test HCL uses **nested-attribute syntax** (`field = [{ ... }]`), not blocks.
- `CheckDestroy` verifies a 404; skip `data.` addresses in shared checkers.
- Account-singleton resources: PreCheck hook that deletes the test's target identifiers up front (LD returns `409 optimistic_locking_error` for duplicates, retries don't help).

## Doc template (only if custom layout needed)

`templates/resources/<name>.md.tmpl` — `tfplugindocs` syntax; rendered by `make generate`. Most resources don't need one; schema `Description` + `examples/` files generate everything.
