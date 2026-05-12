**Wait!**

- Have you added comprehensive tests?
- Have you updated relevant data sources as well as resources?
- Have you updated the docs?

**Testing**

For any changes you make, please ensure your acceptance test conform to the following:

- every single new attribute is tested
- optional attributes revert to null or default values if removed
- attributes that interact interact as expected
- block values behave as expected when reordered
- nested fields on maps or list/set items function as expected. Terraform does not actually enforce most schema attributes on nested items
- each test step for a configuration is followed by a test step where `ImportState` and `ImportStateVerify` are set to true. `ImportStateVerifyIgnore` can be used if we explicitly _expect_ a value to be different when imported, such as in the case of obfuscated values like API keys

## Migration metadata (skip if this is not a SDKv2 -> framework migration PR)

- [ ] Linked to phase: <!-- e.g. 2.3 -->, see `.claude/MIGRATION_PLAN_NON_BREAKING.md`
- [ ] Base branch is `moonshots/terraform-plugin-framework` (not `main`)
- [ ] Branch name follows `moonshots/tpf/<phase-id>-<slug>`
- [ ] Block schemas (`schema.Blocks`) used for any structure that was a block in SDKv2 — not nested attributes
- [ ] All `Required`/`Optional`/`Computed`/`ForceNew`/`Default`/`ConflictsWith`/`Deprecated:` flags preserved verbatim
- [ ] `tfsdk:` tags on the model struct match `launchdarkly/keys.go` constants exactly
- [ ] State-fixture test added or updated under `launchdarkly/testdata/state-fixtures/` and exercised from `launchdarkly/statecompat/`
- [ ] `make generate` produces no diff
- [ ] Per-PR checklist from `MIGRATION_PLAN_NON_BREAKING.md` §Per-PR checklist satisfied
- [ ] Conventional Commit prefix is `refactor:` (no user-visible behaviour change). Never `feat!:` or `BREAKING CHANGE:`.

## LaunchDarkly Employees

For more information on how to build, test, and release, see the [internal provider runbook](https://launchdarkly.atlassian.net/wiki/spaces/PD/pages/3825598468/LaunchDarkly+Terraform+Provider+Runbook).
