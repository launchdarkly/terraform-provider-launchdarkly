---
page_title: "v3.0.0 release notes for the LaunchDarkly provider"
description: |-
  Breaking changes, new resources, new settings, and bug fixes in v3.0.0 of the LaunchDarkly Terraform provider, with links to the migration guide and tooling.
---

# LaunchDarkly Terraform provider v3.0.0 release notes

Version 3.0.0 completes the provider's migration to the HashiCorp Terraform Plugin Framework. The provider now serves the protocol version 6 plugin API, expresses every nested structure as a nested attribute instead of a configuration block, and removes the attributes that v2 deprecated. v3 also adds new resources, data sources, and a provider setting, and it ships a command-line tool that rewrites v2 configurations into v3 syntax. v3.0.0 follows two preview releases, v3.0.0-beta.1 and v3.0.0-beta.2.

> **Before you upgrade:** your v2 configurations do not parse against v3 until you rewrite them from block syntax to nested attribute syntax. To convert them, read the [migration guide](https://registry.terraform.io/providers/launchdarkly/launchdarkly/latest/docs/guides/migrating-to-v3) and run the `migrate-tf-syntax` tool that ships with this release.

## Summary

- The provider is rebuilt on the Terraform Plugin Framework. The plugin protocol moves from version 5 to version 6.
- Block syntax is replaced by nested attribute syntax across every resource and data source.
- Five deprecated attributes are removed, across `launchdarkly_access_token`, `launchdarkly_custom_role`, `launchdarkly_feature_flag`, `launchdarkly_project`, and `launchdarkly_metric`.
- State upgrades run automatically on first apply. No resource is destroyed or recreated.
- v3 adds eight resources and seven data sources. It removes none.
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
| `launchdarkly_metric_group` | `launchdarkly_metric_group` | Beta |
| `launchdarkly_release_policy` | `launchdarkly_release_policy` | Beta |
| `launchdarkly_big_segment_store_integration` | `launchdarkly_big_segment_store_integration` | Beta |
| `launchdarkly_flag_import_configuration` | `launchdarkly_flag_import_configuration` | Beta |
| `launchdarkly_integration_delivery_configuration` | `launchdarkly_integration_delivery_configuration` | Beta |

> **Some new resources are in beta.** The `launchdarkly_metric_group`, `launchdarkly_release_policy`, `launchdarkly_big_segment_store_integration`, `launchdarkly_flag_import_configuration`, and `launchdarkly_integration_delivery_configuration` resources are in beta. The functionality may change without notice or become backwards incompatible.

### Other enhancements

- `launchdarkly_environment` gains a `segment_approval_settings` attribute. This attribute configures approval requirements for segment changes.
- `launchdarkly_feature_flag` validates prerequisite-flag removals at plan time. If you remove a flag that another flag depends on, the provider surfaces a warning during plan instead of failing at apply.

## Bug fixes

- The provider creates a segment correctly when the target environment requires approvals for segment changes. This resolves issue #370.
- `launchdarkly_access_token` applies cleanly when you upgrade state for a token that set a role or custom roles without an inline policy. v2 stored an empty list where v3 stores null, and the update path now treats the two as equal.
- `launchdarkly_ai_config_variation` preserves the configured `description` and `instructions`. The LaunchDarkly API does not return these write-only fields on a read, so the provider keeps your configured value instead of clearing it.
- `launchdarkly_feature_flag` preserves variation `name` and `description` set outside Terraform. When your configuration omits them, the provider no longer clears them on apply.

## Upgrade tooling

v3 ships `migrate-tf-syntax`, a deterministic command-line tool that rewrites v2 block syntax to v3 nested attribute syntax and updates the removed attributes. The release publishes prebuilt binaries for macOS, Linux, and Windows on amd64 and arm64, as separate archives next to the provider. You can also run the tool with Go. To learn how to convert a configuration, read the [migration guide](https://registry.terraform.io/providers/launchdarkly/launchdarkly/latest/docs/guides/migrating-to-v3).
