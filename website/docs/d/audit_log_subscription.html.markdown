---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_audit_log_subscription"
description: |-
  Get information about LaunchDarkly audit log subscriptions.
---

# launchdarkly_audit_log_subscription

Provides a LaunchDarkly audit log subscription data source.

This data source allows you to retrieve information about LaunchDarkly audit log subscriptions.

# Example Usage

```hcl
data "launchdarkly_audit_log_subscription" "test" {
	id = "5f0cd446a77cba0b4c5644a7"
	integration_key = "msteams"
}
```

## Argument Reference

- `id` (Required) - The unique subscription ID. This can be found in the URL of the pull-out configuration sidebar for the given subscription on your [LaunchDarkly Integrations page](https://app.launchdarkly.com/default/integrations).

- `integration_key` (Required) - The integration key. As of January 2022, supported integrations are `"datadog"`, `"dynatrace"`, `"elastic"`, `"honeycomb"`, `"logdna"`, `"msteams"`, `"new-relic-apm"`, `"signalfx"`, and `"splunk"`.

## Attributes Reference

In addition to the arguments above, the resource exports following attributes:

- `name` - The subscription's human-readable name.

- `config` - A block of configuration fields associated with your integration type.

- `statements` - The statement block used to filter subscription events. To learn more, read [Statement Blocks](#statement-blocks).

- `on` - Whether the subscription is enabled.

- `tags` - Set of tags associated with the subscription.

### Statement Blocks

Audit log subscription `statements` blocks are composed of the following arguments:

- `effect` - Either `allow` or `deny`. This argument defines whether the statement allows or denies access to the named resources and actions.

- `resources` - The list of resource specifiers defining the resources to which the statement applies. To learn more about how to configure these read [Using resources](https://docs.launchdarkly.com/home/members/role-resources).

- `not_resources` - The list of resource specifiers defining the resources to which the statement does not apply. To learn more about how to configure these, read [Using resources](https://docs.launchdarkly.com/home/members/role-resources).

- `actions` The list of action specifiers defining the actions to which the statement applies. For a list of available actions, read [Using actions](https://docs.launchdarkly.com/home/members/role-actions).

- `not_actions` The list of action specifiers defining the actions to which the statement does not apply. For a list of available actions, read [Using actions](https://docs.launchdarkly.com/home/members/role-actions).
