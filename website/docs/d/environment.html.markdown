---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_environment"
description: |-
  Get information about LaunchDarkly environments.
---

# launchdarkly_environment

Provides a LaunchDarkly environment data source.

This data source allows you to retrieve environment information from your LaunchDarkly organization.

## Example Usage

```hcl
data "launchdarkly_environment" "example" {
  key         = "example-env"
  project_key = "example-project"
}
```

## Argument Reference

- `key` - (Required) The environment's unique key.

- `project_key` - (Required) The environment's project key.

## Attributes Reference

In addition to the arguments above, the resource exports the following attributes:

- `id` - The unique environment ID in the format `project_key/environment_key`.

- `name` - The name of the environment.

- `color` - The color swatch as an RGB hex value with no leading `#`. For example: `000000`.

- `tags` - Set of tags associated with the environment.

- `secure_mode` - A value of true `true` ensures a user of the client-side SDK cannot impersonate another user.

- `default_track_events` - A value of `true` enables data export for every flag created in this environment. To learn more, read [Data Export](https://docs.launchdarkly.com/home/data-export).

- `default_ttl` - The TTL for the environment. This will be a numeric value between 0 and 60 in minutes. The TTL setting only applies to environments using the PHP SDK. To learn more, read [TTL settings](https://docs.launchdarkly.com/home/organize/environments#ttl-settings).

- `require_comments` - A value of `true` indicates that this environment requires comments for flag and segment changes. 

- `confirm_changes` - A value of `true` indicates that this environment requires confirmation for flag and segment changes.

- `api_key` - The environment's SDK key.

- `mobile_key` - The environment's mobile key.

- `client_side_id` - The environment's client-side ID.
