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

  policy_statements {
    effect    = "allow"
    resources = ["proj/*:env/production:flag/*"]
    actions   = ["*"]
  }
  policy_statements {
    effect    = "allow"
    resources = ["proj/*:env/production"]
    actions   = ["*"]
  }
}
```

## Argument Reference

- `key` - (Required) The unique key that references the custom role.

- `name` - (Required) The human-readable name for the custom role.

- `description` - (Optional) The description of the custom role.

- `policy_statements` - (Required) The custom role policy block. To learn more, read [Policies in custom roles](https://docs.launchdarkly.com/docs/policies-in-custom-roles).

Custom role `policy_statements` blocks are composed of the following arguments:

- `effect` - (Required) - Either `allow` or `deny`. This argument defines whether the statement allows or denies access to the named resources and actions.

- `resources` - (Optional) - The list of resource specifiers defining the resources to which the statement applies. Either `resources` or `not_resources` must be specified. For a list of available resources read [Understanding resource types and scopes](https://docs.launchdarkly.com/home/account-security/custom-roles/resources#understanding-resource-types-and-scopes).

- `not_resources` - (Optional) - The list of resource specifiers defining the resources to which the statement does not apply. Either `resources` or `not_resources` must be specified. For a list of available resources read [Understanding resource types and scopes](https://docs.launchdarkly.com/home/account-security/custom-roles/resources#understanding-resource-types-and-scopes).

- `actions` - (Optional) The list of action specifiers defining the actions to which the statement applies. Either `actions` or `not_actions` must be specified. For a list of available actions read [Actions reference](https://docs.launchdarkly.com/home/account-security/custom-roles/actions#actions-reference).

- `not_actions` - (Optional) The list of action specifiers defining the actions to which the statement does not apply. Either `actions` or `not_actions` must be specified. For a list of available actions read [Actions reference](https://docs.launchdarkly.com/home/account-security/custom-roles/actions#actions-reference).

## Import

You can import LaunchDarkly custom roles by using an existing custom role `key`.

For example:

```
$ terraform import launchdarkly_custom_role.example example-role-key-1
```
