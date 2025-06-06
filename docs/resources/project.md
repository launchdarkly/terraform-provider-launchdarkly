---
page_title: "launchdarkly_project Resource - launchdarkly"
subcategory: ""
description: |-
  Provides a LaunchDarkly project resource.
  This resource allows you to create and manage projects within your LaunchDarkly organization.
---

# launchdarkly_project (Resource)

Provides a LaunchDarkly project resource.

This resource allows you to create and manage projects within your LaunchDarkly organization.

## Example Usage

```terraform
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
      can_review_own_request     = false
      can_apply_declined_changes = false
      min_num_approvals          = 3
      required_approval_tags     = ["approvals_required"]
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

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `environments` (Block List, Min: 1) List of nested `environments` blocks describing LaunchDarkly environments that belong to the project. When managing LaunchDarkly projects in Terraform, you should always manage your environments as nested project resources.

-> **Note:** Mixing the use of nested `environments` blocks and [`launchdarkly_environment`](/docs/providers/launchdarkly/r/environment.html) resources is not recommended. `launchdarkly_environment` resources should only be used when the encapsulating project is not managed in Terraform. (see [below for nested schema](#nestedblock--environments))
- `key` (String) The project's unique key. A change in this field will force the destruction of the existing resource and the creation of a new one.
- `name` (String) The project's name.

### Optional

- `default_client_side_availability` (Block List) A block describing which client-side SDKs can use new flags by default. (see [below for nested schema](#nestedblock--default_client_side_availability))
- `include_in_snippet` (Boolean, Deprecated) Whether feature flags created under the project should be available to client-side SDKs by default. Please migrate to `default_client_side_availability` to maintain future compatibility.
- `tags` (Set of String) Tags associated with your resource.

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedblock--environments"></a>
### Nested Schema for `environments`

Required:

- `color` (String) The color swatch as an RGB hex value with no leading `#`. For example: `000000`
- `key` (String) The project-unique key for the environment. A change in this field will force the destruction of the existing resource and the creation of a new one.
- `name` (String) The name of the environment.

Optional:

- `approval_settings` (Block List) (see [below for nested schema](#nestedblock--environments--approval_settings))
- `confirm_changes` (Boolean) Set to `true` if this environment requires confirmation for flag and segment changes. This field will default to `false` when not set.
- `critical` (Boolean) Denotes whether the environment is critical.
- `default_track_events` (Boolean) Set to `true` to enable data export for every flag created in this environment after you configure this argument. This field will default to `false` when not set. To learn more, read [Data Export](https://docs.launchdarkly.com/home/data-export).
- `default_ttl` (Number) The TTL for the environment. This must be between 0 and 60 minutes. The TTL setting only applies to environments using the PHP SDK. This field will default to `0` when not set. To learn more, read [TTL settings](https://docs.launchdarkly.com/home/organize/environments#ttl-settings).
- `require_comments` (Boolean) Set to `true` if this environment requires comments for flag and segment changes. This field will default to `false` when not set.
- `secure_mode` (Boolean) Set to `true` to ensure a user of the client-side SDK cannot impersonate another user. This field will default to `false` when not set.
- `tags` (Set of String) Tags associated with your resource.

Read-Only:

- `api_key` (String, Sensitive) The environment's SDK key.
- `client_side_id` (String, Sensitive) The environment's client-side ID.
- `mobile_key` (String, Sensitive) The environment's mobile key.

<a id="nestedblock--environments--approval_settings"></a>
### Nested Schema for `environments.approval_settings`

Optional:

- `auto_apply_approved_changes` (Boolean) Automatically apply changes that have been approved by all reviewers. This field is only applicable for approval service kinds other than `launchdarkly`.
- `can_apply_declined_changes` (Boolean) Set to `true` if changes can be applied as long as the `min_num_approvals` is met, regardless of whether any reviewers have declined a request. Defaults to `true`.
- `can_review_own_request` (Boolean) Set to `true` if requesters can approve or decline their own request. They may always comment. Defaults to `false`.
- `min_num_approvals` (Number) The number of approvals required before an approval request can be applied. This number must be between 1 and 5. Defaults to 1.
- `required` (Boolean) Set to `true` for changes to flags in this environment to require approval. You may only set `required` to true if `required_approval_tags` is not set and vice versa. Defaults to `false`.
- `required_approval_tags` (List of String) An array of tags used to specify which flags with those tags require approval. You may only set `required_approval_tags` if `required` is set to `false` and vice versa.
- `service_config` (Map of String) The configuration for the service associated with this approval. This is specific to each approval service. For a `service_kind` of `servicenow`, the following fields apply:

	 - `template` (String) The sys_id of the Standard Change Request Template in ServiceNow that LaunchDarkly will use when creating the change request.
	 - `detail_column` (String) The name of the ServiceNow Change Request column LaunchDarkly uses to populate detailed approval request information. This is most commonly "justification".
- `service_kind` (String) The kind of service associated with this approval. This determines which platform is used for requesting approval. Valid values are `servicenow`, `launchdarkly`. If you use a value other than `launchdarkly`, you must have already configured the integration in the LaunchDarkly UI or your apply will fail.



<a id="nestedblock--default_client_side_availability"></a>
### Nested Schema for `default_client_side_availability`

Required:

- `using_environment_id` (Boolean)
- `using_mobile_key` (Boolean)

## Import

Import is supported using the following syntax:

```sh
# LaunchDarkly projects can be imported using the project's key.
terraform import launchdarkly_project.example example-project
```

**IMPORTANT:** Please note that, regardless of how many `environments` blocks you include on your import, _all_ of the project's environments will be saved to the Terraform state and will update with subsequent applies. This means that any environments not included in your import configuration will be torn down with any subsequent apply. If you wish to manage project properties with Terraform but not nested environments consider using Terraform's [ignore changes](https://www.terraform.io/docs/language/meta-arguments/lifecycle.html#ignore_changes) lifecycle meta-argument; see below for example.

```terraform
resource "launchdarkly_project" "example" {
  lifecycle {
    ignore_changes = [environments]
  }
  name = "testProject"
  key = "%s"
  # environments not included on this configuration will not be affected by subsequent applies
}
```

**Note:** Following an import, the first apply may show a diff in the order of your environments as Terraform realigns its state with the order of configurations in your project configuration. This will not change your environments or their SDK keys.

**Managing environment resources with Terraform should always be done on the project unless the project is not also managed with Terraform.**
