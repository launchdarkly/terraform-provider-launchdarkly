---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_team_role_mapping"
description: |-
  Manage the custom roles associated with a LaunchDarkly team.
---

# launchdarkly_team_role_mapping

Provides a LaunchDarkly team to custom role mapping resource.

This resource allows you to manage the custom roles associated with a LaunchDarkly team. This is useful if the LaunchDarkly team is created and managed externally, such as via [team sync with SCIM](https://launchdarkly.com/docs/home/account/scim#team-sync-with-scim). If you wish to create and manage the team using Terraform, we recommend using the [`launchdarkly_team` resource](https://registry.terraform.io/providers/launchdarkly/launchdarkly/latest/docs/resources/team) instead.

-> **Note:** Teams are available to customers on an Enterprise LaunchDarkly plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).

## Example Usage

```hcl
resource "launchdarkly_team_role_mapping" "platform_team" {
  team_key         = "platform_team"
  custom_role_keys = ["platform", "nomad-administrators"]
}
```

### Scoping a shared custom role across teams

Use `role_attributes` to give the same shared custom role different scopes per team. The values resolve into the role's policy via the `${roleAttribute/<key>}` template, so a single role definition can drive per-team access boundaries (see [Role scope](https://launchdarkly.com/docs/home/account/roles/role-scope)).

```hcl
resource "launchdarkly_team_role_mapping" "team_x" {
  team_key         = "team-x"
  custom_role_keys = ["my-shared-role"]
  role_attributes = {
    domain = ["DomainX"]
  }
}

resource "launchdarkly_team_role_mapping" "team_y" {
  team_key         = "team-y"
  custom_role_keys = ["my-shared-role"]
  role_attributes = {
    domain = ["DomainY"]
  }
}
```

## Argument Reference

- `team_key` - (Required) The team key.

- `custom_role_keys` - (Required) List of custom role keys granted to the team. The referenced custom roles must already exist in LaunchDarkly. If they don't, the provider may behave unexpectedly.

- `role_attributes` - (Optional) Map of role-attribute keys to lists of resource keys. Applied to the team as a whole. Every custom role granted to this team receives these scopes. Leave unset (or remove from configuration) to keep the team's role attributes unchanged from the LaunchDarkly side.

  ~> **Note:** `role_attributes` is also exposed on the [`launchdarkly_team` resource](https://registry.terraform.io/providers/launchdarkly/launchdarkly/latest/docs/resources/team). If you manage the same team with both resources, only one of them should own `role_attributes`. Add `lifecycle { ignore_changes = [role_attributes] }` on whichever resource isn't the primary owner to avoid plan churn.

## Import

A LaunchDarkly team/role mapping can be imported using the team key:

```
$ terraform import launchdarkly_team_role_mapping.platform_team platform_team
```
