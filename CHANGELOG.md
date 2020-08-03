## 1.4.0 (Unreleased)

## 1.3.3 (July 31, 2020)

NOTES:

- Patch release to support Terraform 0.13

## 1.3.2 (May 29, 2020)

ENHANCEMENTS:

- Change data source names from data_source* to data_source_launchdarkly*
- Add pagination for pulling team members in accordance with the latest version of the LaunchDarkly API

BUG_FIXES:

- Fix bug with setting JSON arrays as variation values.
- Fix two-step create that required making an additional API update call to set all parameters.

## 1.3.1 (May 13, 2020)

BUG_FIXES:

- Improve handling of API rate limits. [#26](https://github.com/terraform-providers/terraform-provider-launchdarkly/issues/26)

## 1.3.0 (May 05, 2020)

FEATURES:

- Added `default_on_variation` and `default_off_variation` to `launchdarkly_feature_flag`. These optional attributes can be used to set the default targeting behavior for flags in newly created environments. [#10](https://github.com/terraform-providers/terraform-provider-launchdarkly/issues/10) [#18](https://github.com/terraform-providers/terraform-provider-launchdarkly/issues/18)

BUG_FIXES:

- Improve handling of API rate limits.

## 1.2.2 (April 23, 2020)

FEATURES:

- Added `/examples`, a directory containing a variety of detailed usage examples.

BUG_FIXES:

- Fix non-empty plan after creating a `launchdarkly_team_member` with a custom role.
- Handle missing user target variations in `launchdarkly_feature_flag_environment` [#23](https://github.com/terraform-providers/terraform-provider-launchdarkly/issues/23)

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

- Update keys.go to make keys uppercase.

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
