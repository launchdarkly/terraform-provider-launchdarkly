---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_environment"
description: |-
  Create and manage LaunchDarkly environments.
---

# launchdarkly_environment

Provides a LaunchDarkly environment resource.

This resource allows you to create and manage environments in your LaunchDarkly organization. This resource should _not_ be used if the encapsulated project is also managed via Terraform. In this case, you should _always_ use the nested environments config blocks on your[`launchdarkly_project`](/docs/providers/launchdarkly/r/project.html) resource to manage your environments.

-> **Note:** Mixing the use of nested `environments` blocks in the [`launchdarkly_project`](/docs/providers/launchdarkly/r/project.html) resource and `launchdarkly_environment` resources is not recommended.

## Example Usage

```hcl
resource "launchdarkly_environment" "staging" {
  name  = "Staging"
  key   = "staging"
  color = "ff00ff"
  tags  = ["terraform", "staging"]

  project_key = launchdarkly_project.example.key
}
```

```hcl
resource "launchdarkly_environment" "approvals_example" {
  name  = "Approvals Example Environment"
  key   = "approvals-example"
  color = "ff00ff"
  tags  = ["terraform", "staging"]

  approval_settings {
    required                   = true
    can_review_own_request     = true
    min_num_approvals          = 2
    can_apply_declined_changes = true
  }

  project_key = launchdarkly_project.example.key
}
```

## Argument Reference

- `project_key` - (Required) - The environment's project key. A change in this field will force the destruction of the existing resource and the creation of a new one.

- `name` - (Required) The name of the environment.

- `key` - (Required) The project-unique key for the environment. A change in this field will force the destruction of the existing resource and the creation of a new one.

- `color` - (Required) The color swatch as an RGB hex value with no leading `#`. For example: `000000`.

- `tags` - (Optional) Set of tags associated with the environment.

- `secure_mode` - (Optional) Set to `true` to ensure a user of the client-side SDK cannot impersonate another user. This field will default to `false` when not set.

- `default_track_events` - (Optional) Set to `true` to enable data export for every flag created in this environment after you configure this argument. This field will default to `false` when not set. To learn more, read [Data Export](https://docs.launchdarkly.com/docs/data-export).

- `default_ttl` - (Optional) The TTL for the environment. This must be between 0 and 60 minutes. The TTL setting only applies to environments using the PHP SDK. This field will default to `0` when not set. To learn more, read [TTL settings](https://docs.launchdarkly.com/docs/environments#section-ttl-settings).

- `require_comments` - (Optional) Set to `true` if this environment requires comments for flag and segment changes. This field will default to `false` when not set.

- `confirm_changes` - (Optional) Set to `true` if this environment requires confirmation for flag and segment changes. This field will default to `false` when not set.

- `approval_settings` - (Optional) A nested block describing the environment approval settings. To learn more about this feature, read [Approvals](https://docs.launchdarkly.com/home/feature-workflows/approvals). To learn more about configuring them in Terraform, read [Nested Approval Settings Blocks](#nested-approval-settings-blocks).

## Attribute Reference

In addition to the arguments above, the resource exports the following attributes:

- `id` - The unique environment ID in the format `project_key/environment_key`.

- `api_key` - The environment's SDK key.

- `mobile_key` - The environment's mobile key.

- `client_side_id` - The environment's client-side ID.

### Nested Approval Settings Blocks

Nested `approval_settings` blocks have the following structure:

- `required` - Set to `true` for changes to flags in this environment to require approval. You may only set `required` to true if `required_approval_tags` is not set and vice versa. Defaults to `false`.

- `can_review_own_request` - Set to `true` if requesters can approve or decline their own request. They may always comment. Defaults to `false`.

- `min_num_approvals` - The number of approvals required before an approval request can be applied. This number must be between 1 and 5. Defaults to 1.

- `can_apply_declined_changes` - Set to `true` if changes can be applied as long as the `min_num_approvals` is met, regardless of whether any reviewers have declined a request. Defaults to `true`.

- `required_approval_tags` - An array of tags used to specify which flags with those tags require approval. You may only set `required_approval_tags` if `required` is not set to `true` and vice versa.

## Import

You can import a LaunchDarkly environment using this format: `project_key/environment_key`.

For example:

```
$ terraform import launchdarkly_environment.staging example-project/staging
```
