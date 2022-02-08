---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_audit_log_subscription"
description: |-
  Create and manage LaunchDarkly integration audit log subscriptions.
---

# launchdarkly_audit_log_subscription

Provides a LaunchDarkly audit log subscription resource.

This resource allows you to create and manage LaunchDarkly audit log subscriptions.

# Example Usage

```hcl
resource "launchdarkly_audit_log_subscription" "example" {
	integration_key = "datadog"
	name = "Example Datadog Subscription"
	config {
        api_key = "yoursecretkey"
		host_url = "https://api.datadoghq.com"
    }
	tags = [
		"integrations",
		"terraform"
	]
	statements {
		actions = ["*"]
		effect = "allow"
		resources = ["proj/*:env/*:flag/*"]
	}
}
```

## Argument Reference

- `integration_key` (Required) The integration key. As of January 2022, supported integrations are `"datadog"`, `"dynatrace"`, `"elastic"`, `"honeycomb"`, `"logdna"`, `"msteams"`, `"new-relic-apm"`, `"signalfx"`, `"slack"`, and `"splunk"`. A change in this field will force the destruction of the existing resource and the creation of a new one.

- `name` (Required) - A human-friendly name for your audit log subscription viewable from within the LaunchDarkly Integrations page.

- `config` (Required) - The set of configuration fields corresponding to the value defined for `integration_key`. Refer to the `"formVariables"` field in the corresponding `integrations/<integration_key>/manifest.json` file in [this repo](https://github.com/launchdarkly/integration-framework/tree/master/integrations) for a full list of fields for the integration you wish to configure. **IMPORTANT**: Please note that Terraform will only accept these in snake case, regardless of the case shown in the manifest.

- `statements` (Required) - A block representing the resources to which you wish to subscribe. To learn more about how to configure these blocks, read [Nested Subscription Statements Blocks](#nested-subscription-statements-blocks).

- `on` (Required) - Whether or not you want your subscription enabled, i.e. to actively send events.

- `tags` (Optional) - Set of tags associated with the subscription object.

### Nested Subscription Statements Blocks

Nested subscription `statements` blocks have the following structure:

- `effect` (Required) - Either `allow` or `deny`. This argument defines whether the statement allows or denies access to the named resources and actions.

- `resources` - The list of resource specifiers defining the resources to which the statement applies. To learn more about how to configure these, read [Using resources](https://docs.launchdarkly.com/home/members/role-resources).

- `not_resources` - The list of resource specifiers defining the resources to which the statement does not apply. To learn more about how to configure these, read [Using resources](https://docs.launchdarkly.com/home/members/role-resources).

- `actions` The list of action specifiers defining the actions to which the statement applies. For a list of available actions, read [Using actions](https://docs.launchdarkly.com/home/members/role-actions).

- `not_actions` The list of action specifiers defining the actions to which the statement does not apply. For a list of available actions, read [Using actions](https://docs.launchdarkly.com/home/members/role-actions).

Please note that either `resources` and `actions` _or_ `not_resources` and `not_actions` must be defined.
