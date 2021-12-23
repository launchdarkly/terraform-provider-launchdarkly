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

  defaults {
    on_variation = 2
    off_variation = 0
  }

  tags = [
    "example",
    "terraform",
    "multivariate",
    "building-materials",
  ]
}
```

```hcl
resource "launchdarkly_feature_flag" "json_example" {
  project_key = "example-project"
  key         = "json-example"
  name        = "JSON example flag"

  variation_type = "json"
  variations {
    name  = "Single foo"
    value = jsonencode({ "foo" : "bar" })
  }
  variations {
    name  = "Multiple foos"
    value = jsonencode({ "foos" : ["bar1", "bar2"] })
  }

  defaults {
    on_variation = 1
    off_variation = 0
  }
}
```

## Argument Reference

- `project_key` - (Required) The feature flag's project key.

- `key` - (Required) The unique feature flag key that references the flag in your application code.

- `name` - (Required) The human-readable name of the feature flag.

- `variation_type` - (Required) The feature flag's variation type: `boolean`, `string`, `number` or `json`.

- `variations` - (Required) List of nested blocks describing the variations associated with the feature flag. You must specify at least two variations. To learn more, read [Nested Variations Blocks](#nested-variations-blocks).

- `defaults` - (Optional) A block containing the indices of the variations to be used as the default on and off variations in all new environments. Flag configurations in existing environments will not be changed nor updated if the configuration block is removed. To learn more, read [Nested Defaults Blocks](#nested-defaults-blocks).

- `description` - (Optional) The feature flag's description.

- `tags` - (Optional) Set of feature flag tags.

- `maintainer_id` - (Optional) The feature flag maintainer's 24 character alphanumeric team member ID. If not set, it will automatically be or stay set to the member ID associated with the API key used by your LaunchDarkly Terraform provider or the most recently-set maintainer.

- `temporary` - (Optional) Specifies whether the flag is a temporary flag.

- `include_in_snippet` - **Deprecated** (Optional) Specifies whether this flag should be made available to the client-side JavaScript SDK using the client-side Id. This value gets its default from your project configuration if not set. `include_in_snippet` is now deprecated. Please migrate to `client_side_availability.using_environment_id` to maintain future compatability.

- `client_side_availability` - (Optional) A block describing whether this flag should be made available to the client-side JavaScript SDK using the client-side Id, mobile key, or both. This value gets its default from your project configuration if not set. To learn more, read [Nested Client-Side Availability Block](#nested-client-side-availability-block).

- `custom_properties` - (Optional) List of nested blocks describing the feature flag's [custom properties](https://docs.launchdarkly.com/docs/custom-properties). To learn more, read [Nested Custom Properties](#nested-custom-properties).


### Nested Variations Blocks

Nested `variations` blocks have the following structure:

- `value` - (Required) The variation value. The value's type must correspond to the `variation_type` argument. For example: `variation_type = "boolean"` accepts only `true` or `false`. The `"number"` variation type accepts both floats and ints, but please note that any trailing zeroes on floats will be trimmed (i.e. `1.1` and `1.100` will both be converted to `1.1`).

If you wish to define an empty string variation, you must still define the value field on the variations block like so:

```
variations {
  value = ""
}
```

- `name` - (Optional) The name of the variation.

- `description` - (Optional) The variation's description.

### Nested Defaults Blocks

Nested `defaults` blocks have the following structure:

- `on_variation` - (Required) The index of the variation the flag will default to in all new environments when on.

- `off_variation` - (Required) The index of the variation the flag will default to in all new environments when off.

### Nested Client-Side Availibility Block

The nested `client_side_availability` block has the following structure:

- `using_environment_id` - (Optional) Whether this flag is available to SDKs using the client-side ID.

- `using_mobile_key` - (Optional) Whether this flag is available to SDKs using a mobile key.

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
