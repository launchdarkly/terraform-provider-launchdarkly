# Phase 5 cutover checklist

> Source: `.claude/MIGRATION_PLAN_NON_BREAKING.md` §Phase 5.
> Status: scaffold. The mux remains in `main.go` until every Phase 2-4
> resource has shipped via the framework provider and soaked on the
> moonshots integration branch.

## 5.1 — Remove SDKv2 provider registration

Prereq: every resource and data source registered on the SDKv2 mux
side has a framework counterpart in `plugin_provider.go::Resources()`
or `DataSources()`. Verify:

```bash
# SDKv2 still serving:
grep -E '^\s+"launchdarkly_' launchdarkly/provider.go | grep -v '//' | wc -l
# Should be 0 before this step.
```

Then:

1. Delete `Provider()` in `launchdarkly/provider.go` (the SDKv2 entry
   point) plus the helper closures it depends on.
2. Edit `main.go`:
   ```go
   func main() {
       debug := flag.Bool("debug", false, "Start provider in debug mode.")
       flag.Parse()

       err := providerserver.Serve(context.Background(),
           launchdarkly.NewPluginProvider(version),
           providerserver.ServeOpts{
               Address: "registry.terraform.io/launchdarkly/launchdarkly",
               Debug:   *debug,
           },
       )
       if err != nil {
           log.Fatal(err)
       }
   }
   ```
3. Remove imports of `terraform-plugin-go/tfprotov5/tf5server` and
   `terraform-plugin-mux/tf5muxserver` from `main.go`.
4. Run `go mod tidy`.

## 5.1a — Unify test packages

Once `Provider()` is gone, the `launchdarkly/tests/` sub-package's mux
factory breaks (it calls `launchdarkly.Provider()`). Two options
documented in the master plan; **recommended option 1**: collapse
`tests/` back into the root `launchdarkly` package now that there's
only one provider.

- Move `launchdarkly/tests/resource_team_role_mapping_test.go` to
  `launchdarkly/`.
- Move `launchdarkly/tests/provider_test.go` factory into root
  `provider_test.go` (or delete it; root's
  `testAccProtoV5ProviderFactories` is now framework-only and
  serves both packages).
- Delete the `launchdarkly/tests/` directory.

Triage the 10 helper-test files that import `terraform-plugin-sdk/v2`
directly (`ai_config_variation_helper_test.go`,
`clause_helper_test.go`, etc.). For each:

- Rewrite against framework types **if** the underlying helper is
  still in use.
- Delete **if** the helper itself is dead code post-SDKv2 drop.

Two known shim functions that retain SDKv2 schema references purely
for the tests: `resourceCustomRole()` and `resourceAccessToken()` /
`validateAccessTokenResource()`. Delete these alongside the test
files that use them.

## 5.2 — Drop SDKv2 module

```bash
go mod edit -dropreplace github.com/hashicorp/terraform-plugin-sdk/v2
# remove the require line
go mod tidy
```

Cleanup:

- `helper.go::removeInvalidFieldsForDataSource` becomes dead code.
- All SDKv2 validators in `validation_helper.go` are dead.
- `cty_helpers.go`, `schema_compat.go`, `embedded_schema_compat_test.go`
  — keep or delete based on the Phase 0.6 Upjet decision
  (`docs/migration-schema-compat-upjet.md`).
- `tests/` directory removal completes here.

## 5.3 — Retire keys.go (optional)

Decision pending per Phase 0 open question. If retired:

- Replace `KEY` / `NAME` / etc. constant references with string
  literals inside `tfsdk:` tags only.
- Remove `gofmts` from `make fmt` if no other consumers remain.

## 5.4 — Protocol v5 -> v6

```go
// main.go
err := providerserver.Serve(ctx, launchdarkly.NewPluginProvider(version),
    providerserver.ServeOpts{
        Address:         "registry.terraform.io/launchdarkly/launchdarkly",
        Debug:           *debug,
        ProtocolVersion: 6, // bump from default 5
    },
)
```

Release notes must state the **minimum supported Terraform CLI version**
explicitly (decision locked in Phase 0.9a: v2.x may raise the floor
and does not need to preserve `<1.0` compatibility).

## 5.5 — Test harness rename

```bash
# Mechanical rename across all _test.go files:
sed -i '' 's|ProtoV5ProviderFactories: testAccProtoV5ProviderFactories|ProtoV6ProviderFactories: testAccProtoV6ProviderFactories|g' \
    launchdarkly/*_test.go
# Update provider_test.go to construct the v6 factory:
#   testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){...}
```

## Done-when

- `go.mod` has no reference to `terraform-plugin-sdk/v2` or
  `terraform-plugin-mux`.
- `make build && make test && make testacc` green.
- One downstream consumer (Crossplane Upjet, terraform-modules)
  smoke-tested against the new binary.
- Release notes published with the protocol-v6 minimum-CLI floor.

## Risk callouts

- Removing SDKv2 mid-soak strands any Phase 4 resource that hasn't
  promoted yet. **Don't run Phase 5 until every Phase 2-4 PR has
  merged to the moonshots branch and that branch has promoted to
  `main`.**
- The protocol v5 -> v6 flip is irreversible without a major bump.
  Coordinate with downstream consumers (Crossplane, terraform-modules)
  before flipping.
- `keys.go` retirement (5.3) is optional; the in-flight constants
  reduce diff size for migration PRs. Defer unless there's an
  ergonomic win.
