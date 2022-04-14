---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_flag_trigger"
description: |-
  Create and manage LaunchDarkly flag triggers.
---

# launchdarkly_flag_trigger

Provides a LaunchDarkly flag trigger resource.

-> **Note:** Flag triggers are available to customers on an Enterprise LaunchDarkly plan. To learn more, read about our pricing. To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).

This resource allows you to create and manage flag triggers within your LaunchDarkly organization.

-> **Note:** This resource will store sensitive unique trigger URL value in plaintext in your Terraform state. Be sure your state is configured securely before using this resource. See https://www.terraform.io/docs/state/sensitive-data.html for more details.

## Example Usage

```hcl
resource "launchdarkly_flag_trigger" "example" {
	project_key = launchdarkly_project.example.key
	env_key = "test"
	flag_key = launchdarkly_feature_flag.trigger_flag.key
	integration_key = "generic-trigger"
	instructions {
		kind = "turnFlagOn"
	}
	enabled = false
}
```

## Argument Reference

- `project_key` - (Required) The unique key of the project encompassing the associated flag. A change in this field will force the destruction of the existing resource and the creation of a new one.

- `env_key` - (Required) The unique key of the environment the flag trigger will work in. A change in this field will force the destruction of the existing resource and the creation of a new one.

- `flag_key` - (Required) The unique key of the associated flag. A change in this field will force the destruction of the existing resource and the creation of a new one.

- `integration_key` - (Required) The unique identifier of the integration you intend to set your trigger up with. Currently supported are `"datadog"`, `"dynatrace"`, `"honeycomb"`, `"new-relic-apm"`, `"signalfx"`, and `"generic-trigger"`. `"generic-trigger"` should be used for integrations not explicitly supported. A change in this field will force the destruction of the existing resource and the creation of a new one.

- `instructions` - (Required) Instructions containing the action to perform when invoking the trigger. Currently supported flag actions are `"turnFlagOn"` and `"turnFlagOff"`. This must be passed as the key-value pair `{ kind = "<flag_action>" }`.

- `enabled` - (Optional) Whether the trigger is currently active or not. This property defaults to true upon creation and will thereafter conform to the last Terraform-configured value.

## Additional Attributes

In addition to the above arguments, this resource supports the following computed attributes:

`trigger_url` - The unique URL used to invoke the trigger.

`maintainer_id` - The ID of the member responsible for maintaining the flag trigger. If created via Terraform, this value will be the ID of the member associated with the API key used for your provider configuration.

## Import

LaunchDarkly flag triggers can be imported using the following syntax:

```
$ terraform import launchdarkly_flag_trigger.example example-project-key/example-env-key/example-flag-key/62581d4488def814b831abc3
```

where the string following the final slash is your unique trigger ID.

The unique trigger ID can be found in your saved trigger URL:

```
https://app.launchdarkly.com/webhook/triggers/THIS_IS_YOUR_TRIGGER_ID/aff25a53-17d9-4112-a9b8-12718d1a2e79
```

Please note that if you did not save this upon creation of the resource, you will have to reset it to get a new value, which can cause breaking changes.
