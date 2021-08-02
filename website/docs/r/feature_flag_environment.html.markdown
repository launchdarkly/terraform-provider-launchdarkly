---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_feature_flag_environment"
description: |-
  Create and manage LaunchDarkly environment-specific feature flag attributes.
---

# launchdarkly_feature_flag_environment

Provides a LaunchDarkly environment-specific feature flag resource.

This resource allows you to create and manage environment-specific feature flags attributes within your LaunchDarkly organization.

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

  user_targets {
    values = ["user0"]
  }
  user_targets {
    values = ["user1", "user2"]
  }
  user_targets {
    values = []
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

  flag_fallthrough {
    rollout_weights = [60000, 40000, 0]
  }
}
```

## Argument Reference

- `flag_id` - (Required) The feature flag's unique `id` in the format `project_key/flag_key`.

- `env_key` - (Required) The environment key.

- `targeting_enabled` - (Optional, **Deprecated**) Whether targeting is enabled. This field argument is **deprecated** in favor of `on`. Please update your config to use to `on` to maintain compatibility with future versions. Either `on` or `targeting_enabled` must be specified.

- `on` - (Optional) Whether targeting is enabled.

- `track_events` - (Optional) Whether to send event data back to LaunchDarkly.

- `off_variation` - (Optional) The index of the variation to serve if targeting is disabled.

- `prerequisites` - (Optional) List of nested blocks describing prerequisite feature flags rules. To learn more, read [Nested Prequisites Blocks](#nested-prerequisites-blocks).

- `user_targets` - (Optional) List of nested blocks describing the individual user targets for each variation. The order of the `user_targets` blocks determines the index of the variation to serve if a `user_target` is matched. To learn more, read [Nested User Target Blocks](#nested-user-targets-blocks).

- `rules` - (Optional) List of logical targeting rules. To learn more, read [Nested Rules Blocks](#nested-rules-blocks).

- `flag_fallthrough` - (Optional) Nested block describing the default variation to serve if no `prerequisites`, `user_target`, or `rules` apply. To learn more, read [Nested Flag Fallthrough Block](#nested-flag-fallthrough-block).

### Nested Prerequisites Blocks

Nested `prerequisites` blocks have the following structure:

- `flag_key` - (Required) The prerequisite feature flag's `key`.

- `variation` - (Required) The index of the prerequisite feature flag's variation to target.

### Nested User Targets Blocks

Nested `user_targets` blocks have the following structure:

- `values` - (Optional) List of `user` strings to target.

### Nested Flag Fallthrough Block

The nested `flag_fallthrough` block has the following structure:

- `variation` - (Optional) The default integer variation index to serve if no `prerequisites`, `user_target`, or `rules` apply. You must specify either `variation` or `rollout_weights`.

- `rollout_weights` - (Optional) List of integer percentage rollout weights (in thousandths of a percent) to apply to each variation if no `prerequisites`, `user_target`, or `rules` apply. The sum of the `rollout_weights` must equal 100000. You must specify either `variation` or `rollout_weights`.

- `bucket_by` - (Optional) Group percentage rollout by a custom attribute. This argument is only valid if `rollout_weights` is also specified.

### Nested Rules Blocks

Nested `rules` blocks have the following structure:

- `clauses` - (Required) List of nested blocks specifying the logical clauses to evaluate. To learn more, read [Nested Clauses Blocks](#nested-clauses-blocks).

- `variation` - (Optional) The integer variation index to serve if the rule clauses evaluate to `true`. You must specify either `variation` or `rollout_weights`.

- `rollout_weights` - (Optional) List of integer percentage rollout weights (in thousandths of a percent) to apply to each variation if the rule clauses evaluates to `true`. The sum of the `rollout_weights` must equal 100000. You must specify either `variation` or `rollout_weights`.

- `bucket_by` - (Optional) Group percentage rollout by a custom attribute. This argument is only valid if `rollout_weights` is also specified.

### Nested Clauses Blocks

Nested `clauses` blocks have the following structure:

- `attribute` - (Required) The user attribute to operate on.

- `op` - (Required) The operator associated with the rule clause. Available options are `in`, `endsWith`, `startsWith`, `matches`, `contains`, `lessThan`, `lessThanOrEqual`, `greaterThanOrEqual`, `before`, `after`, `segmentMatch`, `semVerEqual`, `semVerLessThan`, and `semVerGreaterThan`.

- `values` - (Required) The list of values associated with the rule clause.

- `value_type` - (Optional) The type for each of the clause's values. Available types are `boolean`, `string`, and `number`. If omitted, `value_type` defaults to `string`.

- `negate` - (Required) Whether to negate the rule clause.

Nested `flag_fallthrough` blocks have the following structure:

- `variation` - (Optional) The integer variation index to serve if the rule clauses evaluate to `true`. You must specify either `variation` or `rollout_weights`.

- `rollout_weights` - (Optional) List of integer percentage rollout weights (in thousandths of a percent) to apply to each variation if the rule clauses evaluates to `true`. The sum of the `rollout_weights` must equal 100000. You must specify either `variation` or `rollout_weights`.

## Attributes Reference

In addition to the arguments above, the resource exports the following attribute:

- `id` - The unique feature flag environment ID in the format `project_key/env_key/flag_key`.

## Import

LaunchDarkly feature flag environments can be imported using the resource's ID in the form `project_key/env_key/flag_key`, e.g.

```
$ terraform import launchdarkly_feature_flag_environment.example example-project/example-env/example-flag-key
```
