## 1.3.0 (Unreleased)

## 1.2.1 (April 14, 2020)

BUG_FIXES:

- Fix import bug in `launchdarkly_project` introduced in 1.2.0 [#21](https://github.com/terraform-providers/terraform-provider-launchdarkly/issues/21)

NOTES:

- The `environments` block in `launchdarkly_project` has been deprecated in favor of the `launchdarkly_environment` resource. Please update your existing configurations to maintain future compatibility.

## 1.2.0 (April 09, 2020)

FEATURES:

- Add new resource `launchdarkly_destination`. This resource is used to manage LaunchDarkly data export destinations.
- Add `policy_statements` to `launchdarkly_webhook` and `launchdarkly_custom_role` [#16](https://github.com/terraform-providers/terraform-provider-launchdarkly/issues/16).

BUG_FIXES:

- Fixed bug preventing large number variations from being saved in the state correctly. [#14](https://github.com/terraform-providers/terraform-provider-launchdarkly/issues/14)
- Fixed bug in import validation. [#19](https://github.com/terraform-providers/terraform-provider-launchdarkly/issues/19)

NOTES:

- The `policy` block in `launchdarkly_custom_role` has been deprecated in favor of `policy_statements`. Please migrate your existing configurations to maintain future compatibility.

## 1.1.0 (March 10, 2020)

FEATURES:

- Add `bucket_by` argument to `launchdarkly_feature_flag_environment` to enable custom attributes for percentage rollouts.
- Add `require_comments` and `confirm_changes` arguments to `launchdarkly_environment`.

ENHANCEMENTS:

- update keys.go to make keys uppercase

BUG FIXES:

- Fix pagination bug with `launchdarkly_team_member` data source.
- Fix custom roles acceptance test race condition.

## 1.0.1 (January 13, 2020)

ENHANCEMENTS:

- Use randomized project keys in acceptance tests so they can be run in parallel.

BUG FIXES:

- Set the LaunchDarkly API version header to version `20191212`

## 1.0.0 (November 06, 2019)

FEATURES:

- Add tags attribute to `resource_launchdarkly_environment`. [#5](https://github.com/terraform-providers/terraform-provider-launchdarkly/issues/5)
- Add `maintainer_id` input validation.

ENHANCEMENTS:

- Improve `tags` input validation.

BUG FIXES:

- Allow flag `maintainer_id` to be unset. [#6](https://github.com/terraform-providers/terraform-provider-launchdarkly/issues/6)
- Fix typo in initialization error message. Thanks @jen20
- Flags created with invalid schema are deleted instead of left dangling.

## 0.0.1 (October 21, 2019)

NOTES:

- First release.
