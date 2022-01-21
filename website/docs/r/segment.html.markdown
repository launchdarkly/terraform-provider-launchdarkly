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

  rules {
    clauses {
      attribute = "country"
      op        = "startsWith"
      values    = ["en", "de", "un"]
      negate    = false
    }
  }
}
```

## Argument Reference

- `key` - (Required) The unique key that references the segment. A change in this field will force the destruction of the existing resource and the creation of a new one.

- `project_key` - (Required) The segment's project key. A change in this field will force the destruction of the existing resource and the creation of a new one.

- `env_key` - (Required) The segment's environment key. A change in this field will force the destruction of the existing resource and the creation of a new one.

- `name` - (Required) The human-friendly name for the segment.

- `description` - (Optional) The description of the segment's purpose.

- `tags` - (Optional) Set of tags for the segment.

- `included` - (Optional) List of user keys included in the segment.

- `excluded` - (Optional) List of user keys excluded from the segment.

- `rules` - (Optional) List of nested custom rule blocks to apply to the segment. To learn more, read [Nested Rules Blocks](#nested-rules-blocks).

### Nested Rules Blocks

Nested `rules` blocks have the following structure:

- `weight` - (Optional) The integer weight of the rule (between 1 and 100000).

- `bucket_by` - (Optional) The attribute by which to group users together.

- `clauses` - (Optional) List of nested custom rule clause blocks. To learn more, read [Nested Clauses Blocks](#nested-clauses-blocks).

### Nested Clauses Blocks

Nested `clauses` blocks have the following structure:

- `attribute` - (Required) The user attribute to operate on.

- `op` - (Required) The operator associated with the rule clause. Available options are `in`, `endsWith`, `startsWith`, `matches`, `contains`, `lessThan`, `lessThanOrEqual`, `greaterThanOrEqual`, `before`, `after`, `segmentMatch`, `semVerEqual`, `semVerLessThan`, and `semVerGreaterThan`.

- `values` - (Required) The list of values associated with the rule clause.

- `value_type` - (Optional) The type for each of the clause's values. Available types are `boolean`, `string`, and `number`. If omitted, `value_type` defaults to `string`.

- `negate` - (Required) Whether to negate the rule clause.

## Attributes Reference

In addition to the arguments above, the resource exports the following attribute:

- `id` - The unique environment ID in the format `project_key/env_key/segment_key`.

- `creation_date` - The segment's creation date represented as a UNIX epoch timestamp.

## Import

LaunchDarkly segments can be imported using the segment's ID in the form `project_key/env_key/segment_key`, e.g.

```
$ terraform import launchdarkly_segment.example example-project/example-environment/example-segment-key
```
