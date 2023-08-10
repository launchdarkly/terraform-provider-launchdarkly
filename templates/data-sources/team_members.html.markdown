---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_team_members"
description: |-
  Get information about multiple LaunchDarkly team members.
---

# launchdarkly_team_members

Provides a LaunchDarkly team members data source.

This data source allows you to retrieve team member information from your LaunchDarkly organization on multiple team members.

## Example Usage

```hcl
data "launchdarkly_team_member" "example" {
  emails = ["example@example.com", "example2@example.com", "example3@example.com"]
}
```

## Argument Reference

- `emails` - (Required) An array of unique email addresses associated with the team members.

- `ignore_missing` - (Optional) A boolean to determine whether to ignore members that weren't found. 

## Attributes Reference

In addition to the arguments above, the resource exports the found members as `team_members`.  
The following attributes are available for each member:

- `id` - The 24 character alphanumeric ID of the team member.

- `first_name` - The team member's given name.

- `last_name` - The team member's family name.

- `role` - The role associated with team member. Possible roles are `owner`, `reader`, `writer`, or `admin`.

- `custom_role` - (Optional) The list of custom roles keys associated with the team member. Custom roles are only available to customers on enterprise plans. To learn more about enterprise plans, contact sales@launchdarkly.com.
