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
  url  = "http://webhooks.com/webhook"
  name = "Example Webhook"
  tags = ["terraform"]
  on   = false
}
```

## Argument Reference

- `url` - (Required) - The URL of the remote webhook.

- `on` - (Required) - Specifies whether the webhook is enabled.

- `name` - (Optional) - The webhook's human-readable name.

- `secret` - (Optional) - The secret used to sign the webhook.

- `tags` - (Optional) - Set of tags associated with the webhook.

## Attributes Reference

In addition to the arguments above, the resource exports following attribute:

- `id` - The unique webhook ID.

## Import

LaunchDarkly webhooks can be imported using the webhook's 24 character ID, e.g.

```
$ terraform import launchdarkly_webhook.example 57c0af609969090743529967
```
