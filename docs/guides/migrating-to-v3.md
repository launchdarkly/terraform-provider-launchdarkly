---
page_title: "Migrating your configuration to v3 of the LaunchDarkly provider"
description: |-
  This guide explains how to upgrade a v2.x configuration to v3 of the LaunchDarkly provider with the migrate-tf-syntax tool, and what to expect on your first plan.
---

# Migrating your configuration to v3 of the LaunchDarkly provider

## Overview

This topic explains how to upgrade your Terraform configuration from v2 to v3 of the LaunchDarkly provider. v3 changes every nested block to a nested attribute, so configurations written for v2 do not parse after you upgrade the provider. You must rewrite them before your first plan against v3. The provider ships the `migrate-tf-syntax` tool to automate most of the rewrite, and it upgrades your state automatically on first apply.

Here is the same resource in both syntaxes:

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

Attributes that hold exactly one object rather than a list use object syntax — a bare `{ ... }` with no brackets: `client_side_availability` and `defaults` on `launchdarkly_feature_flag`, `default_client_side_availability` on `launchdarkly_project`, `fallthrough` on `launchdarkly_feature_flag_environment`, `approval_settings` on `launchdarkly_environment` (and inside each project environment), `segment_approval_settings` on `launchdarkly_environment`, and `instructions` on `launchdarkly_flag_trigger`. The `migrate-tf-syntax` tool emits this form for you:

```hcl
# v2 block syntax            # v3 object syntax (no brackets)
client_side_availability {   client_side_availability = {
  using_environment_id = true  using_environment_id = true
}                            }
```

When you read one of these from a data source, use object access without a list index: `data.launchdarkly_feature_flag.x.client_side_availability.using_environment_id`.

`launchdarkly_project.environments` becomes a **map keyed by the environment `key`** rather than an ordered list, so reordering, adding, or removing one environment no longer shifts the others or forces a destructive plan. The environment's `key` is also kept inside the object (it equals the map key), so references like `launchdarkly_project.example.environments["production"].key` keep working. The `migrate-tf-syntax` tool performs this rewrite for you:

```hcl
# v2 block syntax              # v3 map syntax (keyed by env key)
environments {                 environments = {
  key   = "production"           "production" = {
  name  = "Production"             key   = "production"
  color = "EEEEEE"                 name  = "Production"
}                                  color = "EEEEEE"
                                 }
                               }
```

The map is **authoritative**: an environment removed from the map is deleted, and a project must have at least one environment. To manage the project in Terraform but its environments in the LaunchDarkly UI (or via [`launchdarkly_environment`](/docs/providers/launchdarkly/r/environment.html) resources), declare your environments and add `lifecycle { ignore_changes = [environments] }`.

~> **Warning:** Changing an environment's key (the map key) deletes that environment — including its SDK keys and all flag targeting — and creates a new one.

Reference an environment by its key instead of by index: a v2 interpolation such as `launchdarkly_project.example.environments[0].client_side_id` becomes `launchdarkly_project.example.environments["production"].client_side_id`. The `migrate-tf-syntax` tool does **not** rewrite these positional references (auto-editing arbitrary expressions risks corrupting your config), but it **detects them and prints the exact replacement** to make — including the resolved key — so the fix is mechanical. See "Finish the migration by hand" below.

Three more collections follow the same key-addressed map pattern, and the tool rewrites all of them:

- `custom_properties` on `launchdarkly_feature_flag` becomes a map keyed by the custom property `key`: `custom_properties = { "my.key" = { name = ..., value = [...] } }`.
- `role_attributes` on `launchdarkly_team` and `launchdarkly_team_member` becomes a plain map of string lists keyed by the role attribute key: `role_attributes = { myAttribute = ["value1", "value2"] }` — the same shape `launchdarkly_team_role_mapping` already uses.
- `edges` on the new `launchdarkly_ai_agent_graph` resource is a map keyed by edge key (net-new in v3, so no rewrite applies).

## Prerequisites

You need the following things to complete this migration:

- Terraform 1.0 or later
- A v2 configuration that applies cleanly, with an empty plan before you start
- A committed copy of your configuration and state, so you can review the upgrade as a diff

## Convert your configuration with migrate-tf-syntax

The provider ships `migrate-tf-syntax`, a deterministic command-line tool that rewrites every affected attribute across a directory of `.tf` files. It also updates the attributes that v3 removed. For example, it renames `policy_statements` to `inline_roles` on `launchdarkly_access_token`, and it updates references to renamed data source attributes such as `client_side_availability` on the `launchdarkly_project` data source. It adds the now-required `variations` to boolean flags that omitted them.

To convert a configuration directory:

1. Download the `migrate-tf-syntax` archive for your platform from the [provider release assets](https://github.com/launchdarkly/terraform-provider-launchdarkly/releases), or run the tool with Go. Replace `v3.0.0` with the version you are upgrading to:

   ```bash
   go run github.com/launchdarkly/terraform-provider-launchdarkly/scripts/migrate-tf-syntax@v3.0.0 \
     -dir ./my-config -direction v2-to-v3 -dry-run
   ```

2. Review the dry-run output. The tool prints each file it intends to change.
3. Run the same command without `-dry-run` to write the changes. Add `-recursive` to convert locally vendored modules in the same pass.
4. Run `terraform fmt` to normalize whitespace.
5. Update the provider version constraint to `~> 3.0` and run `terraform plan`.

## Finish the migration by hand

The tool converts syntax only. Complete these follow-ups yourself:

- Add `variations` by hand only for a flag whose `variation_type` is a non-literal expression, such as a variable or local. The tool cannot resolve those statically, so it warns and skips them. Boolean flags with a literal `variation_type` are handled automatically, and the provider preserves any variation `name` or `description` set outside Terraform when your configuration omits them.
- Rewrite `dynamic` blocks. A `dynamic "variations"` block needs a for expression, for example `variations = [for v in var.values : { value = v }]`. The tool warns with the file and resource address, and it leaves the attribute unchanged.
- Upgrade modules sourced from a registry or a git URL. The tool rewrites only files it reaches on disk, so upgrade those modules at their source.
- Rewrite positional references to `launchdarkly_project` environments. The tool converts the `environments` block to a map but does not edit index expressions elsewhere in your config; it warns on each one with the exact replacement (e.g. `environments[0]` → `environments["production"]`, and `environments[*]` → `values(...)`). Apply those edits by hand.

## How v3 upgrades your state

The provider includes a state upgrader for every resource whose state shape changed. On your first apply, the provider migrates your state automatically. You do not edit the state file by hand:

- `launchdarkly_access_token`: moves `policy_statements` into `inline_roles`, and discards `expire`.
- `launchdarkly_custom_role`: converts `policy` into `policy_statements`.
- `launchdarkly_feature_flag`: converts `include_in_snippet` into `client_side_availability`, and re-keys `custom_properties` into a map keyed by property key.
- `launchdarkly_project`: converts `include_in_snippet` into `default_client_side_availability`, re-keys the ordered `environments` list into a map keyed by environment key, and converts each environment's `approval_settings` to an object.
- `launchdarkly_environment`: converts `approval_settings` (and `segment_approval_settings`) to objects.
- `launchdarkly_feature_flag_environment`: converts `fallthrough` to an object.
- `launchdarkly_flag_trigger`: converts `instructions` to an object.
- `launchdarkly_team` and `launchdarkly_team_member`: re-key `role_attributes` into a map of string lists.
- `launchdarkly_metric`: discards `is_active`.

## Your first plan after upgrading

Expect a non-empty first plan after you upgrade the provider binary. v2 stored empty lists where v3 stores null, so diffs such as `policy_statements = [] -> null` appear once and apply cleanly. A few computed attributes show as known after apply on the first plan, and they resolve on apply. No resource is destroyed or recreated. The follow-up plan is empty.

## What does not change

- v3 removes no resources and no data sources. Every v2 resource and data source remains available.
- Authentication is unchanged. The `access_token`, `oauth_token`, `api_host`, `http_timeout`, and `max_concurrency` provider settings keep the same names and behavior.

## If you consume the provider through Crossplane

If you embed this provider through Crossplane's Upjet, the block-to-attribute change alters the generated custom resource definition (CRD) shape, even though the attribute names do not change. We recommend that you test CRD regeneration against v3 before you upgrade.
