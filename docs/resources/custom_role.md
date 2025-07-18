---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "launchdarkly_custom_role Resource - launchdarkly"
subcategory: ""
description: |-
  Provides a LaunchDarkly custom role resource.
  -> Note: Custom roles are available to customers on an Enterprise LaunchDarkly plan. To learn more, read about our pricing https://launchdarkly.com/pricing/. To upgrade your plan, contact LaunchDarkly Sales https://launchdarkly.com/contact-sales/.
  This resource allows you to create and manage custom roles within your LaunchDarkly organization.
---

# launchdarkly_custom_role (Resource)

Provides a LaunchDarkly custom role resource.

-> **Note:** Custom roles are available to customers on an Enterprise LaunchDarkly plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).

This resource allows you to create and manage custom roles within your LaunchDarkly organization.

## Example Usage

```terraform
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

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `key` (String) A unique key that will be used to reference the custom role in your code. A change in this field will force the destruction of the existing resource and the creation of a new one.
- `name` (String) A name for the custom role. This must be unique within your organization.

### Optional

- `base_permissions` (String) The base permission level - either `reader` or `no_access`. While newer API versions default to `no_access`, this field defaults to `reader` in keeping with previous API versions.
- `description` (String) Description of the custom role.
- `policy` (Block Set, Deprecated) (see [below for nested schema](#nestedblock--policy))
- `policy_statements` (Block List) An array of the policy statements that define the permissions for the custom role. This field accepts [role attributes](https://docs.launchdarkly.com/home/getting-started/vocabulary#role-attribute). To use role attributes, use the syntax `$${roleAttribute/<YOUR_ROLE_ATTRIBUTE>}` in lieu of your usual resource keys. (see [below for nested schema](#nestedblock--policy_statements))

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedblock--policy"></a>
### Nested Schema for `policy`

Required:

- `actions` (List of String)
- `effect` (String)
- `resources` (List of String)


<a id="nestedblock--policy_statements"></a>
### Nested Schema for `policy_statements`

Required:

- `effect` (String) Either `allow` or `deny`. This argument defines whether the statement allows or denies access to the named resources and actions.

Optional:

- `actions` (List of String) The list of action specifiers defining the actions to which the statement applies.
Either `actions` or `not_actions` must be specified. For a list of available actions read [Actions reference](https://docs.launchdarkly.com/home/account-security/custom-roles/actions#actions-reference).
- `not_actions` (List of String) The list of action specifiers defining the actions to which the statement does not apply.
- `not_resources` (List of String) The list of resource specifiers defining the resources to which the statement does not apply.
- `resources` (List of String) The list of resource specifiers defining the resources to which the statement applies.

## Import

Import is supported using the following syntax:

```shell
terraform import launchdarkly_custom_role.example example-role-key-1
```
