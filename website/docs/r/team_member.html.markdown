---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_team_member"
description: |-
  Create and manage LaunchDarkly team members.
---

# launchdarkly_team_member

Provides a LaunchDarkly team member resource.

This resource allows you to create and manage team members within your LaunchDarkly organization.

-> **Note:** You can only manage team members with "admin" level personal access tokens. To learn more, read [Managing Teams](https://docs.launchdarkly.com/docs/teams).

## Example Usage

```hcl
resource "launchdarkly_team_member" "example" {
  email        = "example.user@example.com"
  first_name   = "John"
  last_name    = "Smith"
  role         = "writer"
}
```

## Argument Reference

- `email` - (Required) The unique email address associated with the team member.

- `first_name` - (Optional) The team member's given name. Please note that, once created, this cannot be updated except by the team member themself.

- `last_name` - (Optional) The team member's family name. Please note that, once created, this cannot be updated except by the team member themself.

- `role` - (Optional) The role associated with team member. Supported roles are `reader`, `writer`, or `admin`. If you don't specify a role, `reader` is assigned by default.

- `custom_roles` - (Optional) The list of custom roles keys associated with the team member. Custom roles are only available to customers on enterprise plans. To learn more about enterprise plans, contact sales@launchdarkly.com.

-> **Note:** each `launchdarkly_team_member` must have either a `role` or `custom_roles` argument.

## Attributes Reference

In addition to the arguments above, the resource exports the following attribute:

- `id` - The 24 character alphanumeric ID of the team member.

## Import

LaunchDarkly team members can be imported using the team member's 24 character ID, e.g.

```
$ terraform import launchdarkly_team_member.example 5f05565b48be0b441fb63020
```
