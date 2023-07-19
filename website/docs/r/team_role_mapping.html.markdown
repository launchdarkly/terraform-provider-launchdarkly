---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_team_role_mapping"
description: |-
  Manage the custom roles associated with a LaunchDarkly team.
---

# launchdarkly_team_role_mapping

Provides a LaunchDarkly team to custom role mapping resource.

This resource allows you to manage the custom roles associated with a LaunchDarkly team. This is useful if the LaunchDarkly team is created and managed externally, such as via [team sync with SCIM](https://docs.launchdarkly.com/home/account-security/sso/scim#team-sync-with-scim). If you wish to create and manage the team using Terraform, we recommend using the [`launchdarkly_team` resource](https://registry.terraform.io/providers/launchdarkly/launchdarkly/latest/docs/resources/team) instead.

-> **Note:** Teams are available to customers on an Enterprise LaunchDarkly plan. To learn more, read about our pricing. To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).

## Example Usage

```hcl
resource "launchdarkly_team_role_mapping" "platform_team" {
  team_key         = "platform_team"
  custom_role_keys = ["platform", "nomad-administrators"]
}
```

## Argument Reference

- `team_key` - (Required) The team key.

- `custom_role_keys` - (Required) List of custom role keys the team will access. The referenced custom roles must already exist in LaunchDarkly. If they don't, the provider may behave unexpectedly.

## Import

A LaunchDarkly team/role mapping can be imported using the team key:

```
$ terraform import launchdarkly_team_role_mapping.platform_team platform_team
```
