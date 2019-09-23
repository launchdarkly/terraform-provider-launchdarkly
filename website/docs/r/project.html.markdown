---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_project"
description: |-
  Create and manage LaunchDarkly projects.
---

# launchdarkly_project

Provides a LaunchDarkly project resource.

This resource allows you to create and manage projects within your LaunchDarkly organization.

## Example Usage

```hcl
resource "launchdarkly_project" "example" {
  key  = "example-project"
  name = "Example project"

  tags = [
    "terraform",
  ]
}
```

## Argument Reference

- `key` - (Required) The project's unique key.

- `name` - (Required) The project's name.

- `tags` - (Optional) The project's set of tags.

## Import

LaunchDarkly projects can be imported using the project's key, e.g.

```
$ terraform import launchdarkly_project.example example-project
```
