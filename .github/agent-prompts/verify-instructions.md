You are stage 3 (verification) of the LaunchDarkly Terraform provider autogen
pipeline. A stage-2 agent scaffolded a NEW resource on the currently checked-out
branch and opened a draft PR. Verify it against a REAL LaunchDarkly account
before a human finishes it.

The PR number and title for THIS run are in the "## This run" block prepended
above these instructions — read it FIRST. Wherever these instructions say
"the PR number", use that value (e.g. the verification project is keyed
`tf-verify-pr<PR number>`).

Environment already prepared for you:
- The provider is built from this branch and installed to $GOBIN.
- ~/.terraformrc has a dev_override for "launchdarkly/launchdarkly" pointing
  there, so `terraform plan`/`apply` run THIS build with NO `terraform init`.
- Real LD credentials are in the environment: LAUNCHDARKLY_ACCESS_TOKEN and
  LAUNCHDARKLY_API_HOST=https://app.launchdarkly.com. NEVER print the token.
- `terraform` is on PATH.
- A previous `tf-verify-pr<PR number>` project (if any) was already deleted by a
  workflow step, so the account is clean for a project-scoped apply.

Do this, in order:

1. REVIEW the scaffolded code on this branch. Run `git fetch origin main`
   first, then `git diff origin/main...HEAD` to see only the scaffold, for
   correctness against CLAUDE.md and the vendored playbook at
   .claude/skills/terraform-provider-add-resource/ (SKILL.md + references/).
   Note real defects (schema/CRUD/helper/test/docs issues), not style nits.

2. EXERCISE the resource. Write a minimal terraform config under a temp dir
   (e.g. "$RUNNER_TEMP/verify") with a `provider "launchdarkly" {}` block (it
   reads creds from the environment), a prerequisite project keyed
   `tf-verify-pr<PR number>` (use this key so the live resources are identifiable
   and the workflow can pre-clean it next run), and the new resource (plus its
   data source if one was scaffolded). If the new resource is account-scoped
   rather than project-scoped, there is no project for the workflow to pre-clean,
   so name its key/name per-run as `tf-verify-pr<PR number>-run<the run number>`
   to avoid colliding with a resource retained by a previous run. (If the
   resource is an account SINGLETON, per-run naming cannot prevent a collision —
   only one can exist account-wide — so you will be destroying it in step 4; if a
   leftover from an earlier run still blocks `apply`/`create`, delete that one
   first via a targeted `terraform import` + `destroy`, or report the blocker.)
   Then:
     terraform -chdir="$RUNNER_TEMP/verify" plan  -input=false
     terraform -chdir="$RUNNER_TEMP/verify" apply -input=false -auto-approve
   A second `plan` afterwards MUST be empty (no perpetual diff).

   Then exercise the UPDATE path, not just create: mutate the config (change an
   optional attribute; if the resource has a Map/List/Set nested attribute, ADD
   an element — for a MapNestedAttribute add an entry with the inner `key`
   omitted so it defaults from the map key) and apply + re-plan again; the
   re-plan MUST also be empty. Element-adds to existing resources catch a class
   of plan-consistency bugs (null inner key filled by Read) that create-only
   verification provably misses. Do NOT commit the scratch config or any
   terraform.tfstate*.

   Beta-API caveat: a destroy+recreate at the SAME key can transiently 400
   (e.g. "could not fetch Agent Graph flag ...") while the backend finishes
   deleting the old entity. Retry the apply once after ~30s before treating it
   as a real defect; report it as a flake if the retry succeeds.

3. FIX as needed. If plan/apply or the empty-diff check fails, fix the resource
   implementation/tests/docs on the branch, re-run `go install .` (the dev
   override auto-picks up the rebuilt binary), and retry. Where practical, also
   run the resource's acceptance test:
     TF_ACC=1 TF_ACC_TERRAFORM_PATH="$(command -v terraform)" \
       go test -run <TestAccName> ./launchdarkly/... -v -timeout 30m
   Run `make fmt` and `go build ./...` before committing.

4. RETAIN or DESTROY by scope:
   - PROJECT-scoped resources: RETAIN them (no `terraform destroy`). They sit
     under the namespaced `tf-verify-pr<PR number>` project that the next run
     pre-cleans, so a human can inspect them.
   - ACCOUNT-scoped resources — ESPECIALLY account SINGLETONS (the API allows
     only one per account; a second create 409s): run
     `terraform -chdir="$RUNNER_TEMP/verify" destroy -input=false -auto-approve`
     at the end. A retained singleton has no per-run namespace and permanently
     holds the account's only slot, which blocks BOTH the next verify run and the
     resource's own acceptance test in shared CI (both then 409). `destroy` only
     removes what THIS run created (terraform state) — it never touches unrelated
     account data. Say in the report which resources you retained vs destroyed.

5. COMMIT any fixes to this branch (clear conventional-commit message; touch only
   files relevant to the fix — never the scratch config or state) and push.

   Also append ONE line to `.github/agent-prompts/verify-findings.jsonl` (create
   it if missing) and include it in the commit — unlike verify-result.json this
   ledger IS committed; it is the pipeline's durable record of recurring scaffold
   defects, periodically folded back into scaffold-instructions.md and the
   vendored skill. Exactly one JSON object on one line:
     {"pr": <PR number>, "resource": "<terraform resource name>",
      "defect_categories": [...], "notes": "<one sentence>"}
   defect_categories is a list of short kebab-case labels for the REAL defects
   you found in the scaffold (empty list if none). Reuse labels already in the
   ledger when one fits; otherwise coin a new one in the same style. Examples:
   missing-ci-matrix-entry, docs-not-regenerated, plan-inconsistency-null-inner-key,
   beta-helper-redeclared, missing-update-path, wrong-attribute-type.
   PUBLIC REPO: labels + notes must describe the defect class only — no API
   paths, operationIds, or roadmap detail.

6. REPORT. Post a PR comment on the PR. This repo is PUBLIC, so keep the comment
   to: the verification verdict (did plan/apply succeed? was the re-plan clean?
   which tests ran?), any code fixes you made, and the retained project key
   `tf-verify-pr<PR number>`. Do NOT restate API endpoint paths / operationIds /
   roadmap detail beyond what the diff already shows, and note the LD console link
   is in the private Slack thread. Use:
     gh pr comment <PR number> --body-file <file>

7. Write a machine-readable result to `verify-result.json` at the repo root (do
   NOT commit it) with exactly these fields:
     {"status":"pass"|"fail","project_key":"tf-verify-pr<PR number>",
      "ld_url":"https://app.launchdarkly.com/projects/tf-verify-pr<PR number>",
      "summary":"<one or two sentences: what you verified and any fixes>",
      "defect_categories":[...]}
   status is "pass" only if apply succeeded AND the re-plan was clean.
   defect_categories repeats the same labels you appended to the ledger in
   step 5 (empty list if the scaffold had no real defects).
