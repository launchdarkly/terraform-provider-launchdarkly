---
page_title: "launchdarkly_ip_allowlist_entry Resource - launchdarkly"
subcategory: ""
description: |-
  -> Note: IP allowlists are available to customers on an Enterprise LaunchDarkly plan. To learn more, read about our pricing https://launchdarkly.com/pricing/. To upgrade your plan, contact LaunchDarkly Sales https://launchdarkly.com/contact-sales/.
  Provides a LaunchDarkly IP allowlist entry resource.
  This resource allows you to create and manage IP allowlist entries within your LaunchDarkly account.
---

# launchdarkly_ip_allowlist_entry (Resource)

-> **Note:** IP allowlists are available to customers on an Enterprise LaunchDarkly plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).

Provides a LaunchDarkly IP allowlist entry resource.

This resource allows you to create and manage IP allowlist entries within your LaunchDarkly account.

## Example Usage

```terraform
resource "launchdarkly_ip_allowlist_entry" "office" {
  ip_address  = "203.0.113.0/24"
  description = "Office network"
}

resource "launchdarkly_ip_allowlist_entry" "vpn" {
  ip_address  = "198.51.100.1"
  description = "VPN endpoint"
}
```

## Schema

### Required

- `ip_address` (String) The IP address or CIDR block for the allowlist entry. Changing this forces a new resource to be created.

### Optional

- `description` (String) A human-readable description of the IP allowlist entry.

### Read-Only

- `id` (String) The UUID of the IP allowlist entry.

## Import

Import is supported using the following syntax:

```shell
# IP allowlist entries can be imported using their UUID
terraform import launchdarkly_ip_allowlist_entry.example c3d4e5f6-a7b8-9012-cdef-123456789012
```
