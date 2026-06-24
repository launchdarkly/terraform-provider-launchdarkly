---
name: terraform-provider-v3-migration
description: >
  Migrate a full LaunchDarkly Terraform v2.x setup to v3 end-to-end, and try/verify the v3 provider's
  new resources against a real LaunchDarkly account. Use when the user wants to (1) upgrade an entire
  v2.x configuration to v3 — run migrate-tf-syntax, fix the manual follow-ups, build the v3 provider,
  apply, and confirm an idempotent state upgrade with zero forced replacements; or (2) exercise and
  validate the new v3 resources (context_kind, announcement, oauth_client, metric_group, release_policy,
  big_segment_store_integration, flag_import_configuration, integration_delivery_configuration) with
  create/read/update/delete and idempotency checks before calling them GA. Triggers: "migrate a full v2
  setup", "end-to-end v3 upgrade", "verify the new v3 resources", "test the scaffolded resources",
  "validate v3 before GA", "exercise the beta resources".
compatibility: >
  Run from within the terraform-provider-launchdarkly repository on the v3 (plugin-framework) line.
  Needs Go (version per .go-version), a real `terraform` CLI (>= 1.0), and `make` on PATH. Workflow B
  and the apply step of Workflow A hit a real LaunchDarkly account and require a LAUNCHDARKLY_ACCESS_TOKEN
  for an enterprise account.
metadata:
  author: launchdarkly
  version: "1.0.0"
---

# LaunchDarkly Terraform provider: v3 migration and new-resource verification

This skill drives two end-to-end workflows for the v3 line of the provider:

- **Workflow A — Migrate a full v2.x setup to v3.** Convert a complete v2 configuration to v3 syntax, build the v3 provider, apply it against existing state, and confirm the upgrade is idempotent and destroys nothing.
- **Workflow B — Try and verify the new v3 resources.** Exercise each net-new v3 resource against a real LaunchDarkly account with a create, read, update, and delete cycle, and report a pass/blocked/fail matrix.

The two workflows share a provider build and the same state-hygiene rules. Run them independently or back to back.

## When to use

- The user wants to upgrade an entire v2.x project to v3, not just convert one snippet.
- The user wants to confirm the v3 state upgrade is clean on a representative configuration.
- The user wants to validate the new v3 resources, several of which were scaffolded by the autogen pipeline and have not yet been exercised against a real account.
- The user asks whether a specific new resource works end to end.

## Do NOT use

- For converting a single HCL snippet between block and nested-attribute syntax. Use the `terraform-provider-block-to-nested-attrs` skill, which this skill calls into for the mechanical rewrite.
- For implementing a brand-new resource in the provider source. Use the `terraform-provider-add-resource` skill.
- For a quick local build with no migration or verification goal. Use the `terraform-provider-local-testing` skill directly.

## Prerequisites (shared)

1. **Build the v3 provider and point Terraform at it.** Follow the `terraform-provider-local-testing` skill: `make build` installs into `$GOPATH/bin`, and a `~/.terraformrc` `dev_overrides` block points at that directory. `terraform init` is skipped under dev overrides, so rebuild with `make build` between provider changes.
   - **Pre-flight the override path every time.** A stale override silently runs an old binary and masks everything — this misconfiguration is common after a Go-version bump. Confirm the two paths match before building:
     ```bash
     grep launchdarkly ~/.terraformrc   # the dev_override target
     echo "$(go env GOPATH)/bin"        # where `make build` installs — MUST match the line above
     ```
     If they differ, fix `~/.terraformrc` first, then `make build`.
2. **Authenticate.** Export `LAUNCHDARKLY_ACCESS_TOKEN`. Confirm `LAUNCHDARKLY_API_HOST` matches the account the token belongs to. A `401 "Invalid key"` usually means the host points at the wrong environment, not a bad token. Note: an explicit `api_host` argument in the config's `provider "launchdarkly"` block overrides the `LAUNCHDARKLY_API_HOST` env var, so read `provider.tf` before assuming the env controls the host (for example, a blitz prod token needs `app.launchdarkly.com` even when the env points at staging).
3. **Keep scratch state out of git.** Work inside `local-testing/`. Its `terraform.tfstate*` files are throwaway and must never be committed.

## Workflow A — Migrate a full v2.x setup to v3

The goal is a clean upgrade: the first v3 plan shows only cosmetic drift, the apply succeeds, the follow-up plan is empty, and nothing is replaced.

### A1. Prepare the v2 configuration

- For a customer setup, copy their `.tf` files into a scratch directory under `local-testing/`.
- For an internal end-to-end test, the genuine v2.29 **block-syntax** setup is in the backup archive `local-testing/full-account-v2.29-backup-*.zip`. Extract it to a scratch dir and use that as the v2 source. The on-disk `local-testing/full-account-v2.29/` and `full-account-v2.29.original/` dirs are **both already v3 nested syntax** (last applied with provider 3.0.0-beta.1) — they are not v2-block sources. To regenerate a block-syntax config from a v3 one, run the tool with `-direction v3-to-v2` first.
- Establish a clean baseline. Against the v2 provider, `terraform apply` until the plan is empty. The migration starts from a state with no pending changes.

### A2. Convert the syntax with migrate-tf-syntax

Run the tool that ships in this repo. It is deterministic and rewrites every affected attribute, and it updates the removed attributes.

`migrate-tf-syntax` is its **own Go module**, so run it with `go -C` (running `go run ./scripts/migrate-tf-syntax` from the repo root fails with `main module does not contain package`). Pass an **absolute** `-dir`.

```bash
DIR="$PWD/local-testing/your-config"

# Dry run first. The tool prints every file it intends to change.
go -C scripts/migrate-tf-syntax run . -dir "$DIR" -direction v2-to-v3 -dry-run

# Apply the rewrite. Add -recursive to convert locally vendored modules in the same pass.
go -C scripts/migrate-tf-syntax run . -dir "$DIR" -direction v2-to-v3

# Normalize whitespace.
terraform -chdir="$DIR" fmt
```

The tool converts block syntax to nested-attribute syntax, drops `expire` and `is_active`, renames `policy_statements` to `inline_roles`, converts `policy` to `policy_statements`, rewrites `include_in_snippet` into the matching client-side-availability attribute, and updates data-source references such as `client_side_availability` on the `launchdarkly_project` data source.

This also covers upgrades from an **early v3 preview to a later v3 build** (for example 3.0.0-beta.1 to GA). A preview config has no block syntax to convert, but it may still set an attribute that a later v3 removed (for example `policy` on `launchdarkly_custom_role`), which fails `terraform plan` with `Unsupported argument`. The tool drops or renames those on already-nested input too, so run it for preview-to-GA upgrades as well.

### A3. Finish the follow-ups the tool cannot do

Read the `terraform-provider-block-to-nested-attrs` skill for the full gotcha list. The tool leaves these for you:

- **Newly required attributes.** `launchdarkly_feature_flag` requires `variations` for every variation type, including boolean. The tool now synthesizes these automatically for boolean flags whose `variation_type` is the literal `"boolean"`, so you usually do not add them. One case still needs you: `variation_type` is a non-literal expression (a var/local) — the tool warns and skips, so add `variations` by hand. Variation `name`/`description` set outside Terraform are preserved by the provider when the config omits them, so no action is needed there. For a multivariate flag missing variations the parse error is `Missing required argument`.
- **`dynamic` blocks.** The tool warns and skips them. Rewrite each as a for expression, for example `variations = [for v in var.values : { value = v }]`.
- **Remote modules.** Registry- or git-sourced modules are out of scope. Upgrade those at their source.

Run `terraform validate` as the inner loop until the configuration parses. It checks schema shape without hitting the API.

### A4. Apply the v3 provider

1. Update the provider version constraint to `~> 3.0`.
2. `make build` to install the v3 binary, with the dev override in place.
3. `terraform plan`. Confirm the plan matches the expected drift in the "Known drift" section below. **A forced replacement is a bug, not cosmetic drift — stop and investigate.**
4. `terraform apply`. The provider runs its state upgraders automatically. You do not edit state by hand.
5. `terraform plan` again. It must report `No changes`.

### A5. Optional rollback check

To confirm a safe downgrade path, run the inverse conversion and validate against the v2 provider:

```bash
go -C scripts/migrate-tf-syntax run . -dir "$PWD/local-testing/your-config" -direction v3-to-v2
```

The inverse appends reversed blocks at the end of each body, so attribute order shifts. `terraform fmt` normalizes whitespace but not order.

### A — Definition of done

- The configuration parses and applies against the v3 provider.
- The first plan showed only cosmetic `[] -> null` and known-after-apply drift, with zero forced replacements.
- The follow-up plan is empty.

## Workflow B — Try and verify the new v3 resources

The net-new v3 resources are listed with their stability, scope, example path, and known gotchas in [references/new-resources.md](references/new-resources.md). Seven of the eight resources were added as autogen scaffolds after the `v3.0.0-beta.2` preview and have not been exercised in a tagged release, so this workflow is the gate before treating them as GA.

### B1. Stand up a scratch project

Most new resources are project scoped. Create one throwaway `launchdarkly_project` with a `production` environment to host them, so cleanup is a single `terraform destroy`. Account-scoped resources (`announcement`, `oauth_client`) do not need it.

### B2. Exercise each resource

For each resource in the matrix:

1. Read `examples/resources/launchdarkly_<name>/resource.tf` for the canonical config. This is the source of truth the docs render from. Adapt keys to your scratch project.
2. If the example references attributes you are unsure of, confirm them against the schema in `launchdarkly/resource_<name>_framework.go` and the rendered `docs/resources/<name>.md`.
3. `terraform apply`. Confirm the create succeeds.
4. `terraform plan`. It must report `No changes`. A non-empty plan right after create is a read/round-trip bug — capture the diff.
5. Change one mutable attribute, `apply`, and confirm the update path works.
6. `terraform destroy`. Confirm the delete succeeds and a follow-up plan is clean.
7. Record the outcome using the classification below.

### B3. Handle the credential- and entitlement-gated resources

Several beta resources integrate with external systems or gated account features:

- `big_segment_store_integration` needs reachable store credentials, for example a Redis or DynamoDB store.
- `flag_import_configuration` and `integration_delivery_configuration` need valid third-party credentials for the chosen integration, for example Split or Fastly.
- Any beta resource may require an account entitlement that the test account lacks.

When the only blocker is a missing credential or entitlement, classify the resource **BLOCKED**, not **FAIL**. Record exactly what is missing so it can be retried on an entitled account.

### Result classification

Report every resource as one of:

- **PASS** — create, read with an empty follow-up plan, update, and delete all succeeded.
- **BLOCKED** — the workflow could not run because of a missing credential, entitlement, or external dependency. Not a provider defect. Record what is needed.
- **FAIL** — the provider returned an error, produced perpetual drift, forced an unexpected replacement, or left an orphan after destroy. Capture the error and the resource address.

Present the results as a table: resource, stability, outcome, and a one-line note.

## Known drift and quirks

These are expected and documented in `.claude/V3_PRERELEASE_NOTES.md`. Do not flag them as failures:

- **`[] -> null` cosmetic drift on the first plan after upgrade.** v2 stored empty lists where v3 stores null. Applies once, cleanly, with no replacement.
- **Known-after-apply on mutually-exclusive computed pairs**, for example `maintainer_id` and `maintainer_team_key` on `launchdarkly_feature_flag`, and `model` and `model_config_key` on `launchdarkly_ai_config_variation`.
- **Versioned AI Config variations.** `launchdarkly_ai_config_variation` recomputes `version`, `creation_date`, and `variation_id` on every update, because the API models variations as versioned entities.
- **Write-only AI Config variation fields.** `description`, `instructions`, and `tool_keys` are settable but not returned by the API read. Out-of-band changes are not detected.
- **One announcement per account.** `launchdarkly_announcement` returns `409` if one already exists. Treat this as a singleton: clean up an existing announcement before creating a new one, following the orphan-cleanup precheck pattern used for other account-singleton resources.

## Cleanup and state hygiene

- `terraform destroy` every resource you created. Confirm a follow-up plan is clean.
- For account-singleton resources, if a prior run left an orphan, delete it by its known identifier before the next run. Retries do not help, because LaunchDarkly reuses `409 optimistic_locking_error` for both genuine races and duplicate rejection.
- Never commit `local-testing/terraform.tfstate*` or scratch `.tf` files.

## Related skills and references

- `terraform-provider-block-to-nested-attrs` — the mechanical block-to-nested-attribute rewrite and the full gotcha list. Workflow A step A3 depends on it.
- `terraform-provider-local-testing` — the dev-override build flow, scratch configs, and state hygiene. Both workflows depend on it.
- `terraform-provider-add-resource` — for changing provider source, not for using or verifying resources.
- `docs/guides/migrating-to-v3.md` — the customer-facing migration guide.
- `docs/guides/v3-release-notes.md` — the v3.0.0 release notes, including the removed-attribute table.
- `scripts/migrate-tf-syntax/` — the conversion tool and its embedded mapping.
- `references/new-resources.md` — the new-resource verification matrix.
