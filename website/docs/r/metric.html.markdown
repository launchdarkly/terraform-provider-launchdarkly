---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_metric"
description: |-
  Create and manage LaunchDarkly metrics.
---

# launchdarkly_metric

Provides a LaunchDarkly metric resource.

This resource allows you to create and manage metrics within your LaunchDarkly organization.

To learn more about metrics and experimentation, read [Experimentation Documentation](https://docs.launchdarkly.com/home/experimentation).

## Example Usage

```hcl
resource "launchdarkly_metric" "example" {
  project_key    = launchdarkly_project.example.key
  key            = "example-metric"
  name           = "Example Metric"
  description    = "Metric description."
  kind           = "pageview"
  tags           = ["example"]
  urls {
    kind = "substring"
    substring = "foo"
  }
}
```

## Argument Reference

- `key` - (Required) The unique key that references the metric.

- `project_key` - (Required) The metrics's project key.

- `name` - (Required) The human-friendly name for the metric.

- `kind` - (Required) The metric type. Available choices are `click`, `custom`, and `pageview`.

- `description` - (Optional) The description of the metric's purpose.

- `tags` - (Optional) Set of tags for the metric.

- `isNumeric` - (Optional) Whether a `custom` metric is a numeric metric or not.

- `isActive` - (Optional) Whether a metric is a active.

- `maintainerId` - (Optional) The userId of the user maintaining the metric.

- `selector` - (Required for kind `click`) The CSS selector for `click` metrics. 

- `urls` - (Required for kind `click` and `pageview`) A block determining which URLs the metric watches. To learn more, read [Nested Urls Blocks](#nested-urls-blocks).

- `event_key` - (Required for kind `custom`) The event key to watch for `custom` metrics. 

- `success_criteria` - (Required for kind `custom`) The success criteria for numeric `custom` metrics.

- `unit` - (Required for kind `custom`) The unit for numeric `custom` metrics. 

### Nested Urls Blocks

Nested `urls` blocks have the following structure:

- `kind` - (Required) The URL type. Available choices are `exact`, `canonical`, `substring` and `regex`.

- `url` - (Required for kind `exact` and `canonical`) The exact or canonical URL.

- `substring` - (Required for kind `substring`) The URL substring to match by.

- `pattern` - (Required for kind `regex`) The regex pattern to match by.

## Attributes Reference

In addition to the arguments above, the resource exports the following attribute:

- `id` - The unique environment ID in the format `project_key/metric_key`.

- `creation_date` - The metric's creation date represented as a UNIX epoch timestamp.

## Import

LaunchDarkly metrics can be imported using the metric's ID in the form `project_key/metric_key`, e.g.

```
$ terraform import launchdarkly_metric.example example-project/example-metric-key
```
