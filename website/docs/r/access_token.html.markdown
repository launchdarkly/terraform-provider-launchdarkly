---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_access_token"
description: |-
  Create and manage LaunchDarkly access tokens.
---

# launchdarkly_access_token

Provides a LaunchDarkly access token resource.

This resource allows you to create and manage access tokens within your LaunchDarkly organization.

-> **Note:** This resource will store the full plaintext secret for your access token in Terraform state. Be sure your state is configured securely before using this resource. See https://www.terraform.io/docs/state/sensitive-data.html for more details.

## Example Usage

The resource must contain either a `role`, `custom_role` or an `inline_roles` (previously `policy_statements`) block. As of v1.7.0, `policy_statements` has been deprecated in favor of `inline_roles`.

With a built-in role

```hcl
resource "launchdarkly_access_token" "reader_token" {
  name = "Reader token managed by Terraform"
  role = "reader"
}
```

With a custom role

```hcl
resource "launchdarkly_access_token" "custom_role_token" {
  name = "DevOps"
  custom_roles  = ["ops"]
}
```

With an inline custom role (policy statements)

```hcl
resource "launchdarkly_access_token" "token_with_policy_statements" {
  name = "Integration service token"
  inline_roles {
    actions   = ["*"]
    effect    = "deny"
    resources = ["proj/*:env/production"]
  }
  service_token = true
}
```

## Argument Reference

- `name` - (Optional) A human-friendly name for the access token.

- `service_token` - (Optional) Whether the token will be a [service token](https://docs.launchdarkly.com/home/account-security/api-access-tokens#service-tokens). A change in this field will force the destruction of the existing token and the creation of a new one.

- `default_api_version` - (Optional) The default API version for this token. Defaults to the latest API version. A change in this field will force the destruction of the existing token in state and the creation of a new one.

An access token may have its permissions specified by a built-in LaunchDarkly role, a set of custom role keys, or by an inline custom role (policy statements).

- `role` - (Optional) A built-in LaunchDarkly role. Can be `reader`, `writer`, or `admin`

- `custom_roles` - (Optional) A list of custom role IDs to use as access limits for the access token

- `policy_statements` - (Optional, **Deprecated**) Define inline custom roles. An array of statements represented as config blocks with 3 attributes: effect, resources, actions. May be used in place of a built-in or custom role. [Policies in custom roles](https://docs.launchdarkly.com/docs/policies-in-custom-roles). May be specified more than once. This field argument is **deprecated**. Please update your config to use `inline_role` to maintain compatibility with future versions.

- `inline_role` - (Optional) Define inline custom roles. An array of statements represented as config blocks with 3 attributes: effect, resources, actions. May be used in place of a built-in or custom role. [Policies in custom roles](https://docs.launchdarkly.com/docs/policies-in-custom-roles). May be specified more than once.

Access token `policy_statements` and `inline_role` blocks are composed of the following arguments:

- `effect` - (Required) - Either `allow` or `deny`. This argument defines whether the statement allows or denies access to the named resources and actions.

Either `resources` or `not_resources` must be specified. For a list of available resources read [Understanding resource types and scopes](https://docs.launchdarkly.com/home/account-security/custom-roles/resources#understanding-resource-types-and-scopes).

- `resources` - (Optional) - The list of resource specifiers defining the resources to which the statement applies.

- `not_resources` - (Optional) - The list of resource specifiers defining the resources to which the statement does not apply.

Either `actions` or `not_actions` must be specified. For a list of available actions read [Actions reference](https://docs.launchdarkly.com/home/account-security/custom-roles/actions#actions-reference).

- `actions` - (Optional) The list of action specifiers defining the actions to which the statement applies.

- `not_actions` - (Optional) The list of action specifiers defining the actions to which the statement does not apply.

* `expire` - (Optional, **Deprecated**) An expiration time for the current token secret, expressed as a Unix epoch time. Replace the computed token secret with a new value. The expired secret will no longer be able to authorize usage of the LaunchDarkly API. This field argument is **deprecated**. Please update your config to remove `expire` to maintain compatibility with future versions.

## Attribute Reference

In addition to the arguments above, the resource exports the following attributes:

- `id` - The unique access token ID.

- `token` - The access token used to authorize usage of the LaunchDarkly API.
