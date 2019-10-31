---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_feature_flag"
description: |-
  Create and manage LaunchDarkly feature flags.
---

# launchdarkly_feature_flag

Provides a LaunchDarkly feature flag resource.

This resource allows you to create and manage feature flags within your LaunchDarkly organization.

## Example Usage

```hcl
resource "launchdarkly_feature_flag" "building_materials" {
  project_key = launchdarkly_project.example.key
  key         = "building-materials"
  name        = "Building materials"
  description = "this is a multivariate flag with string variations."

  variation_type = "string"
  variations {
    value       = "straw"
    name        = "Straw"
    description = "Watch out for wind."
  }
  variations {
    value       = "sticks"
    name        = "Sticks"
    description = "Sturdier than straw"
  }
  variations {
    value       = "bricks"
    name        = "Bricks"
    description = "The strongest variation"
  }

  tags = [
    "example",
    "terraform",
    "multivariate",
    "building-materials",
  ]
}
```

## Argument Reference

- `project_key` - (Required) The feature flag's project key.

- `key` - (Required) The unique feature flag key that references the flag in your application code.

- `name` - (Required) The human-readable name of the feature flag.

- `variation_type` - (Required) The feature flag's variation type: `boolean`, `string`, `number` or `json`.

- `variations` - (Required) List of nested blocks describing the variations associated with the feature flag. You must specify at least two variations. To learn more, read [Nested Variations Blocks](#nested-variations-blocks).

- `description` - (Optional) The feature flag's description.

- `tags` - (Optional) Set of feature flag tags.

- `maintainer_id` - (Optional) The feature flag maintainer's 24 character alphanumeric team member ID.

- `temporary` - (Optional) Specifies whether the flag is a temporary flag.

- `include_in_snippet` - (Optional) Specifies whether this flag should be made available to the client-side JavaScript SDK.

- `custom_properties` - (Optional) List of nested blocks describing the feature flag's [custom properties](https://docs.launchdarkly.com/docs/custom-properties). To learn more, read [Nested Custom Properties](#nested-custom-properties).

### Nested Variations Blocks

Nested `variations` blocks have the following structure:

- `value` - (Required) The variation value. The value's type must correspond to the `variation_type` argument. For example: `variation_type = "boolean"` accepts only `true` or `false`.

- `name` - (Optional) The name of the variation.

- `description` - (Optional) The variation's description.

### Nested Custom Properties

Nested `custom_properties` have the following structure:

- `key` - (Required) The unique custom property key.

- `name` - (Required) The name of the custom property.

- `value` - (Required) The list of custom property value strings.

## Attributes Reference

In addition to the arguments above, the resource exports the following attribute:

- `id` - The unique feature flag ID in the format `project_key/flag_key`.

## Import

You can import a feature flag using the feature flag's ID in the format `project_key/flag_key`.

For example:

```
$ terraform import launchdarkly_feature_flag.building_materials example-project/building-materials
```
