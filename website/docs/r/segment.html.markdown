---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_segment"
description: |-
  Create and manage LaunchDarkly segments.
---

# launchdarkly_segment

Provides a LaunchDarkly segment resource.

This resource allows you to create and manage segments within your LaunchDarkly organization.

## Example Usage

```hcl
resource "launchdarkly_segment" "example" {
  key         = "example-segment-key"
  project_key = launchdarkly_project.example.key
  env_key     = launchdarkly_environment.example.key
  name        = "example segment"
  description = "This segment is managed by Terraform"
  tags        = ["segment-tag-1", "segment-tag-2"]
  included    = ["user1", "user2"]
  excluded    = ["user3", "user4"]
}
```

## Argument Reference

- `key` - (Required) The unique key that references the segment.

- `project_key` - (Required) The segment's project key.

- `env_key` - (Required) The segment's environment key.

- `name` - (Required) The human-friendly name for the segment.

- `description` - (Optional) The description of the segment's purpose.

- `tags` - (Optional) Set of tags for the segment.

- `included` - (Optional) List of users included in the segment.

- `excluded` - (Optional) List of user excluded from the segment.

- `rules` - (Optional) List of nested custom rule blocks to apply to the segment. To learn more, read [Nested Rules Blocks](#rules-blocks).

### <a id='rules-blocks'>Nested Rules Blocks</a>

Nested `rules` blocks have the following structure:

- `weight` - (Optional) The integer weight of the rule (between 1 and 100000).

- `bucket_by` - (Optional) The operator used to group users together. Available options are `in`, `endsWith`, `startsWith`, `matches`, `contains`, `lessThan`, `lessThanOrEqual`, `greaterThanOrEqual`, `before`, `after`, `segmentMatch`, `semVerEqual`, `semVerLessThan`, and `semVerGreaterThan`.

- `clauses` - (Optional) List of nested custom rule clause blocks. To learn more, read [Nested Clauses Blocks](#clauses-blocks).

### <a id='rules-blocks'>Nested Rules Blocks</a>

Nested `clauses` blocks have the following structure:

- `attribute` - (Required) The user attribute to operate on.

- `op` - (Required) The operator associated with the rule clause. Available options are `in`, `endsWith`, `startsWith`, `matches`, `contains`, `lessThan`, `lessThanOrEqual`, `greaterThanOrEqual`, `before`, `after`, `segmentMatch`, `semVerEqual`, `semVerLessThan`, and `semVerGreaterThan`.

- `values` - (Required) The list of values associated with the rule clause.

- `negate` - (Required) Whether to negate the rule clause.

## Attributes Reference

In addition to the arguments above, the provider exports the following attribute:

- `id` - The unique environment ID in the format `project_key/env_key/segment_key`.

## Import

LaunchDarkly segments can be imported using the segment's ID in the form `project_key/env_key/segment_key`, e.g.

```
$ terraform import launchdarkly_segment.example example-project/example-environment/example-segment-key
```
