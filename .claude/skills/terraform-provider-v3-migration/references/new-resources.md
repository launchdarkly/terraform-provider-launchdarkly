# New v3 resources: verification matrix

These are the resources and data sources that v3 adds over the latest v2 line. v3 removes none. Use this matrix with Workflow B in the parent `SKILL.md`.

Only `launchdarkly_context_kind` shipped in a tagged preview (`v3.0.0-beta.2`). The other seven resources were added as autogen scaffolds after `beta.2`, so this matrix and Workflow B are the gate before treating them as GA. The canonical config for each is `examples/resources/launchdarkly_<name>/resource.tf` — read it at verification time rather than trusting the summaries below, since scaffolds can change.

| Resource | Data source | Stability | Scope | Origin |
|---|---|---|---|---|
| `launchdarkly_context_kind` | yes | Stable API | Project | beta.2 |
| `launchdarkly_announcement` | no | Stable API | Account | scaffold #460 |
| `launchdarkly_oauth_client` | yes | Stable API | Account | scaffold #466 |
| `launchdarkly_metric_group` | yes | Beta API | Project | scaffold #453 |
| `launchdarkly_release_policy` | yes | Beta API | Project | scaffold #471 |
| `launchdarkly_big_segment_store_integration` | yes | Beta API | Project + environment | scaffold #468 |
| `launchdarkly_flag_import_configuration` | yes | Beta API | Project | scaffold #469 |
| `launchdarkly_integration_delivery_configuration` | yes | Beta API | Project + environment | scaffold #467 |

"Beta API" means the resource calls a LaunchDarkly beta endpoint through the provider's beta client. Those endpoints can change without notice and may require an account entitlement.

## Verification status (last real-account run: 2026-07-02, blitz prod, ffeldberg/v3-rc-prep RC build)

| Resource | Result |
|---|---|
| `launchdarkly_context_kind` | PASS (full CRUD + data source) |
| `launchdarkly_announcement` | PASS (full CRUD; singleton orphan deleted first) |
| `launchdarkly_oauth_client` | PASS (full CRUD + data source) |
| `launchdarkly_ai_agent_graph` | PASS (full CRUD incl. edge-handoff update + data source) |
| `launchdarkly_metric_group` | PASS (full CRUD incl. funnel entry rename + data source) |
| `launchdarkly_release_policy` | PASS (both release methods, full CRUD + data source) |
| `launchdarkly_big_segment_store_integration` | PASS (full CRUD + data source; API accepts config without connectivity validation, so dummy store credentials with `on = false` suffice) |
| `launchdarkly_flag_import_configuration` | PASS (full CRUD + data source with dummy Split credentials — the API stores config unvalidated) |
| `launchdarkly_integration_delivery_configuration` | PASS (full CRUD + data source with dummy Fastly credentials, `on = false`) |

No read round-trip bugs or forced replacements. All eight data sources verified against their live resources. One example defect found and fixed: the `ai_agent_graph` example omitted `mode = "agent"` on its AI configs — the API rejects `completion`-mode configs as graph nodes with `400 invalid_request`. Note `mode` is immutable on `launchdarkly_ai_config` (changing it plans a replace).

The three integration resources do NOT need live third-party credentials for CRUD verification — the LaunchDarkly API persists `config` without validating connectivity. Live credentials only matter for verifying the integration actually functions, which is out of scope for provider CRUD testing.

## Per-resource notes

### launchdarkly_context_kind — Stable, project scoped

- Required: `project_key`, `key`, `name`. Optional: `description`.
- Simplest of the set and the only one validated in a tagged preview. Use it as the smoke test that the dev-override build is wired up correctly.
- Verification: expect a clean create, an empty follow-up plan, an update on `name` or `description`, and a clean delete.

### launchdarkly_announcement — Stable, account scoped, singleton

- Required: `title`, `message`, `severity`. Optional: `is_dismissible`, `start_time`, `end_time` as Unix timestamps in milliseconds.
- **Singleton.** LaunchDarkly allows one announcement per account and returns `409` on a second create. Before creating, clean up any existing announcement, following the account-singleton orphan-cleanup precheck. Retries do not clear this — LaunchDarkly reuses `409 optimistic_locking_error` for duplicates.
- Verification: create, empty follow-up plan, update `message`, delete. If create returns `409`, remove the existing announcement first; that is BLOCKED, not FAIL.

### launchdarkly_oauth_client — Stable, account scoped

- Required: `name`, `redirect_uri`. Optional: `description`.
- The API likely returns a client secret on create. Confirm any secret field is marked sensitive and is not echoed in plan output. Treat a secret that is settable but not returned on read as write-only, and confirm the provider does not blank it on refresh.
- May require an Admin or Owner role on the account. A permissions error is BLOCKED.

### launchdarkly_metric_group — Beta, project scoped

- Required: `project_key`, `key`, `name`, `kind` (for example `funnel` or `standard`), and a `metrics` list.
- The `metrics` entries reference metric keys that must already exist in the project. Create the referenced `launchdarkly_metric` resources first, or the create fails with a not-found error, which is FAIL only if the metrics do exist.
- A `funnel` group expects ordered metrics. Confirm ordering round-trips without drift.

### launchdarkly_release_policy — Beta, project scoped

- Required: `project_key`, `key`, `name`, `release_method` (for example `guarded-release`), and a `scope` with `environment_keys` and `flag_tag_keys`. The config also carries nested phase and stage structures — read the example in full.
- The largest config in the set. Likely requires a release pipeline or release guardian entitlement. A missing entitlement is BLOCKED.
- Verification: pay attention to the nested structures round-tripping. Perpetual drift on a nested block is FAIL.

### launchdarkly_big_segment_store_integration — Beta, project + environment scoped

- Required: `project_key`, `environment_key`, `integration_key` (for example `redis`), `name`, `on`, and `config` as a `jsonencode` object.
- Needs a reachable backing store, for example Redis or DynamoDB, with valid connection details in `config`. Without one this is BLOCKED.
- The `config` keys vary by `integration_key`. Match the integration's manifest.

### launchdarkly_flag_import_configuration — Beta, project scoped

- Required: `project_key`, `integration_key` (for example `split`), `name`, and `config` as a `jsonencode` object. Optional: `tags`.
- The `config` keys vary by integration and come from the integration manifest `formVariables`. The `split` example needs a workspace API key, workspace ID, environment ID, and a LaunchDarkly API key.
- Needs valid third-party credentials. Without them this is BLOCKED.

### launchdarkly_integration_delivery_configuration — Beta, project + environment scoped

- Required: `project_key`, `env_key`, `integration_key` (for example `fastly`), `name`, `on`, and a `config`.
- Note the attribute is `env_key` here, while `big_segment_store_integration` uses `environment_key`. Confirm the actual attribute name against the schema before writing config — this naming difference is a likely scaffold inconsistency worth flagging.
- Needs valid third-party credentials for the chosen integration. Without them this is BLOCKED.

## Data sources

Every new resource except `launchdarkly_announcement` also ships a matching data source. After a resource verifies PASS, confirm its data source reads the same object back by `key` and the relevant scope. A data source that cannot find a resource that demonstrably exists is FAIL.

## When this matrix is stale

The source of truth for the registered surface is `launchdarkly/plugin_provider.go` — the `Resources()` and `DataSources()` slices. If a later release adds or removes an entry, update this matrix. To check whether a resource uses the beta client, grep its `resource_<name>_framework.go` for `newBetaClient` or `forceBetaAPIVersion`.
