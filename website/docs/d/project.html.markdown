---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_project"
description: |-
  Get information about LaunchDarkly projects.
---

# launchdarkly_project

Provides a LaunchDarkly project data source.

This data source allows you to retrieve project information from your LaunchDarkly organization.

-> **Note:** LaunchDarkly data sources do not provide access to the project's environments. If you wish to import environment configurations as data sources you must use the [`launchdarkly_environment` data source](/docs/providers/launchdarkly/d/environment.html).

## Example Usage

```hcl
data "launchdarkly_project" "example" {
  key = "example-project"
}
```

## Argument Reference

- `key` - (Required) The project's unique key.

## Attributes Reference

In addition to the arguments above, the resource exports the following attributes:

- `name` - The project's name.

- `client_side_availability` - A map describing whether flags in this project are available to the client-side JavaScript SDK by default. To learn more, read [Nested Client-Side Availability Block](#nested-client-side-availability-block).

- `tags` - The project's set of tags.

### Nested Client-Side Availibility Block

The nested `client_side_availability` block has the following attributes:

- `using_environment_id` - When set to true, the flags in this project are available to SDKs using the client-side ID by default.

- `using_mobile_key` - When set to true, the flags in this project are available to SDKs using a mobile key by default.
