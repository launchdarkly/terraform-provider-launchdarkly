---
title: "launchdarkly_relay_proxy_configuration"
description: "Create and manage Relay Proxy configurations"
---

# launchdarkly_relay_proxy_configuration

Provides a LaunchDarkly Relay Proxy configuration resource for use with the Relay Proxy's [automatic configuration feature](https://docs.launchdarkly.com/home/relay-proxy/automatic-configuration).

-> **Note:** Relay Proxy automatic configuration is available to customers on an Enterprise LaunchDarkly plan. To learn more, read about our pricing. To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).

This resource allows you to create and manage Relay Proxy configurations within your LaunchDarkly organization.

-> **Note:** This resource will store the full plaintext secret for your Relay Proxy configuration's unique key in Terraform state. Be sure your state is configured securely before using this resource. See https://www.terraform.io/docs/state/sensitive-data.html for more details.

## Example Usage

```hcl
resource "launchdarkly_relay_proxy_configuration" "example" {
	name = "example-config"
	policy {
		actions   = ["*"]
		effect    = "allow"
		resources = ["proj/*:env/*"]
	}
}
```

## Argument Reference

- `name` - (Required) The human-readable name for your Relay Proxy configuration.

- `policy` - (Required) The Relay Proxy configuration's rule policy block. This determines what content the Relay Proxy receives. To learn more, read [Understanding policies](https://docs.launchdarkly.com/home/members/role-policies#understanding-policies).

Relay proxy configuration `policy` blocks are composed of the following arguments

- `effect` - (Required) - Either `allow` or `deny`. This argument defines whether the rule policy allows or denies access to the named resources and actions.

- `resources` - (Optional) - The list of resource specifiers defining the resources to which the rule policy applies. Either `resources` or `not_resources` must be specified. For a list of available resources read [Understanding resource types and scopes](https://docs.launchdarkly.com/home/account-security/custom-roles/resources#understanding-resource-types-and-scopes).

- `not_resources` - (Optional) - The list of resource specifiers defining the resources to which the rule policy does not apply. Either `resources` or `not_resources` must be specified. For a list of available resources read [Understanding resource types and scopes](https://docs.launchdarkly.com/home/account-security/custom-roles/resources#understanding-resource-types-and-scopes).

- `actions` - (Optional) The list of action specifiers defining the actions to which the rule policy applies. Either `actions` or `not_actions` must be specified. For a list of available actions read [Actions reference](https://docs.launchdarkly.com/home/account-security/custom-roles/actions#actions-reference).

- `not_actions` - (Optional) The list of action specifiers defining the actions to which the rule policy does not apply. Either `actions` or `not_actions` must be specified. For a list of available actions read [Actions reference](https://docs.launchdarkly.com/home/account-security/custom-roles/actions#actions-reference).

## Attribute Reference

- `id` - The Relay Proxy configuration's ID

- `full_key` - The Relay Proxy configuration's unique key. Because the `full_key` is only exposed upon creation, it will not be available if the resource is imported.

- `display_key` - The last 4 characters of the Relay Proxy configuration's unique key.

## Import

Relay Proxy configurations can be imported using the configuration's unique 24 character ID, e.g.

```shell-session
$ terraform import launchdarkly_relay_proxy_configuration.example 51d440e30c9ff61457c710f6
```
