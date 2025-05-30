---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "launchdarkly_team Resource - launchdarkly"
subcategory: ""
description: |-
  Provides a LaunchDarkly team resource.
  This resource allows you to create and manage a team within your LaunchDarkly organization.
  -> Note: Teams are available to customers on an Enterprise LaunchDarkly plan. To learn more, read about our pricing https://launchdarkly.com/pricing/. To upgrade your plan, contact LaunchDarkly Sales https://launchdarkly.com/contact-sales/.
---

# launchdarkly_team (Resource)

Provides a LaunchDarkly team resource.

This resource allows you to create and manage a team within your LaunchDarkly organization.

-> **Note:** Teams are available to customers on an Enterprise LaunchDarkly plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).

## Example Usage

```terraform
resource "launchdarkly_team" "platform_team" {
  key              = "platform_team"
  name             = "Platform team"
  description      = "Team to manage internal infrastructure"
  member_ids       = ["507f1f77bcf86cd799439011", "569f183514f4432160000007"]
  maintainers      = ["12ab3c45de678910abc12345"]
  custom_role_keys = ["platform", "nomad-administrators"]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `key` (String) The team key. A change in this field will force the destruction of the existing resource and the creation of a new one.
- `name` (String) A human-friendly name for the team.

### Optional

- `custom_role_keys` (Set of String) List of custom role keys the team will access. The referenced custom roles must already exist in LaunchDarkly. If they don't, the provider may behave unexpectedly.
- `description` (String) The team description.
- `maintainers` (Set of String) List of member IDs for users who maintain the team.
- `member_ids` (Set of String) List of member IDs who belong to the team.
- `role_attributes` (Block Set) A role attributes block. One block must be defined per role attribute. The key is the role attribute key and the value is a string array of resource keys that apply. (see [below for nested schema](#nestedblock--role_attributes))

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedblock--role_attributes"></a>
### Nested Schema for `role_attributes`

Required:

- `key` (String) The key / name of your role attribute. In the example `$${roleAttribute/testAttribute}`, the key is `testAttribute`.
- `values` (List of String) A list of values for your role attribute. For example, if your policy statement defines the resource `"proj/$${roleAttribute/testAttribute}"`, the values would be the keys of the projects you wanted to assign access to.

## Import

Import is supported using the following syntax:

```shell
# A LaunchDarkly team can be imported using the team key
terraform import launchdarkly_team.platform_team platform_team
```
