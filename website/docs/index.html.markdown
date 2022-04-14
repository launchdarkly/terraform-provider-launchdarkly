---
layout: "launchdarkly"
page_title: "Provider: LaunchDarkly"
description: |-
  The LaunchDarkly provider is used to interact with the LaunchDarkly resources
---

# LaunchDarkly Provider

[LaunchDarkly](https://launchdarkly.com/) is a continuous delivery platform that provides feature flags as a service and allows developers to iterate quickly and safely. Use the LaunchDarkly provider to interact with LaunchDarkly resources, such as projects, environments, feature flags, and more. You must configure the provider with the proper credentials before you can use it.

## Example Usage

```hcl
terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "~> 2.0"
    }
  }
}

# Configure the LaunchDarkly provider
provider "launchdarkly" {
  access_token = var.launchdarkly_access_token
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

Please refer to [Terraform's documentation on upgrading to v0.13](https://www.terraform.io/upgrade-guides/0-13.html) for more information.

## Argument Reference

The provider supports the following arguments:

- `access_token` - (Optional) The [personal access token](https://docs.launchdarkly.com/docs/api-access-tokens) you use to authenticate with LaunchDarkly. You can also set this with the `LAUNCHDARKLY_ACCESS_TOKEN` environment variable. You must provide either `access_token` or `oauth_token`.

- `oauth_token` - (Optional) An OAuth V2 token you use to authenticate with LaunchDarkly. You can also set this with the `LAUNCHDARKLY_OAUTH_TOKEN` environment variable. You must provide either `access_token` or `oauth_token`.

- `api_host` - (Optional) The LaunchDarkly host address. If this argument is not specified, the default host address is `https://app.launchdarkly.com`.
