---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_team"
description: |-
  Create and manage a LaunchDarkly team.
---

# launchdarkly_team

Provides a LaunchDarkly team resource.

This resource allows you to create and manage a team within your LaunchDarkly organization.

-> **Note:** Teams are available to customers on an Enterprise LaunchDarkly plan. To learn more, read about our pricing. To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).

## Example Usage

```hcl
resource "launchdarkly_team" "platform_team" {
  key                   = "platform_team"
  name                  = "Platform team"
  description           = "Team to manage internal infrastructure"
  member_ids            = ["507f1f77bcf86cd799439011", "569f183514f4432160000007"]
  maintainers           = ["12ab3c45de678910abc12345"]
  custom_role_keys      = ["platform", "nomad-administrators"]
}
```

## Argument Reference

- `key` - (Required) The team key.

- `name` - (Required) A human-friendly name for the team.

- `description` - (Optional) The team description.

- `member_ids` - (Optional) List of member IDs who belong to the team.

- `maintainers` - (Optional) List of member IDs for users who maintain the team.

- `custom_role_keys` - (Optional) List of custom role keys the team will access.


## Import

A LaunchDarkly team can be imported using the team key:

```
$ terraform import launchdarkly_team.platform_team platform_team
```
