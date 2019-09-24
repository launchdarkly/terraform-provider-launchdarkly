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

  targeting_enabled = true

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

- `targeting_enabled` - (Optional) Whether or not targeting is enabled.

- `track_events` - (Optional) Whether or not to send event data back to LaunchDarkly.

- `off_variation` - (Optional) The index of the variation to serve if targeting is disabled.

- `prerequisites` - (Optional) List of nested blocks describing prerequisite feature flags rules. The structure of this block is described below.

- `user_targets` - (Optional) List of nested blocks describing the individual user targets for each variation. The order of the `user_targets` blocks determines the index of the variation to serve if a `user_target` is matched. The structure of this block is described below.

- `rules` - (Optional) List of logical targeting rules. The structure of this block is described below.

- `flag_fallthrough` - (Optional) Nested block describing the default variation to serve if no `prerequisites`, `user_target`, or `rules` apply. The structure of this block is described below.

Nested `prerequisites` blocks have the following structure:

- `flag_key` - (Required) The prerequisite feature flag's `key`.

- `variation` - (Required) The index of the prerequisite feature flag's variation to target.

Nested `user_targets` blocks have the following structure:

- `values` - (Optional) List of `users` to target.

Nested `rules` blocks have the following structure:

- `clauses` - (Required) List of nested blocks specifying the logical clauses to evaluate.

- `variation` - (Optional) The integer variation index to serve if the rule clauses evaluate to `true`. Either `variation` or `rollout_weights` must be specified.

- `rollout_weights` - (Optional) List of integer percentage rollout weights to apply to each variation if the rule clauses evaluates to `true`. The sum of the `rollout_weights` must equal 1000000. Either `variation` or `rollout_weights` must be specified.

Nested `clauses` blocks have the following structure:

- `attribute` - (Required) The user attribute to operate on.

- `op` - (Required) The operator associated with the rule clause. Available options are `in`, `endsWith`, `startsWith`, `matches`, `contains`, `lessThan`, `lessThanOrEqual`, `greaterThanOrEqual`, `before`, `after`, `segmentMatch`, `semVerEqual`, `semVerLessThan`, and `semVerGreaterThan`.

- `values` - (Required) The list of values associated with the rule clause.

- `negate` - (Required) Whether to negate the rule clause.

Nested `flag_fallthrough` blocks have the following structure:

- `variation` - (Optional) The integer variation index to serve if the rule clauses evaluate to `true`. Either `variation` or `rollout_weights` must be specified.

- `rollout_weights` - (Optional) List of integer percentage rollout weights to apply to each variation if the rule clauses evaluates to `true`. The sum of the `rollout_weights` must equal 1000000. Either `variation` or `rollout_weights` must be specified.

## Attributes Reference

In addition to the arguments above, the following attribute is exported:

- `id` - The unique feature flag environment ID in the format `project_key/env_key/flag_key`.

## Import

LaunchDarkly feature flag environments can be imported using the segment's ID in the form `project_key/env_key/flag_key`, e.g.

```
$ terraform import launchdarkly_segment.example example-project/example-environment/example-segment-key
```
