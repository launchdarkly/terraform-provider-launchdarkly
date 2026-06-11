---
page_title: "Migrating your configuration to v3 of the LaunchDarkly provider"
description: |-
  This guide explains how to rewrite v2 block syntax to v3 nested attribute syntax, either with the migrate-tf-syntax tool or by hand.
---

# Migrating your configuration to v3 of the LaunchDarkly provider

Version 3 of the LaunchDarkly Terraform provider changes every nested block to a nested attribute. Configurations written for v2 no longer parse after you upgrade the provider, so you must rewrite them before your first `terraform plan` against v3.

Here is an example of the same resource in both syntaxes:

```hcl
# v2 block syntax
resource "launchdarkly_feature_flag" "example" {
  variation_type = "boolean"
  variations {
    value = "true"
  }
  variations {
    value = "false"
  }
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

## Converting your configuration with migrate-tf-syntax

The provider repository includes `migrate-tf-syntax`, a deterministic command-line tool that rewrites every affected attribute in a directory of `.tf` files. It also migrates attributes that v3 removed: it replaces them with their successors, for example `policy_statements` with `inline_roles` on `launchdarkly_access_token`, and updates references to renamed data source attributes such as `client_side_availability` on the `launchdarkly_project` data source.

To convert a configuration directory:

1. Download the `migrate-tf-syntax` archive for your platform from the [provider release assets](https://github.com/launchdarkly/terraform-provider-launchdarkly/releases), or run the tool directly with Go:

   ```bash
   go run github.com/launchdarkly/terraform-provider-launchdarkly/scripts/migrate-tf-syntax@preview-v3 \
     -dir ./my-config -direction v2-to-v3 -dry-run
   ```

2. Review the dry-run output. The tool prints each file it intends to change.
3. Run the same command without `-dry-run` to write the changes. Add `-recursive` to convert locally vendored modules in the same pass.
4. Run `terraform fmt` to normalize whitespace.
5. Upgrade the provider version constraint to `~> 3.0` and run `terraform plan`.

## What the tool does not do

Complete these follow-ups by hand:

- Add attributes that v3 newly requires. For example, `launchdarkly_feature_flag` requires `variations` for every variation type, including boolean.
- Rewrite `dynamic` blocks. A `dynamic "variations"` block needs a for expression, for example `variations = [for v in var.values : { value = v }]`. The tool warns with the file and resource address and leaves the attribute unchanged.
- Upgrade modules sourced from a registry or a git URL. The tool rewrites only files it can reach on disk, so upgrade those modules at their source.

## Your first plan after upgrading

Expect a non-empty first plan after upgrading the provider binary. v2 stored empty lists where v3 stores null, so diffs such as `policy_statements = [] -> null` appear once and apply cleanly. No resource is destroyed or recreated. The follow-up plan is empty.
