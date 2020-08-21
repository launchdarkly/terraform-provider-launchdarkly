---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_access_token"
description: |-
  Create and manage LaunchDarkly access tokens.
---

# launchdarkly_access

Provides a LaunchDarkly access token resource.

This resource allows you to create and manage access tokens within your LaunchDarkly organization.

-> **Note:** This resource will store the full plaintext secret for your access token in Terraform state. Be sure your state is configured securely before using this resource. See https://www.terraform.io/docs/state/sensitive-data.html for more details.

## Example Usage

With a built-in role

```hcl
resource "launchdarkly_access_token" "test" {
  name          = "Example token"
  role          = "reader"
  service_token = true
}
```

With a custom role

```hcl
resource "launchdarkly_custom_role" "example_role" {
  key         = "%s"
  name        = "Example role"
  description = "Allow all actions on production environments"
  policy {
    actions   = ["*"]
    effect    = "allow"
    resources = ["proj/*:env/production"]
  }
}

resource "launchdarkly_access_token" "test" {
  name          = "Example token"
  custom_roles  = [launchdarkly_custom_role.example_role.key]
  service_token = true
}
```

## Argument Reference


- `name` - (Required) The human-readable name for the access token.

- `service_token` - (Optional) Whether the token will be a [service token](https://docs.launchdarkly.com/home/account-security/api-access-tokens#service-tokens)
- `default_api_version` - (Optional) The default API version for this token. Defaults to the latest API version.

An access token may have its permissions specified by a built-in LaunchDarkly role, a set of custom role keys, or by an inline custom role policy.

- `role` - A built-in LaunchDarkly role. Can be `reader`, `writer`, or `admin`

- `custom_roles` - A set of custom role keys

- `policy_statements` - The access token policy block. To learn more, read [Policies in custom roles](https://docs.launchdarkly.com/docs/policies-in-custom-roles). May be specified more than once.

Access token `policy_statements` blocks are composed of the following arguments:

- `effect` - (Required) - Either `allow` or `deny`. This argument defines whether the statement allows or denies access to the named resources and actions.

- `resources` - (Optional) - The list of resource specifiers defining the resources to which the statement applies. Either `resources` or `not_resources` must be specified. For a list of available resources read [Understanding resource types and scopes](https://docs.launchdarkly.com/home/account-security/custom-roles/resources#understanding-resource-types-and-scopes).

- `not_resources` - (Optional) - The list of resource specifiers defining the resources to which the statement does not apply. Either `resources` or `not_resources` must be specified. For a list of available resources read [Understanding resource types and scopes](https://docs.launchdarkly.com/home/account-security/custom-roles/resources#understanding-resource-types-and-scopes).

- `actions` - (Optional) The list of action specifiers defining the actions to which the statement applies. Either `actions` or `not_actions` must be specified. For a list of available actions read [Actions reference](https://docs.launchdarkly.com/home/account-security/custom-roles/actions#actions-reference).

- `not_actions` - (Optional) The list of action specifiers defining the actions to which the statement does not apply. Either `actions` or `not_actions` must be specified. For a list of available actions read [Actions reference](https://docs.launchdarkly.com/home/account-security/custom-roles/actions#actions-reference).

- `expire` - (Optional) Replace the computed token secret with a new value. The expired secret will no longer be able to authorize usage of the LaunchDarkly API. `expire` should be an expiration time for the current token secret, expressed as a Unix epoch time in milliseconds. Setting this to a negative value will expire the existing token immediately. To reset the token value again, change 'expire' to a new value. Setting this field at resource creation time WILL NOT set an expiration time for the token.

## Attribute Reference

In addition to the arguments above, the resource exports the following attributes:

- `id` - The unique access token ID.

- `token` - The secret key for this token. Used to authenticate with the LaunchDarkly API.

