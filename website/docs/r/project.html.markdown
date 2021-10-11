---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_project"
description: |-
  Create and manage LaunchDarkly projects.
---

# launchdarkly_project

Provides a LaunchDarkly project resource.

This resource allows you to create and manage projects within your LaunchDarkly organization.

## Example Usage

```hcl
resource "launchdarkly_project" "example" {
  key  = "example-project"
  name = "Example project"

  tags = [
    "terraform",
  ]

  environments {
		key   = "production"
		name  = "Production"
		color = "EEEEEE"
		tags  = ["terraform"]
		approval_settings {
			can_review_own_request = false
			can_apply_declined_changes = false
			min_num_approvals      = 3
			required_approval_tags = ["approvals_required"]
		}
	}

  environments {
		key   = "staging"
		name  = "Staging"
		color = "000000"
		tags  = ["terraform"]
	}
}
```

## Argument Reference

- `key` - (Required) The project's unique key.

- `name` - (Required) The project's name.

- `environments` - (Required) List of nested `environments` blocks describing LaunchDarkly environments that belong to the project. When managing LaunchDarkly projects in Terraform, you should always manage your environments as nested project resources. To learn more, read [Nested Environments Blocks](#nested-environments-blocks).

-> **Note:** Mixing the use of nested `environments` blocks and [`launchdarkly_environment`](/docs/providers/launchdarkly/r/environment.html) resources is not recommended. `launchdarkly_environment` resources should only be used when the encapsulating project is not managed in Terraform.

- `include_in_snippet - (Optional) Whether feature flags created under the project should be available to client-side SDKs by default.

- `tags` - (Optional) The project's set of tags.

### Nested Environments Blocks

Nested `environments` blocks have the following structure:

- `name` - (Required) The name of the environment.

- `key` - (Required) The project-unique key for the environment.

- `color` - (Required) The color swatch as an RGB hex value with no leading `#`. For example: `000000`.

- `tags` - (Optional) Set of tags associated with the environment.

- `secure_mode` - (Optional) Set to `true` to ensure a user of the client-side SDK cannot impersonate another user. This field will default to `false` when not set.

- `default_track_events` - (Optional) Set to `true` to enable data export for every flag created in this environment after you configure this argument. This field will default to `false` when not set. To learn more, read [Data Export](https://docs.launchdarkly.com/docs/data-export).

- `default_ttl` - (Optional) The TTL for the environment. This must be between 0 and 60 minutes. The TTL setting only applies to environments using the PHP SDK. This field will default to `0` when not set. To learn more, read [TTL settings](https://docs.launchdarkly.com/docs/environments#section-ttl-settings).

- `require_comments` - (Optional) Set to `true` if this environment requires comments for flag and segment changes. This field will default to `false` when not set.

- `confirm_changes` - (Optional) Set to `true` if this environment requires confirmation for flag and segment changes. This field will default to `false` when not set.

- `approval_settings` - (Optional) A nested block describing the environment approval settings. To learn more about this feature, read [Approvals](https://docs.launchdarkly.com/home/feature-workflows/approvals). To learn more about configuring them in Terraform, read [Nested Environments Approval Settings Blocks](#nested-environments-approval-settings-blocks).

### Nested Environments Approval Settings Blocks

Nested environments `approval_settings` blocks have the following structure:

- `required` - Set to `true` for changes to flags in this environment to require approval. You may only set `required` to true if `required_approval_tags` is not set and vice versa. Defaults to `false`.

- `can_review_own_request` - Set to `true` if requesters can approve or decline their own request. They may always comment. Defaults to `false`.

- `min_num_approvals` - The number of approvals required before an approval request can be applied. This number must be between 1 and 5. Defaults to 1.

- `can_apply_declined_changes` - Set to `true` if changes can be applied as long as the `min_num_approvals` is met, regardless of whether any reviewers have declined a request. Defaults to `true`.

- `required_approval_tags` - An array of tags used to specify which flags with those tags require approval. You may only set `required_approval_tags` if `required` is not set to `true` and vice versa.

## Import

LaunchDarkly projects can be imported using the project's key, e.g.

```
$ terraform import launchdarkly_project.example example-project
```

**IMPORTANT:** Please note that, regardless of how many `environments` blocks you include on your import, _all_ of the project's environments will be saved to the Terraform state and will update with subsequent applies. This means that any environments not included in your import configuration will be torn down with any subsequent apply. If you wish to manage project properties with Terraform but not nested environments consider using Terraform's [ignore changes](https://www.terraform.io/docs/language/meta-arguments/lifecycle.html#ignore_changes) lifecycle meta-argument; see below for example.

```
resource "launchdarkly_project" "example" {
		lifecycle {
			ignore_changes = [environments]
		}
		name = "testProject"
		key = "%s"
		# environments not included on this configuration will not be affected by subsequent applies
	}
```

Managing environment resources with Terraform should always be done on the project unless the project is not also managed with Terraform.
