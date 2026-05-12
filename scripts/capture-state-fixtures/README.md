# State fixture capture + safety policy

State-compat fixtures back the regression harness in
`launchdarkly/statecompat/`. Each Phase 2-4 migration PR (per
`.claude/MIGRATION_PLAN_NON_BREAKING.md`) lands a fixture for the
resource being moved from SDKv2 to terraform-plugin-framework so the
harness can prove `v2.29 state -> framework plan` produces zero diff.

## What this directory contains

| File | Purpose |
|---|---|
| `capture.sh` | One-shot capture flow: apply a synthetic config, sanitise the resulting state, drop it into `launchdarkly/testdata/state-fixtures/`, tear down. |
| `scan.sh` | Fixture-safety regex scan. CI runs this on every push; it fails the build if any committed fixture contains a real-looking token / SDK key / mobile key. |
| `safe-placeholders.txt` | Allowlist of known-safe placeholder values that fixtures may legitimately contain. |
| `configs/` _(populated per-PR)_ | Synthetic `.tf` configs that map 1:1 to fixture filenames. `configs/relay_proxy_basic.tf` -> `relay_proxy_basic.tfstate`. |

## Fixture-safety policy (locked)

Per the master plan §Phase 0.5, fixtures committed to the repo:

1. **MUST be generated from synthetic configs** under
   `scripts/capture-state-fixtures/configs/`. Real production state is
   forbidden regardless of sanitisation claims — the `access_token`
   resource stores token secrets in plaintext state, and other resources
   carry production identifiers (project keys, environment SDK keys,
   user emails, team names) that we don't want in git history.
2. **MUST use deterministic synthetic identifiers** (`fixture-project-1`,
   `fixture-token-PLACEHOLDER`, etc.) so capturing the same config twice
   produces byte-identical fixtures.
3. **MUST pass `scan.sh`** before being committed. CI enforces this;
   `.githooks/pre-commit` runs the same script locally as best-effort
   developer DX.
4. **MUST NOT bypass the allowlist.** If `scan.sh` flags a substring
   that's legitimately safe, add it to `safe-placeholders.txt` with a
   short justification comment — don't widen the regex.

## Capturing a new fixture

```bash
# 1. Write a synthetic .tf config in configs/
#    Use placeholder values from safe-placeholders.txt for any
#    tokens / IDs the resource needs.
$EDITOR scripts/capture-state-fixtures/configs/my_new_fixture.tf

# 2. Run capture.sh — it applies the config against your LD test
#    account, sanitises the state, drops it in testdata/state-fixtures/,
#    tears down, and re-runs scan.sh.
LAUNCHDARKLY_ACCESS_TOKEN=<test-account-token> \
  ./scripts/capture-state-fixtures/capture.sh my_new_fixture

# 3. Reference the fixture from a state-compat test in launchdarkly/statecompat/:
#    statecompat.Run(t, statecompat.Case{
#        HCLConfig: myConfigHCL,
#        FixtureFile: "my_new_fixture.tfstate",
#        PreviousVersion: "2.29.0",
#        ProtoV5ProviderFactories: testAccProtoV5ProviderFactories,
#        PreCheck: func() { testAccPreCheck(t) },
#    })

# 4. Commit the .tf config + the .tfstate fixture together. CI's
#    fixture-safety scan runs on every push; if scan.sh fails, fix the
#    fixture before merging.
```

## Sanitiser jq script

`sanitize.jq` is intentionally not committed yet — Phase 0.5 lands the
policy + harness + scan. Each Phase 2-4 PR that captures a fixture for a
particular resource owns the per-resource jq logic, because the
identifiers to scrub differ by resource shape (a project fixture scrubs
different paths than a webhook fixture). When the first migration PR
lands `sanitize.jq`, this README will be updated to describe the
convention.

## Where the harness lives

The Go-side wire-compat regression harness is in
`launchdarkly/statecompat/`. It uses
`terraform-plugin-testing/helper/resource` (the modern testing API)
plus `plancheck.ExpectEmptyPlan()` to assert zero diff after the
SDKv2 -> framework swap. It lives in its own package because
`terraform-plugin-sdk/v2/helper/resource` and
`terraform-plugin-testing/helper/resource` both register a `sweep` flag
in `init()` — importing both into one test binary panics. See
`launchdarkly/statecompat/harness.go` for the entry point.

## CI integration

`.github/workflows/test.yml` invokes `scan.sh` as a `build`-job step on
every push and pull request. The scan runs in <2s even with hundreds of
fixtures and exits 1 on the first violation, blocking the merge.

## Allowlist hygiene

Every entry in `safe-placeholders.txt` weakens the scan a little.
Reviewers should push back on PRs that grow the allowlist without a
matching justification comment naming the fixture + reason. The bar:
each placeholder must be a *deterministic* string the capture script
produces, not a sanitisation artefact you can fix.
