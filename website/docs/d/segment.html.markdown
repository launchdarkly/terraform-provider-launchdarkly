---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_segment"
description: |-
  Get information about LaunchDarkly segments.
---

# launchdarkly_segment

Provides a LaunchDarkly segment data source.

This data source allows you to retrieve segment information from your LaunchDarkly organization.

## Example Usage

```hcl
data "launchdarkly_segment" "example" {
  key         = "example-segment"
  project_key = "example-project"
  env_key     = "example-env"
}
```

## Argument Reference

- `key` - (Required) The unique key that references the segment.

- `project_key` - (Required) The segment's project key.

- `env_key` - (Required) The segment's environment key.

## Attributes Reference

In addition to the arguments above, the resource exports the following attribute:

- `id` - The unique environment ID in the format `project_key/env_key/segment_key`.

- `name` - The human-friendly name for the segment.

- `description` - The description of the segment's purpose.

- `tags` - Set of tags for the segment.

- `included` - List of users included in the segment.

- `excluded` - List of user excluded from the segment.

- `rules` - List of nested custom rule blocks to apply to the segment. To learn more, read [Nested Rules Blocks](#nested-rules-blocks).

### Nested Rules Blocks

Nested `rules` blocks have the following structure:

- `weight` - The integer weight of the rule (between 1 and 100000).

- `bucket_by` - The attribute by which to group users together.

- `clauses` - List of nested custom rule clause blocks. To learn more, read [Nested Clauses Blocks](#nested-clauses-blocks).

### Nested Clauses Blocks

Nested `clauses` blocks have the following structure:

- `attribute` - The user attribute operated on.

- `op` - The operator associated with the rule clause. This will be one of `in`, `endsWith`, `startsWith`, `matches`, `contains`, `lessThan`, `lessThanOrEqual`, `greaterThanOrEqual`, `before`, `after`, `segmentMatch`, `semVerEqual`, `semVerLessThan`, and `semVerGreaterThan`.

- `values` - The list of values associated with the rule clause.

- `negate` - Whether the rule clause is negated.


