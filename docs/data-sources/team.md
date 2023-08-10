---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_team"
description: |-
  Get information about a LaunchDarkly team.
---

# launchdarkly_team

Provides a LaunchDarkly team data source.

This data source allows you to retrieve team information from your LaunchDarkly organization.

-> **Note:** Teams are available to customers on an Enterprise LaunchDarkly plan. To learn more, read about our pricing. To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).
## Example Usage

```hcl
data "launchdarkly_team" "platform_team" {
  key = "platform_team"
}
```

## Argument Reference

- `key` - (Required) The team key.

## Attributes Reference

In addition to the arguments above, the resource exports the following attributes:

- `custom_role_keys` - The list of the keys of the custom roles that you have assigned to the team.

- `description` - The team description.

- `maintainers` - The list of team maintainers as [team member objects](/docs/providers/launchdarkly/d/team_member.html).

- `name` - Human readable name for the team.

- `project_keys` - The list of keys of the projects that the team has any write access to.
