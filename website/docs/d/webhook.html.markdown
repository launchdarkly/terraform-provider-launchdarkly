---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_webhook"
description: |-
  Get information about LaunchDarkly webhooks.
---

# launchdarkly_webhook

Provides a LaunchDarkly webhook data source.

This data source allows you to retrieve webhook information from your LaunchDarkly organization.

## Example Usage

```hcl
data "launchdarkly_webhook" "example" {
  id = "57c0af6099690907435299"
}
```

## Argument Reference
- `id` - (Required) The unique webhook ID.

## Attributes Reference

In addition to the arguments above, the resource exports following attributes:

- `url` - The URL of the remote webhook.

- `enabled` - Whether the webhook is enabled.

- `name` - The webhook's human-readable name.

- `secret` - The secret used to sign the webhook.

- `tags` - Set of tags associated with the webhook.

- `policy_statements` - List of policy statement blocks used to filter webhook events. For more information on webhook policy filters read [Adding a policy filter](https://docs.launchdarkly.com/integrations/webhooks#adding-a-policy-filter). To learn more, read [Policy Statement Blocks](#policy-statement-blocks).

### Policy Statement Blocks

Webhook `policy_statements` blocks are composed of the following arguments:

- `effect` - Either `allow` or `deny`. This argument defines whether the statement allows or denies access to the named resources and actions.

- `resources` - The list of resource specifiers defining the resources to which the statement applies. For a list of available resources read [Understanding resource types and scopes](https://docs.launchdarkly.com/home/account-security/custom-roles/resources#understanding-resource-types-and-scopes).

- `not_resources` - The list of resource specifiers defining the resources to which the statement does not apply. For a list of available resources read [Understanding resource types and scopes](https://docs.launchdarkly.com/home/account-security/custom-roles/resources#understanding-resource-types-and-scopes).

- `actions` The list of action specifiers defining the actions to which the statement applies. For a list of available actions read [Actions reference](https://docs.launchdarkly.com/home/account-security/custom-roles/actions#actions-reference).

- `not_actions` The list of action specifiers defining the actions to which the statement does not apply. For a list of available actions read [Actions reference](https://docs.launchdarkly.com/home/account-security/custom-roles/actions#actions-reference).