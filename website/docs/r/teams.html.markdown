---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_team"
description: |-
  Create and manage a LaunchDarkly team.
---

# launchdarkly_team

Provides a LaunchDarkly team resource.

This resource allows you to create and manage a teamwithin your LaunchDarkly organization.

-> **Note:** You can only manage a team with "admin" level personal access tokens. To learn more, read [Managing Teams](https://docs.launchdarkly.com/docs/teams).

## Example Usage

```hcl
resource "launchdarkly_team" "example" {
  key                   = "example_key"
  name                  = "Team Name"
  description           = "example team description"

  members               = ["example_user_key_1", "example_user_key_2"]
  maintainers           = ["example_user_key_1", "example_user_key_2"]
  custom_role_keys      = ["example_role_1", "example_role_1"]
}
```

## Argument Reference

- `key` - (Required) The team's human readable key. This cannot be changed after creation. 

- `name` - (Required) The team's name. Please note that, once created, this must be changed by someone with admin access to teams in LaunchDarkly.

- `description` - (Required) The team's description. Please note that, once created, this must be changed by someone with admin access to teams in LaunchDarkly.

- `members` - (Required) The list of team member's IDs as strings. Please note that this must be edited by someone with admin access to teams in LaunchDarkly.

- `maintainers` - (Required) The list of the team's maintainers. Please note that this must be edited by someone with admin access to teams in LaunchDarkly.

- `custom_role_keys` - (Required) The list of custom roles keys associated with the team. Please note that this must be edited by someone with admin access to teams in LaunchDarkly.


## Import

A LaunchDarkly team can be imported using the team's key e.g.

```
$ terraform import launchdarkly_team.example example_team_key
```
