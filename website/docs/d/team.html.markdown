---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_team"
description: |-
  Get information about a LaunchDarkly team.
---

# launchdarkly_team

Provides a LaunchDarkly team data source.

This data source allows you to retrieve team information from your LaunchDarkly organization on a team.

## Example Usage

```hcl
data "launchdarkly_team" "example" {
  key = ["example_key_1"]
}
```

## Argument Reference

- `key` - (Required) A string associated with a team key.

- `name` - (Optional) A string associated with a team name.

- `description` - (Optional) A string associated with a team description.

## Attributes Reference

In addition to the arguments above, the resource exports the found team as `team`.  
The following attributes are available for a team:

- `member ID's` - The list of team member's IDs as strings.

- `maintainers` - The list of team maintainers as strings.

- `custom_role_keys` - The list of keys for custom roles the team has.
