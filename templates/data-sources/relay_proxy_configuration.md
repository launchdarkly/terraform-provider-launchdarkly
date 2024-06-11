---
title: "launchdarkly_relay_proxy_configuration"
description: "Get information about Relay Proxy configurations."
---

# launchdarkly_relay_proxy_configuration

Provides a LaunchDarkly Relay Proxy configuration data source for use with the Relay Proxy's [automatic configuration feature](https://docs.launchdarkly.com/home/relay-proxy/automatic-configuration).

-> **Note:** Relay Proxy automatic configuration is available to customers on an Enterprise LaunchDarkly plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).

This data source allows you to retrieve Relay Proxy configuration information from your LaunchDarkly organization.

-> **Note:** It is not possible for this data source to retrieve your Relay Proxy configuration's unique key. This is because the unique key is only exposed upon creation. If you need to reference the Relay Proxy configuration's unique key in your terraform config, use the `launchdarkly_relay_proxy_configuration` resource instead.

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

- `id` - (Required) The Relay Proxy configuration's unique 24 character ID. The unique relay proxy ID can be found in the relay proxy edit page URL, which you can locate by clicking the three dot menu on your relay proxy item in the UI and selecting 'Edit configuration':

```
https://app.launchdarkly.com/settings/relay/THIS_IS_YOUR_RELAY_PROXY_ID/edit
```

## Attribute Reference

In addition to the argument above, the resource exports the following attributes:

- `name` - The human-readable name for your Relay Proxy configuration.

- `display_key` - The last 4 characters of the Relay Proxy configuration's unique key.

- `policy` - The Relay Proxy configuration's rule policy block. This determines what content the Relay Proxy receives. To learn more, read [Understanding policies](https://docs.launchdarkly.com/home/members/role-policies#understanding-policies).

Relay proxy configuration `policy` blocks are composed of the following arguments:

- `effect` - Either `allow` or `deny`. This argument defines whether the rule policy allows or denies access to the named resources and actions.

- `resources` - The list of resource specifiers defining the resources to which the rule policy applies. Either `resources` or `not_resources` must be specified. For a list of available resources read [Understanding resource types and scopes](https://docs.launchdarkly.com/home/account-security/custom-roles/resources#understanding-resource-types-and-scopes).

- `not_resources` - The list of resource specifiers defining the resources to which the rule policy does not apply. Either `resources` or `not_resources` must be specified. For a list of available resources read [Understanding resource types and scopes](https://docs.launchdarkly.com/home/account-security/custom-roles/resources#understanding-resource-types-and-scopes).

- `actions` The list of action specifiers defining the actions to which the rule policy applies. Either `actions` or `not_actions` must be specified. For a list of available actions read [Actions reference](https://docs.launchdarkly.com/home/account-security/custom-roles/actions#actions-reference).

- `not_actions` The list of action specifiers defining the actions to which the rule policy does not apply. Either `actions` or `not_actions` must be specified. For a list of available actions read [Actions reference](https://docs.launchdarkly.com/home/account-security/custom-roles/actions#actions-reference).
