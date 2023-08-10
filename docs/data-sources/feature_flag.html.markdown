---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_feature_flag"
description: |-
  Get information about LaunchDarkly feature flags.
---

# launchdarkly_feature_flag

Provides a LaunchDarkly feature flag data source.

This data source allows you to retrieve feature flag information from your LaunchDarkly organization.

## Example Usage

```hcl
data "launchdarkly_feature_flag" "example" {
  key         = "example-flag"
  project_key = "example-project"
}
```

## Argument Reference

- `project_key` - (Required) The feature flag's project key.

- `key` - (Required) The unique feature flag key that references the flag in your application code.

## Attributes Reference

In addition to the arguments above, the resource exports the following attributes if set:

- `id` - The unique feature flag ID in the format `project_key/flag_key`.

- `name` - The human-readable name of the feature flag.

- `variation_type` - The feature flag's variation type: `boolean`, `string`, `number` or `json`.

- `variations` - List of nested blocks describing the variations associated with the feature flag. To learn more, read [Nested Variations Blocks](#nested-variations-blocks).

- `defaults` - A map describing the index of the variation served when the flag is on for new environments. To learn more, read [Nested Defaults Blocks](#nested-defaults-blocks).

- `description` - The feature flag's description.

- `tags` - Set of feature flag tags.

- `maintainer_id` - The feature flag maintainer's 24 character alphanumeric team member ID.

- `temporary` - Whether the flag is a temporary flag.

- `archived` - Whether the flag is archived.

- `include_in_snippet` - **Deprecated** A boolean describing whether this flag has been made available to the client-side Javescript SDK using the client-side ID only. `include_in_snippet` is now deprecated. Please retrieve information from `client_side_availability.using_environment_id` to maintain future compatability.

- `client_side_availability` - A map describing whether this flag has been made available to the client-side JavaScript SDK. To learn more, read [Nested Client-Side Availability Block](#nested-client-side-availability-block).

- `custom_properties` - List of nested blocks describing the feature flag's [custom properties](https://docs.launchdarkly.com/home/connecting/custom-properties). To learn more, read [Nested Custom Properties](#nested-custom-properties).

### Nested Variations Blocks

Nested `variations` blocks have the following attributes:

- `value` - The variation value.

- `name` - The name of the variation.

- `description` - The variation's description.

### Nested Defaults Blocks

Nested `defaults` blocks have the following structure:

- `on_variation` - (Required) The index of the variation the flag will default to in all new environments when on.

- `off_variation` - (Required) The index of the variation the flag will default to in all new environments when off.

### Nested Client-Side Availability Block

The nested `client_side_availability` block has the following attributes:

- `using_environment_id` - When set to true, this flag is available to SDKs using the client-side ID.

- `using_mobile_key` - When set to true, this flag is available to SDKs using a mobile key.

### Nested Custom Properties

Nested `custom_properties` have the following attributes:

- `key` - The unique custom property key.

- `name` - The name of the custom property.

- `value` - The list of custom property value strings.
