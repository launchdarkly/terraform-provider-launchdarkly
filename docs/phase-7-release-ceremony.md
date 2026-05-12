# Phase 7 — Release ceremony

> Source: `.claude/MIGRATION_PLAN_NON_BREAKING.md` §Phase 7.
> Status: scaffold. Each completed phase triggers one execution of
> this checklist when promoting `moonshots/terraform-plugin-framework`
> -> `main`.

## Promotion cadence

| Phase batch | Target v2.x minor | Soak period |
|---|---|---|
| Phase 0 alone | does **not** ship to `main`; lives only on the moonshots branch | n/a |
| Phase 1 (all 19 data sources) | v2.30.0 | 1-2 weeks of nightly testacc on moonshots |
| Phase 2 (all 9 leaf resources) | v2.31.0 | 1-2 weeks |
| Phase 3 (all 10 medium resources) | v2.32.0 | 1-2 weeks |
| Phase 4.1 `launchdarkly_project` | v2.33.0 | 2 weeks (highest risk; customizeProjectDiff) |
| Phase 4.2 `launchdarkly_segment` | v2.34.0 | 1-2 weeks |
| Phase 4.3 `launchdarkly_feature_flag` | v2.35.0 | 2 weeks |
| Phase 4.4 `launchdarkly_feature_flag_environment` | v2.36.0 | 2 weeks |
| Phase 5 (cutover) | v2.37.0 | 1-2 weeks; coordinate Crossplane + terraform-modules |
| Phase 6.x | v2.38.0+ (rolling, additive) | per-feature |

## Per-promotion checklist

### Soak

- [ ] `make testacc` green on the moonshots branch every night for the
      soak period.
- [ ] Apply -> destroy lifecycle on a tester LD project covers every
      resource that's part of the promotion batch.
- [ ] State-upgrade soak: a v2.29-applied state file produces zero
      diff on `terraform plan` against the moonshots build for every
      resource in the batch.

### Promote

- [ ] Squash-merge `moonshots/terraform-plugin-framework` -> `main`
      (or cherry-pick the phase's commit range).
- [ ] Verify `make generate` produces no diff post-promotion.
- [ ] Release-please picks up the `refactor:` commits and opens the
      v2.x.y minor release PR.
- [ ] Review the auto-generated CHANGELOG entry. Add a "Migration
      notes" subsection explicitly stating this batch is internal-SDK-
      swap only (no user-facing breaking change).

### Verify downstream

- [ ] Crossplane Upjet rebuilds clean against the framework schemas
      (Phase 0.6 de-risked but verify on each promotion).
- [ ] `terraform-modules` smoke-tested.
- [ ] Internal `local-testing/` dev-override run confirms binary
      installs and applies against a real LD account.

### Communicate

- [ ] CHANGELOG.md describes the internal SDK swap for users.
- [ ] No migration-guide artefact needed (no breaking changes).
- [ ] Tag the release; release-please pushes the binary to the
      registry.

## Phase 5.4 protocol v6 special-case

When the v6 cutover ships (after Phase 4.4 promotion, batched with
Phase 5):

- [ ] Release notes for the v6-cutover minor (e.g. v2.37.0) must
      state the **minimum supported Terraform CLI version** explicitly.
      Decision locked in Phase 0.9a: v2.x may raise the floor and does
      not need to preserve `<1.0` compatibility.
- [ ] Announce the floor in the GitHub release description.
- [ ] Notify Crossplane provider-launchdarkly maintainers so they can
      rebuild against the v6 ABI.

## Risks per promotion

- **Phase 1 -> v2.30.0**: low. Data sources have no state round-trip.
- **Phase 2 -> v2.31.0**: medium. State-fixture parity required for
  every resource; deprecated `policy_statements` / `expire` /
  `include_in_snippet` must carry forward verbatim.
- **Phase 3 -> v2.32.0**: medium. CustomizeDiff -> ModifyPlan ports;
  schema-version upgraders must round-trip cleanly.
- **Phase 4.1 -> v2.33.0**: HIGH. project customizeProjectDiff edge
  cases (IIS / CSA / view-association). Plan an extended soak.
- **Phase 4.3-4.4**: HIGH. feature_flag variation customPropertyHash
  parity + FFE 1300-LOC test suite must pass unchanged.
- **Phase 5 -> v2.37.0**: HIGH. Protocol v6 cutover is irreversible
  without a major bump. Coordinate downstream.
