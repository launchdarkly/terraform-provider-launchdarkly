## [Unreleased]

FEATURES:

- Added the `creation_date` attribute to the `launchdarkly_segment` data source and resource.

ENHANCEMENTS:

- Upgraded the Terraform plugin SDK to [v1.17.2](https://github.com/hashicorp/terraform-plugin-sdk/blob/v1-maint/CHANGELOG.md#1172-april-27-2021).

- Upgraded the LaunchDarkly API client to v5.3.0.


## [1.6.0] (July 20, 2021)

FEATURES:

- Added support for an `azure-event-hubs` data source `kind` on the `launchdarkly_destination` resource.

ENHANCEMENTS:

- Improved 429 retry handling.

- Upgraded the Go version to 1.16.

- Upgraded the LaunchDarkly API client to v5.1.0.

BUG FIXES:

- Fixed a bug in the feature flag resource where multivariate (non-boolean) resource config with zero variations would create a boolean flag.

- Fixed a bug in the feature flag resource where `default_on_variation` and `default_off_variation` would still show up in `terraform plan` following their removal.

- Updated the destination `config` `Elem` type to `TypeString` and made the `config` field required. Added improved validation to check fields for different destination kinds.

NOTES:

- The `launchdarkly_destination` resource's `enabled` argument has been deprecated in favor of `on`. Please update your config to use `on` in order to maintain compatibility with future versions.
- The `launchdarkly_webhook` resource's `policy_statements` argument has been deprecated in favor of `statements`. Please update your config to use `statements` in order to maintain compatibility with future versions.
- The `launchdarkly_webhook` data source's `policy_statements` attribute has been deprecated in favor of `statements`. Please update all references of `policy_statements` to `statements` in order to maintain compatibility with future versions.
- The `launchdarkly_webhook` resource's `enabled` argument has been deprecated in favor of `on`. Please update your config to use `on` in order to maintain compatibility with future versions.
- The `launchdarkly_webhook` data source's `enabled` attribute has been deprecated in favor of `on`. Please update your all references of `enabled` to `on` in order to maintain compatibility with future versions.

## [1.5.1] (March 16, 2021)

BUG FIXES:

- Fixed a bug preventing number and boolean values in targeting rules clauses from working. The new `value_type` attribute must be set in order to utilize number and boolean values. All values for a given targeting rule clause must be of the same type. [#51](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/51)

## [1.5.0] (September 29, 2020)

FEATURES:

- Added a `launchdarkly_project` data source.

- Added a `launchdarkly_environment` data source.

- Added a `launchdarkly_feature_flag` data source.

- Added a `launchdarkly_feature_flag_environment` data source.

- Added a `launchdarkly_segment` data source.

- Added a `launchdarkly_webhook` data source.

ENHANCEMENTS:

- Upgraded the LaunchDarkly API version to 3.5.0.

BUG FIXES:

- Resolved issues with the `launchdarkly_project`'s `environments` attribute. This attribute is no longer marked as deprecated and should be used when you wish to override the behavior of creating `Test` and `Production` environments during project creation.

- Fixed a bug where creating a `launchdarkly_feature_flag_environment` with an `off_variation` was not actually setting the off variation.

NOTES:

- The `launchdarkly_project`'s `environments` attribute is no longer marked as `computed`. This means that if you have `launchdarkly_project` resources without nested `environments` that were created before this version, you will see a diff denoting the removal of the computed environments from your state. It is safe to apply this change as no changes be made to your LaunchDarkly resources when applied.

## [1.4.1] (September 8, 2020)

FEATURES:

- Fixed a bug where omitted optional `launchdarkly_feature_flag_environment` parameters where making unwanted changes
  to the resource upon creation. [#38](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/38)

## [1.4.0] (August 21, 2020)

FEATURES:

- Added the `launchdarkly_access_token` resource.

FEATURES:

- Added the `launchdarkly_access_token` resource.

## [1.3.4] (August 3, 2020)

NOTES:

- Point `go.mod` to github.com/launchdarkly/terraform-provider-launchdarkly
- Automatically set version header at build time

## [1.3.3] (July 31, 2020)

NOTES:

- Patch release to support Terraform 0.13

## [1.3.2] (May 29, 2020)

ENHANCEMENTS:

- Change data source names from data_source* to data_source_launchdarkly*
- Add pagination for pulling team members in accordance with the latest version of the LaunchDarkly API

BUG_FIXES:

- Fix bug with setting JSON arrays as variation values.
- Fix two-step create that required making an additional API update call to set all parameters.

## [1.3.1] (May 13, 2020)

BUG_FIXES:

- Improve handling of API rate limits. [#26](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/26)

## [1.3.0] (May 05, 2020)

FEATURES:

- Added `default_on_variation` and `default_off_variation` to `launchdarkly_feature_flag`. These optional attributes can be used to set the default targeting behavior for flags in newly created environments. [#10](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/10) [#18](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/18)

BUG_FIXES:

- Improve handling of API rate limits.

## [1.2.2] (April 23, 2020)

FEATURES:

- Added `/examples`, a directory containing a variety of detailed usage examples.

BUG_FIXES:

- Fix non-empty plan after creating a `launchdarkly_team_member` with a custom role.
- Handle missing user target variations in `launchdarkly_feature_flag_environment` [#23](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/23)

## [1.2.1] (April 14, 2020)

BUG_FIXES:

- Fix import bug in `launchdarkly_project` introduced in 1.2.0 [#21](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/21)

NOTES:

- The `environments` block in `launchdarkly_project` has been deprecated in favor of the `launchdarkly_environment` resource. Please update your existing configurations to maintain future compatibility.

## [1.2.0] (April 09, 2020)

FEATURES:

- Add new resource `launchdarkly_destination`. This resource is used to manage LaunchDarkly data export destinations.
- Add `policy_statements` to `launchdarkly_webhook` and `launchdarkly_custom_role` [#16](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/16).

BUG_FIXES:

- Fixed bug preventing large number variations from being saved in the state correctly. [#14](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/14)
- Fixed bug in import validation. [#19](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/19)

NOTES:

- The `policy` block in `launchdarkly_custom_role` has been deprecated in favor of `policy_statements`. Please migrate your existing configurations to maintain future compatibility.

## [1.1.0] (March 10, 2020)

FEATURES:

- Add `bucket_by` argument to `launchdarkly_feature_flag_environment` to enable custom attributes for percentage rollouts.
- Add `require_comments` and `confirm_changes` arguments to `launchdarkly_environment`.

ENHANCEMENTS:

- Update keys.go to make keys uppercase.

BUG FIXES:

- Fix pagination bug with `launchdarkly_team_member` data source.
- Fix custom roles acceptance test race condition.

## [1.0.1] (January 13, 2020)

ENHANCEMENTS:

- Use randomized project keys in acceptance tests so they can be run in parallel.

BUG FIXES:

- Set the LaunchDarkly API version header to version `20191212`

## [1.0.0] (November 06, 2019)

FEATURES:

- Add tags attribute to `resource_launchdarkly_environment`. [#5](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/5)
- Add `maintainer_id` input validation.

ENHANCEMENTS:

- Improve `tags` input validation.

BUG FIXES:

- Allow flag `maintainer_id` to be unset. [#6](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/6)
- Fix typo in initialization error message. Thanks @jen20
- Flags created with invalid schema are deleted instead of left dangling.

## [0.0.1] (October 21, 2019)

NOTES:

- First release.
