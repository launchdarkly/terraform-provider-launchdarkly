---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_metric"
description: |-
  Get information about LaunchDarkly metrics.
---

# launchdarkly_metric

Provides a LaunchDarkly metric data source.

This data source allows you to retrieve metric information from your LaunchDarkly organization.

## Example Usage

```hcl
data "launchdarkly_metric" "example" {
  key         = "example-metric"
  project_key = "example-project"
}
```

## Argument Reference

- `key` - (Required) The metric's unique key.

- `project_key` - (Required) The metric's project key.

## Attributes Reference

In addition to the arguments above, the resource exports the following attributes:

- `id` - The unique metric ID in the format `project_key/metric_key`.

- `name` - The name of the metric.

- `project_key` - The metrics's project key.

- `kind` - The metric type. Available choices are `click`, `custom`, and `pageview`.

- `tags` - Set of tags associated with the metric.

- `description` - The description of the metric's purpose.

- `is_numeric` - Whether a `custom` metric is a numeric metric or not.

- `is_active` - Whether a metric is a active.

- `maintainer_id` - The userId of the user maintaining the metric.

- `selector` - The CSS selector for `click` metrics.

- `urls` - Which URLs the metric watches.

- `event_key` - The event key to watch for `custom` metrics.

- `success_criteria` - The success criteria for numeric `custom` metrics.

- `unit` - The unit for numeric `custom` metrics.

- `randomization_units` - A set of one or more context kinds that this metric can measure events from. Metrics can only use context kinds marked as "Available for experiments." For more information, read [Allocating experiment audiences](https://docs.launchdarkly.com/home/creating-experiments/allocation)
