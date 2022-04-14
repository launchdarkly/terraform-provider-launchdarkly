---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_flag_trigger"
description: |-
  Get information about LaunchDarkly flag trigers.
---

# launchdarkly_flag_trigger

Provides a LaunchDarkly flag trigger data source.

-> **Note:** Flag triggers are available to customers on an Enterprise LaunchDarkly plan. To learn more, read about our pricing. To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).

This data source allows you to retrieve information about flag triggers from your LaunchDarkly organization.

## Example Usage

```hcl
data "launchdarkly_flag_trigger" "example" {
	id = "<project_key>/<env_key>/<flag_key>/61d490757f7821150815518f"
	integration_key = "datadog"
	instructions {
		kind = "turnFlagOff"
	}
}
```

## Argument Reference

- `id` - (Required) The Terraform trigger ID. This ID takes the following format: `<project_key>/<env_key>/<flag_key>/<trigger_id>`. The unique trigger ID can be found in your saved trigger URL:

```
https://app.launchdarkly.com/webhook/triggers/THIS_IS_YOUR_TRIGGER_ID/aff25a53-17d9-4112-a9b8-12718d1a2e79
```

Please note that if you did not save this upon creation of the resource, you will have to reset it to get a new value, which can cause breaking changes.

## Attributes Reference

In addition to the arguments above, the resource exports the following attributes:

- `project_key` - The unique key of the project encompassing the associated flag.

- `env_key` - The unique key of the environment the flag trigger will work in.

- `flag_key` - The unique key of the associated flag.

- `integration_key` - The unique identifier of the integration your trigger is set up with.

- `instructions` - Instructions containing the action to perform when invoking the trigger. Currently supported flag actions are `"turnFlagOn"` and `"turnFlagOff"`. These can be found on the `kind` field nested on the `instructions` attribute.

- `maintainer_id` - The ID of the member responsible for maintaining the flag trigger. If created via Terraform, this value will be the ID of the member associated with the API key used for your provider configuration.

- `enabled` - Whether the trigger is currently active or not.

Please note that the original trigger URL itself will not be surfaced.
