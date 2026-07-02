---
page_title: "v3.0.0 release notes for the LaunchDarkly provider"
description: |-
  Breaking changes, new resources, new settings, and bug fixes in v3.0.0 of the LaunchDarkly Terraform provider, with links to the migration guide and tooling.
---

# LaunchDarkly Terraform provider v3.0.0 release notes

Version 3.0.0 completes the provider's migration to the HashiCorp Terraform Plugin Framework. The provider now serves the protocol version 6 plugin API, expresses every nested structure as a nested attribute instead of a configuration block, and removes the attributes that v2 deprecated. v3 also adds new resources, data sources, and a provider setting, and it ships a command-line tool that rewrites v2 configurations into v3 syntax. v3.0.0 follows a series of preview releases (v3.0.0-beta.1 through v3.0.0-beta.4).

> **Before you upgrade:** your v2 configurations do not parse against v3 until you rewrite them from block syntax to nested attribute syntax. To convert them, read the [migration guide](https://registry.terraform.io/providers/launchdarkly/launchdarkly/latest/docs/guides/migrating-to-v3) and run the `migrate-tf-syntax` tool that ships with this release.

## Summary

- The provider is rebuilt on the Terraform Plugin Framework. The plugin protocol moves from version 5 to version 6.
- Block syntax is replaced by nested attribute syntax across every resource and data source.
- Single-object attributes (`client_side_availability`, `defaults`, `default_client_side_availability`, `fallthrough`, `approval_settings`, `segment_approval_settings`, `instructions`, `boolean_defaults`) use object syntax (`= { ... }`), not a single-element list.
- Keyed collections become maps: `launchdarkly_project.environments` (by environment key), `launchdarkly_feature_flag.custom_properties` (by property key), and `role_attributes` on `launchdarkly_team` / `launchdarkly_team_member` (a plain map of string lists). Adding or removing one entry no longer churns its siblings.
- Five deprecated attributes are removed, across `launchdarkly_access_token`, `launchdarkly_custom_role`, `launchdarkly_feature_flag`, `launchdarkly_project`, and `launchdarkly_metric`.
- State upgrades run automatically on first apply. No resource is destroyed or recreated.
- v3 adds nine resources and eight data sources. It removes none.
- v3 adds the `archive_flags_on_destroy` provider setting.
- v3 ships `migrate-tf-syntax`, a configuration conversion tool, as a release asset.

## Breaking changes

### Migration to the Terraform Plugin Framework

The provider no longer uses the legacy Terraform SDK. Every nested block becomes a nested attribute, so you assign a single nested structure with `=` and you wrap a repeated structure in a list of objects.

Here is an example:

```hcl
# v2 block syntax
resource "launchdarkly_feature_flag" "example" {
  variation_type = "boolean"
  variations { value = "true" }
  variations { value = "false" }
}

# v3 nested attribute syntax
resource "launchdarkly_feature_flag" "example" {
  variation_type = "boolean"
  variations = [
    { value = "true" },
    { value = "false" },
  ]
}
```

This change affects every block in the provider. These are the affected attribute names:

```
approval_settings, boolean_defaults, clauses, client_side_availability,
context_targets, custom_properties, default_client_side_availability,
defaults, environments, excluded_contexts, fallthrough, included_contexts,
inline_roles, instructions, linked_segments, maintainers, messages, policy,
policy_statements, prerequisites, role_attributes, rules, segments,
statements, targets, urls, variations
```

The `launchdarkly_audit_log_subscription` `config` block also moves to map attribute syntax, written as `config = { ... }`.

### Single nested attributes use object syntax

Most blocks become a list of objects, but attributes that hold exactly one object use object syntax — a bare `{ ... }` with no surrounding brackets:

| Resource or data source | Attribute |
|---|---|
| `launchdarkly_feature_flag` | `client_side_availability` |
| `launchdarkly_feature_flag` | `defaults` |
| `launchdarkly_project` | `default_client_side_availability` |
| `launchdarkly_project` | `approval_settings` (inside each `environments` entry) |
| `launchdarkly_feature_flag_environment` | `fallthrough` |
| `launchdarkly_environment` | `approval_settings` |
| `launchdarkly_environment` | `segment_approval_settings` |
| `launchdarkly_flag_trigger` | `instructions` |
| `launchdarkly_flag_templates` | `boolean_defaults` |

```hcl
# v2 block syntax
resource "launchdarkly_feature_flag" "example" {
  client_side_availability {
    using_environment_id = true
    using_mobile_key     = false
  }
}

# v3 object syntax (no brackets)
resource "launchdarkly_feature_flag" "example" {
  client_side_availability = {
    using_environment_id = true
    using_mobile_key     = false
  }
}
```

The `migrate-tf-syntax` tool emits this object form automatically. When you read these attributes from a data source, drop the list index too: `data.launchdarkly_feature_flag.x.client_side_availability.using_environment_id`, not `...client_side_availability[0].using_environment_id`.

> **Upgrading from a `3.0.0-beta` pre-release?** These attributes were modeled as single-element lists through the `3.0.0-beta` pre-releases (`= [{ ... }]`). The switch to object syntax landed for GA. The pre-release → GA jump is not state-compatible for them — update the configuration to the object form; the provider's state upgraders convert pre-release state in place. Upgrades from v2 are handled automatically by the state upgrade and the `migrate-tf-syntax` tool.

### Keyed collections become maps

Four collections whose elements have a natural unique key are maps in v3, so adding, removing, or reordering one entry no longer forces a diff (or a destructive plan) on its siblings:

| Resource | Attribute | Map key |
|---|---|---|
| `launchdarkly_project` | `environments` | environment `key` |
| `launchdarkly_feature_flag` | `custom_properties` | custom property `key` |
| `launchdarkly_team`, `launchdarkly_team_member` | `role_attributes` | role attribute key |
| `launchdarkly_ai_agent_graph` | `edges` | edge `key` |

`launchdarkly_project.environments` moves from an ordered list to a map keyed by the environment `key`. The `key` attribute also stays inside each object (it is Optional+Computed and always equals the map key), so references like `launchdarkly_project.example.environments["production"].key` keep working:

```hcl
# v2 block syntax              # v3 map syntax (keyed by env key)
environments {                 environments = {
  key   = "production"           "production" = {
  name  = "Production"             name  = "Production"
  color = "EEEEEE"                 color = "EEEEEE"
}                                }
                               }
```

The map is **authoritative**: an environment removed from the map is deleted on apply, and a project must declare at least one environment. To manage a project in Terraform but its environments elsewhere, add `lifecycle { ignore_changes = [environments] }`.

~> **Warning:** Changing an environment's key (the map key) deletes that environment — including its SDK keys and all flag targeting — and creates a new one.

Reference environments by key, not index: `environments["production"].client_side_id`, never `environments[0].client_side_id`. The `migrate-tf-syntax` tool rewrites the blocks and warns on each positional reference with the exact replacement.

`custom_properties` follows the same object-map pattern (`custom_properties = { "my.key" = { name = ..., value = [...] } }`). `role_attributes` collapses further to a plain map of string lists — `role_attributes = { myAttribute = ["value1", "value2"] }` — matching both the LaunchDarkly API shape and the `launchdarkly_team_role_mapping` resource.

> **Upgrading from a `3.0.0-beta` pre-release?** These collections were lists or sets through the `3.0.0-beta` pre-releases. Update the configuration to the map form; the provider's state upgraders convert both v2 and pre-release state in place (except `launchdarkly_project.environments` state from `3.0.0-beta.1`–`beta.3`, which pre-dates the map and must be re-imported).

### Removed attributes

v3 removes the attributes that v2 marked deprecated. The state upgrade migrates existing state for each one, and the `migrate-tf-syntax` tool rewrites your configuration. This table lists each removed attribute and its replacement:

| Resource or data source | Removed attribute | Replacement |
|---|---|---|
| `launchdarkly_access_token` | `policy_statements` | `inline_roles` |
| `launchdarkly_access_token` | `expire` | None. Remove it from your configuration. |
| `launchdarkly_custom_role` | `policy` | `policy_statements` |
| `launchdarkly_feature_flag` | `include_in_snippet` | `client_side_availability` |
| `launchdarkly_project` resource | `include_in_snippet` | `default_client_side_availability` |
| `launchdarkly_project` data source | `client_side_availability` | `default_client_side_availability` |
| `launchdarkly_metric` | `is_active` | None. Remove it from your configuration. |

### Minimum versions

- Terraform: the protocol version 6 plugin API requires Terraform 1.0 or later.
- Go: building the provider from source requires Go 1.25.8, up from 1.25.5. This does not affect anyone who installs the provider from the Terraform Registry.

## New features

### New provider setting

v3 adds `archive_flags_on_destroy`. When you set it to `true`, removing a `launchdarkly_feature_flag` resource from your configuration archives the flag in LaunchDarkly instead of deleting it. The default is `false`, which preserves the v2 destroy behavior. This setting affects only `launchdarkly_feature_flag`.

### New resources and data sources

This table lists the new resources, their data sources, and their API stability:

| Resource | Data source | API stability |
|---|---|---|
| `launchdarkly_context_kind` | `launchdarkly_context_kind` | Stable API |
| `launchdarkly_announcement` | None | Stable API |
| `launchdarkly_oauth_client` | `launchdarkly_oauth_client` | Stable API |
| `launchdarkly_ai_agent_graph` | `launchdarkly_ai_agent_graph` | Beta |
| `launchdarkly_metric_group` | `launchdarkly_metric_group` | Beta |
| `launchdarkly_release_policy` | `launchdarkly_release_policy` | Beta |
| `launchdarkly_big_segment_store_integration` | `launchdarkly_big_segment_store_integration` | Beta |
| `launchdarkly_flag_import_configuration` | `launchdarkly_flag_import_configuration` | Beta |
| `launchdarkly_integration_delivery_configuration` | `launchdarkly_integration_delivery_configuration` | Beta |

> **Some new resources are in beta.** The `launchdarkly_ai_agent_graph`, `launchdarkly_metric_group`, `launchdarkly_release_policy`, `launchdarkly_big_segment_store_integration`, `launchdarkly_flag_import_configuration`, and `launchdarkly_integration_delivery_configuration` resources are in beta. The functionality may change without notice or become backwards incompatible.

### Other enhancements

- `launchdarkly_environment` gains a `segment_approval_settings` attribute. This attribute configures approval requirements for segment changes.
- `launchdarkly_feature_flag` validates prerequisite-flag removals at plan time. If you remove a flag that another flag depends on, the provider surfaces a warning during plan instead of failing at apply.
- `launchdarkly_feature_flag_environment` makes `off_variation` optional. In v2 it was required. Omitting it now leaves the off variation unset, matching the LaunchDarkly UI's "Not set" state, and removing a previously configured value clears it. Note the behavior change: when the off variation is unset and targeting is off, LaunchDarkly serves no variation, so SDKs return the application-provided default value and the evaluation carries a null variation index (which affects Data Export and Experimentation). Setting `off_variation = 0` remains a distinct, valid configuration. This resolves issue #482.

## Bug fixes

- The provider creates a segment correctly when the target environment requires approvals for segment changes. This resolves issue #370.
- `launchdarkly_access_token` applies cleanly when you upgrade state for a token that set a role or custom roles without an inline policy. v2 stored an empty list where v3 stores null, and the update path now treats the two as equal.
- `launchdarkly_ai_config_variation` preserves the configured `description` and `instructions`. The LaunchDarkly API does not return these write-only fields on a read, so the provider keeps your configured value instead of clearing it.
- `launchdarkly_feature_flag` preserves variation `name` and `description` set outside Terraform. When your configuration omits them, the provider no longer clears them on apply.

## Upgrade tooling

v3 ships `migrate-tf-syntax`, a deterministic command-line tool that rewrites v2 block syntax to v3 nested attribute syntax and updates the removed attributes. The release publishes prebuilt binaries for macOS, Linux, and Windows on amd64 and arm64, as separate archives next to the provider. You can also run the tool with Go. To learn how to convert a configuration, read the [migration guide](https://registry.terraform.io/providers/launchdarkly/launchdarkly/latest/docs/guides/migrating-to-v3).
