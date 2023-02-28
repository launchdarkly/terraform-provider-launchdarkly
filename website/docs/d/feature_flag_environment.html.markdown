---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_feature_flag_environment"
description: |-
  Get information about LaunchDarkly environment-specific feature flag configurations.
---

# launchdarkly_feature_flag_environment

Provides a LaunchDarkly environment-specific feature flag data source.

This data source allows you to retrieve environment-specific feature flag information from your LaunchDarkly organization.

## Example Usage

```hcl
data "launchdarkly_feature_flag_environment" "example" {
  flag_id = "example-project/example-flag"
  env_key = "example-env"
}
```

## Argument Reference

- `flag_id` - (Required) The feature flag's unique `id` in the format `project_key/flag_key`.

- `env_key` - (Required) The environment key.

## Attributes Reference

In addition to the arguments above, the resource exports the following attributes:

- `on` - Whether targeting is enabled.

- `track_events` - Whether event data will be sent back to LaunchDarkly.

- `off_variation` - The index of the variation served when targeting is disabled.

- `prerequisites` - List of nested blocks describing prerequisite feature flags rules. To learn more, read [Nested Prequisites Blocks](#nested-prerequisites-blocks).

- `targets` (previously `user_targets`) - Set of nested blocks describing the individual user targets for each variation. To learn more, read [Nested Targets / Context Targets Blocks](#nested-targets-context-targets-blocks).

- `context_targets` - Set of nested blocks describing the individual targets for non-user context kinds for each variation. To learn more, read [Nested Targets / Context Targets Blocks](#nested-targets-context-targets-blocks).

- `rules` - List of logical targeting rules. To learn more, read [Nested Rules Blocks](#nested-rules-blocks).

- `fallthrough` (previously `flag_fallthrough`) - Nested block describing the default variation to serve if no `prerequisites`, `target`, or `rules` apply. To learn more, read [Nested Fallthrough Block](#nested-fallthrough-block).

### Nested Prerequisites Blocks

Nested `prerequisites` blocks have the following structure:

- `flag_key` - The prerequisite feature flag's `key`.

- `variation` - The index of the prerequisite feature flag's variation targeted.

### Nested Targets / Context Targets Blocks

Nested `targets` blocks have the following structure:

- `values` - List of `user` strings to target.

- `variation` - The index of the variation to serve is a user target value is matched.

- `context_kind` - The context kind on which the flag should target in this environment.

### Nested Fallthrough Block

The nested `fallthrough` block has the following structure:

- `variation` - The default integer variation index served if no `prerequisites`, `target`, or `rules` apply.

- `rollout_weights` - List of integer percentage rollout weights applied to each variation when no `prerequisites`, `target`, or `rules` apply.

- `bucket_by` - Group percentage rollout by a custom attribute.

- `context_kind` - The context kind associated with the specified rollout.

### Nested Rules Blocks

Nested `rules` blocks have the following structure:

- `clauses` - List of nested blocks specifying the logical clauses evaluated. To learn more, read [Nested Clauses Blocks](#nested-clauses-blocks).

- `description` - (Optional) A human-readable description of the targeting rule.

- `variation` - The integer variation index served if the rule clauses evaluate to `true`.

- `rollout_weights` - List of integer percentage rollout weights applied to each variation when the rule clauses evaluates to `true`.

- `bucket_by` - Group percentage rollout by a custom attribute.

### Nested Clauses Blocks

Nested `clauses` blocks have the following structure:

- `attribute` - The user attribute operated on.

- `op` - The operator associated with the rule clause. This will be one of `in`, `endsWith`, `startsWith`, `matches`, `contains`, `lessThan`, `lessThanOrEqual`, `greaterThanOrEqual`, `before`, `after`, `segmentMatch`, `semVerEqual`, `semVerLessThan`, and `semVerGreaterThan`.

- `values` - The list of values associated with the rule clause.

- `value_type` - The type for each of the clause's values. Available types are `boolean`, `string`, and `number`.

- `negate` - Whether the rule clause is negated.

- `context_kind` - The context kind associated with the specified rollout.

Nested `fallthrough` blocks have the following structure:

- `variation` - The integer variation index served when the rule clauses evaluate to `true`.

- `rollout_weights` - List of integer percentage rollout weights applied to each variation when the rule clauses evaluates to `true`.

- `context_kind` - (Optional) The context kind associated with the specified rollout. This argument is only valid if `rollout_weights` is also specified. If omitted, defaults to `"user"`.
