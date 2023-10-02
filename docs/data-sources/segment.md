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

- `unbounded` - Whether to create a standard segment (false) or a BigSegment (true).

- `unbounded_context_kind` - For Big Segments, the targeted context kind.

- `included` - List of user keys included in the segment.

- `excluded` - List of user keys excluded from the segment.

- `included_contexts` - Non-user target objects included in the segment. To learn more, read [Nested Context Target Blocks](#nested-context-target-blocks).

- `excluded_contexts` - Non-user target objects excluded from the segment. To learn more, read [Nested Context Target Blocks](#nested-context-target-blocks).

- `rules` - List of nested custom rule blocks to apply to the segment. To learn more, read [Nested Rules Blocks](#nested-rules-blocks).

- `creation_date` - The segment's creation date represented as a UNIX epoch timestamp.

### Nested Rules Blocks

Nested `rules` blocks have the following structure:

- `weight` - The integer weight of the rule (between 0 and 100000).

- `bucket_by` - The attribute by which to group users together.

- `clauses` - List of nested custom rule clause blocks. To learn more, read [Nested Clauses Blocks](#nested-clauses-blocks).

- `rollout_context_kind` - The context kind associated with the segment rule.

### Nested Clauses Blocks

Nested `clauses` blocks have the following structure:

- `attribute` - The user attribute operated on.

- `op` - The operator associated with the rule clause. This will be one of `in`, `endsWith`, `startsWith`, `matches`, `contains`, `lessThan`, `lessThanOrEqual`, `greaterThanOrEqual`, `before`, `after`, `segmentMatch`, `semVerEqual`, `semVerLessThan`, and `semVerGreaterThan`.

- `values` - The list of values associated with the rule clause.

- `value_type` - The type for each of the clause's values. Available types are `boolean`, `string`, and `number`.

- `negate` - Whether the rule clause is negated.

### Nested Context Target Blocks

Other context types can be targeted on using the `included_contexts` and `excluded_contexts` attribute blocks. These have the following structure:

- `values` - List of target object keys included in or excluded from the segment.

- `context_kind` - The context kind associated with this segment target. To view included or excluded user contexts, see the `included` and `excluded` attributes.
