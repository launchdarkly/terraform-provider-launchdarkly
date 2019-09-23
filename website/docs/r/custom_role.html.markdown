---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_custom_role"
description: |-
  Create and manage LaunchDarkly custom roles.
---

# launchdarkly_custom_role

Provides a LaunchDarkly custom role resource.

This resource allows you to create and manage custom roles within your LaunchDarkly organization.

-> **Note:** Custom roles are only available to customers on enterprise plans. To learn more about enterprise plans, contact sales@launchdarkly.com.

## Example Usage

```hcl
resource "launchdarkly_custom_role" "example" {
  key         = "example-role-key-1"
  name        = "example role"
  description = "This is an example role"

  policy {
    effect    = "allow"
    resources = ["proj/*:env/production"]
    actions   = ["*"]
  }
}
```

## Argument Reference

The following arguments are supported:

- `key` - (Required) The unique key that will be used to reference the custom role.

- `name` - (Required) The human-readable name for the custom role.

- `description` - (Optional) The description of the custom role.

- `policy` - (Required) The custom role policy block. Custom role policies are documented below.

Custom role `policy` blocks are composed as follows:

- `effect` - (Required) - Either `allow` or `deny`. This argument defines whether the statement allows or denies access to the named resources and actions.

- `resources` - (Required) - The list of resource specifiers defining the resources to which the statement applies or does not apply.

- `actions` - (Required) The list of action specifiers defining the actions to which the statement applies.

See [Policies in custom roles](https://docs.launchdarkly.com/docs/policies-in-custom-roles) for more information on how policies work in LaunchDarkly custom roles.

## Import

LaunchDarkly custom roles can be imported using an existing custom role `key`, e.g.

```
$ terraform import launchdarkly_custom_role.example example-role-key-1
```
