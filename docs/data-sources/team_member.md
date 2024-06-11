---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_team_member"
description: |-
  Get information about LaunchDarkly team members.
---

# launchdarkly_team_member

Provides a LaunchDarkly team member data source.

This data source allows you to retrieve team member information from your LaunchDarkly organization.

## Example Usage

```hcl
data "launchdarkly_team_member" "example" {
  email = "example@example.com"
}
```

## Argument Reference

- `email` - (Required) The unique email address associated with the team member.

## Attributes Reference

In addition to the arguments above, the resource exports the following attributes:

- `id` - The 24 character alphanumeric ID of the team member.

- `first_name` - The team member's given name.

- `last_name` - The team member's family name.

- `role` - The role associated with team member. Possible roles are `owner`, `reader`, `writer`, or `admin`.

- `custom_role` - (Optional) The list of custom roles keys associated with the team member. Custom roles are only available to customers on an Enterprise plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).
