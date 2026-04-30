---
page_title: "launchdarkly_ip_allowlist_config Resource - launchdarkly"
subcategory: ""
description: |-
  -> Note: IP allowlists are available to customers on an Enterprise LaunchDarkly plan. To learn more, read about our pricing https://launchdarkly.com/pricing/. To upgrade your plan, contact LaunchDarkly Sales https://launchdarkly.com/contact-sales/.
  Provides a LaunchDarkly IP allowlist configuration resource.
  This resource allows you to manage the IP allowlist configuration for your LaunchDarkly account.
---

# launchdarkly_ip_allowlist_config (Resource)

-> **Note:** IP allowlists are available to customers on an Enterprise LaunchDarkly plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).

Provides a LaunchDarkly IP allowlist configuration resource.

This resource allows you to manage the IP allowlist configuration for your LaunchDarkly account. There is only one configuration per account, so only a single instance of this resource should be defined.

## Example Usage

```terraform
resource "launchdarkly_ip_allowlist_config" "example" {
  session_allowlist_enabled = true
  scoped_allowlist_enabled  = true
}
```

## Schema

### Optional

- `session_allowlist_enabled` (Boolean) Whether the session IP allowlist is enabled. Defaults to `false`.
- `scoped_allowlist_enabled` (Boolean) Whether the scoped (API token) IP allowlist is enabled. Defaults to `false`.

### Read-Only

- `id` (String) The ID of this resource (always `ip-allowlist-config`).

~> **Note:** Destroying this resource will reset both `session_allowlist_enabled` and `scoped_allowlist_enabled` to `false`.

## Import

Import is supported using the following syntax:

```shell
terraform import launchdarkly_ip_allowlist_config.example ip-allowlist-config
```
