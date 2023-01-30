---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_feature_flag_environment"
description: |-
  Create and manage LaunchDarkly environment-specific feature flag attributes.
---

# launchdarkly_feature_flag_environment

Provides a LaunchDarkly environment-specific feature flag resource.

This resource allows you to create and manage environment-specific feature flags attributes within your LaunchDarkly organization.

-> **Note:** If you intend to attach a feature flag to any experiments, we do _not_ recommend configuring environment-specific flag settings using Terraform. Subsequent applies may overwrite the changes made by experiments and break your experiment. An alternate workaround is to use the [lifecycle.ignore_changes](https://developer.hashicorp.com/terraform/language/meta-arguments/lifecycle#ignore_changes) Terraform meta-argument on the `fallthrough` field to prevent potential overwrites.

## Example Usage

```hcl
resource "launchdarkly_feature_flag_environment" "number_env" {
  flag_id = launchdarkly_feature_flag.number.id
  env_key = launchdarkly_environment.staging.key

  on = true

  prerequisites {
    flag_key  = launchdarkly_feature_flag.basic.key
    variation = 0
  }

  targets {
    values    = ["user0"]
    variation = 0
  }
  targets {
    values    = ["user1", "user2"]
    variation = 1
  }

  rules {
    clauses {
      attribute = "country"
      op        = "startsWith"
      values    = ["aus", "de", "united"]
      negate    = false
    }
    clauses {
      attribute = "segmentMatch"
      op        = "segmentMatch"
      values    = [launchdarkly_segment.example.key]
      negate    = false
    }
    variation = 0
  }

  fallthrough {
    rollout_weights = [60000, 40000, 0]
  }
  off_variation = 2
}
```

## Argument Reference

- `flag_id` - (Required) The feature flag's unique `id` in the format `project_key/flag_key`. A change in this field will force the destruction of the existing resource and the creation of a new one.

- `env_key` - (Required) The environment key. A change in this field will force the destruction of the existing resource and the creation of a new one.

- `on` (previously `targeting_enabled`) - (Optional) Whether targeting is enabled. Defaults to `false` if not set.

- `track_events` - (Optional) Whether to send event data back to LaunchDarkly. Defaults to `false` if not set.

- `off_variation` - (Required) The index of the variation to serve if targeting is disabled.

- `prerequisites` - (Optional) List of nested blocks describing prerequisite feature flags rules. To learn more, read [Nested Prequisites Blocks](#nested-prerequisites-blocks).

- `targets` (previously `user_targets`) - (Optional) Set of nested blocks describing the individual user targets for each variation. To learn more, read [Nested Target Blocks](#nested-targets-blocks).

- `rules` - (Optional) List of logical targeting rules. To learn more, read [Nested Rules Blocks](#nested-rules-blocks).

- `fallthrough` (previously `flag_fallthrough`) - (Required) Nested block describing the default variation to serve if no `prerequisites`, `target`, or `rules` apply.To learn more, read [Nested Fallthrough Block](#nested-fallthrough-block).

### Nested Prerequisites Blocks

Nested `prerequisites` blocks have the following structure:

- `flag_key` - (Required) The prerequisite feature flag's `key`.

- `variation` - (Required) The index of the prerequisite feature flag's variation to target.

### Nested Targets Blocks

Nested `targets` blocks have the following structure:

- `values` - (Required) List of `user` strings to target.

- `variation` - (Required) The index of the variation to serve is a user target value is matched.

### Nested Fallthrough Block

The nested `fallthrough` (previously `flag_fallthrough`) block has the following structure:

- `variation` - (Optional) The default integer variation index to serve if no `prerequisites`, `target`, or `rules` apply. You must specify either `variation` or `rollout_weights`.

- `rollout_weights` - (Optional) List of integer percentage rollout weights (in thousandths of a percent) to apply to each variation if no `prerequisites`, `target`, or `rules` apply. The sum of the `rollout_weights` must equal 100000 and the number of rollout weights specified in the array must match the number of flag variations. You must specify either `variation` or `rollout_weights`.

- `bucket_by` - (Optional) Group percentage rollout by a custom attribute. This argument is only valid if `rollout_weights` is also specified.

### Nested Rules Blocks

Nested `rules` blocks have the following structure:

- `clauses` - (Required) List of nested blocks specifying the logical clauses to evaluate. To learn more, read [Nested Clauses Blocks](#nested-clauses-blocks).

- `variation` - (Optional) The integer variation index to serve if the rule clauses evaluate to `true`. You must specify either `variation` or `rollout_weights`.

- `rollout_weights` - (Optional) List of integer percentage rollout weights (in thousandths of a percent) to apply to each variation if the rule clauses evaluates to `true`. The sum of the `rollout_weights` must equal 100000 and the number of rollout weights specified in the array must match the number of flag variations. You must specify either `variation` or `rollout_weights`.

- `bucket_by` - (Optional) Group percentage rollout by a custom attribute. This argument is only valid if `rollout_weights` is also specified.

### Nested Clauses Blocks

Nested `clauses` blocks have the following structure:

- `attribute` - (Required) The user attribute to operate on.

- `op` - (Required) The operator associated with the rule clause. Available options are `in`, `endsWith`, `startsWith`, `matches`, `contains`, `lessThan`, `lessThanOrEqual`, `greaterThanOrEqual`, `before`, `after`, `segmentMatch`, `semVerEqual`, `semVerLessThan`, and `semVerGreaterThan`.

- `values` - (Required) The list of values associated with the rule clause.

- `value_type` - (Optional) The type for each of the clause's values. Available types are `boolean`, `string`, and `number`. If omitted, `value_type` defaults to `string`.

- `negate` - (Required) Whether to negate the rule clause.

Nested `fallthrough` blocks have the following structure:

- `variation` - (Optional) The integer variation index to serve if the rule clauses evaluate to `true`. You must specify either `variation` or `rollout_weights`.

- `rollout_weights` - (Optional) List of integer percentage rollout weights (in thousandths of a percent) to apply to each variation if the rule clauses evaluates to `true`. The sum of the `rollout_weights` must equal 100000 and the number of rollout weights specified in the array must match the number of flag variations. You must specify either `variation` or `rollout_weights`.

## Attributes Reference

In addition to the arguments above, the resource exports the following attribute:

- `id` - The unique feature flag environment ID in the format `project_key/env_key/flag_key`.

## Import

LaunchDarkly feature flag environments can be imported using the resource's ID in the form `project_key/env_key/flag_key`, e.g.

```
$ terraform import launchdarkly_feature_flag_environment.example example-project/example-env/example-flag-key
```
