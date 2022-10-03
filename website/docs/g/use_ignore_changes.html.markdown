---
layout: "launchdarkly"
page_title: "LaunchDarkly: Use ignore_changes to create resources with Terraform and update them in the UI"
description: |-
  This guide covers 
---

# Using `ignore_changes` 

This guide explains when and how to use [ignore_changes lifecycle metadata](https://www.terraform.io/language/meta-arguments/lifecycle#ignore_changes) to avoid having Terraform try to update resources that were modified. 

## When to use `ignore_changes`

### Use Terraform to create a resource in LaunchDarkly, and manage the resource through the UI

For example, you might provision teams in LaunchDarkly using Terraform, then allow team members to add new members to the team in the UI without needing to udpate Terraform. This is a case where you simply want to *create* the team using Terraform, but then manage the team for the rest of it's lifecycle through the UI. In order to continue managing the team using Terraform you would need to manually update your manifest to reflect the current state of the team, and then apply.

```
data "launchdarkly_team_member" "spongebob" {
  email = "spongebob@squarepants.net"
}

resource "launchdarkly_team" "krusty_krab_staff" {
  key                   = "krusty_krab_staff"
  name                  = "Krusty Krab staff"
  description           = "Team serving Krabby patties"
  members               = [data.launchdarkly_team_member.spongebob.id]

  lifecycle {
    ignore_changes = [member_ids]
  }
}
```

### When a resource is modified as a side-effect of other actions in LaunchDarkly

Sometimes resources in LaunchDarkly are modified as a side-effect of other actions in LaunchDarkly. For example, if you create an experiment using a flag, and then try to apply a Terraform manifest that manages that flag, it will fail. As a workaround for this you can use `ignore_changes` to tell Terraform to not try to update the modified resources.

```
resource "launchdarkly_feature_flag" "example" {
  project_key = launchdarkly_project.example.key
  key         = "example-flag"
  name        = "Example flag"
  description = "This demonstrates using ignore_changes"

  variation_type = "boolean"
  variations {
    value       = "true"
    name        = "True"
  }
  variations {
    value       = "false"
    name        = "False"
  }
  
  defaults {
    on_variation = 1
    off_variation = 0
  }

  lifecycle {
    ignore_changes = [all]
  }
}
```
