---
layout: "launchdarkly"
page_title: "Provider: LaunchDarkly"
description: |-
  The LaunchDarkly provider is used to interact with the LaunchDarkly resources
---

# LaunchDarkly Provider

[LaunchDarkly](https://launchdarkly.com/) is a continuous delivery platform that provides feature flags as a service and allows developers to iterate quickly and safely. The LaunchDarkly provider is used to interact with the LaunchDarkly resources, such as project, environments, feature flags and more. The provider needs to be configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the LaunchDarkly provider
provider "launchdarkly" {
  api_key = var.launchdarkly_api_key
}

# Create a new project
resource "launchdarkly_project" "terraform" {
  # ...
}

# Create a new feature flag
resource "launchdarkly_feature_flag" "terraform" {
  # ...
}
```

## Argument Reference

The following argument is supported:

- `api_key` - (Required) The [personal access token](https://docs.launchdarkly.com/docs/api-access-tokens) used to authenticate with LaunchDarkly. This can also be set via the `LAUNCHDARKLY_API_KEY` environment variable.
