---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_webhook"
description: |-
  Create and manage LaunchDarkly webhooks.
---

# launchdarkly_webhook

Provides a LaunchDarkly webhook resource.

This resource allows you to create and manage webhooks within your LaunchDarkly organization.

## Example Usage

```hcl
resource "launchdarkly_webhook" "example" {
  url     = "http://webhooks.com/webhook"
  name    = "Example Webhook"
  tags    = ["terraform"]
  enabled = true

  policy_statements {
    actions     = ["*"]
    effect      = "allow"
    resources   = ["proj/*:env/production:flag/*"]
  }
  policy_statements {
    actions     = ["*"]
    effect      = "allow"
    resources   = resources = ["proj/test:env/production:segment/*"]
  }
}
```

## Argument Reference

- `url` - (Required) The URL of the remote webhook.

- `enabled` - (Required) Specifies whether the webhook is enabled.

- `name` - (Optional) The webhook's human-readable name.

- `secret` - (Optional) The secret used to sign the webhook.

- `tags` - (Optional) Set of tags associated with the webhook.

- `policy_statements` - (Optional) List of policy statement blocks used to filter webhook events. For more information on webhook policy filters read [Adding a policy filter](https://docs.launchdarkly.com/integrations/webhooks#adding-a-policy-filter)

Webhook `policy_statements` blocks are composed of the following arguments:

- `effect` - (Required) Either `allow` or `deny`. This argument defines whether the statement allows or denies access to the named resources and actions.

- `resources` - (Optional) The list of resource specifiers defining the resources to which the statement applies. Either `resources` or `not_resources` must be specified. For a list of available resources read [Understanding resource types and scopes](https://docs.launchdarkly.com/home/account-security/custom-roles/resources#understanding-resource-types-and-scopes).

- `not_resources` - (Optional) The list of resource specifiers defining the resources to which the statement does not apply. Either `resources` or `not_resources` must be specified. For a list of available resources read [Understanding resource types and scopes](https://docs.launchdarkly.com/home/account-security/custom-roles/resources#understanding-resource-types-and-scopes).

- `actions` - (Optional) The list of action specifiers defining the actions to which the statement applies. Either `actions` or `not_actions` must be specified. For a list of available actions read [Actions reference](https://docs.launchdarkly.com/home/account-security/custom-roles/actions#actions-reference).

- `not_actions` - (Optional) The list of action specifiers defining the actions to which the statement does not apply. Either `actions` or `not_actions` must be specified. For a list of available actions read [Actions reference](https://docs.launchdarkly.com/home/account-security/custom-roles/actions#actions-reference).

## Attributes Reference

In addition to the arguments above, the resource exports following attribute:

- `id` - The unique webhook ID.

## Import

LaunchDarkly webhooks can be imported using the webhook's 24 character ID, e.g.

```
$ terraform import launchdarkly_webhook.example 57c0af609969090743529967
```
