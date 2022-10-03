## [2.9.3] (October 3, 2022)

BUG FIXES:

- Correctly set bucketBy to nil when explicitly set to an empty string to avoid API errors [#120](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/120)
- Print error message from API response for Teams resource

NOTES:

- Add `ignore_changes` guide

## [2.9.2] (September 1, 2022)

BUG FIXES:

- Fixes a bug in the `launchdarkly_feature_flag` resource where explicit defaults were not getting set for boolean flags upon creation.

## [2.9.1] (August 24, 2022)

BUG FIXES:

- Fixes a bug in the `launchdarkly_feature_flag_environment` that prevented users from updating targeting rule clauses when the targeting rule was being used as the fallthrough variation with a percentage rollout.

- Fixes a bug in the `launchdarkly_feature_flag_environment` that resulted in the default `string` rule clause value type not being respected. [#102](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/102)

## [2.9.0] (August 05, 2022)

FEATURES:

- Add `no_access` role as a valid value for the `role` key for the `launchdarkly_team_member` resource.

## [2.8.0] (August 04, 2022)

FEATURES:

- Added `launchdarkly_team` data source and provider.

NOTES:

- Updated LaunchDarkly go api client from v7 to v10.

## [2.7.2] (July 28, 2022)

BUG FIXES:

- Remove invalid configurations from the `launchdarkly_audit_log_subscription` resource.

## [2.7.1] (July 27, 2022)

BUG FIXES:

- The `launchdarkly_feature_flag_environment` data source now checks whether the environment exists and prints out a more descriptive error. [#101](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/101)

NOTES:

- Upgrade Go version to 1.18

## [2.7.0] (May 5, 2022)

FEATURES:

- Added the `base_permissions` field to the `launchdarkly_custom_role` resource.

## [2.6.1] (April 12, 2022)

NOTES:

- Removed callout to `bypassRequiredApproval` action in documentation pending further development.
- Fix formatting in some documentation

## [2.6.0] (April 7, 2022)

ENHANCEMENTS:

- Added the `hide_member_details` argument to the Datadog `config` for the `launchdarkly_audit_log_subscription` resource. When `hide_member_details` is `true`, LaunchDarkly member information will be redacted before events are sent to Datadog.

NOTES:

- Added a callout to the `bypassRequiredApproval` action in documentation.

## [2.5.0] (February 7, 2022)

FEATURES:

- Added Slack webhooks to the `launchdarkly_audit_log_subscription` resource and data source. [#16](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/16)
- Added more Datadog host URLs to the Datadog `launchdarkly_audit_log_subscription` resource.

BUG FIXES:

- Fixed an issue where the `config` was not being set on the `launchdarkly_audit_log_subscription` data source.

## [2.4.1] (January 21, 2022)

BUG FIXES:

- Fixed a [bug](https://app.shortcut.com/launchdarkly/story/138913/terraform-provider-panics-when-trying-to-create-triggers-that-are-enabled) preventing `launchdarkly_flag_trigger`s from being created in the `enabled` state.

- Fixed a [bug](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/79) introduced in v2.2.0 where `launchdarkly_segments` with `rule` blocks not containing a `weight` were defaulting to a `weight` of 0.

## [2.4.0] (January 19, 2022)

FEATURES:

- Added a `launchdarkly_team_members` data source to allow using multiple team members in one data source.

- Added a new `launchdarkly_metric` resource and data source for managing LaunchDarkly experiment flag metrics.

- Added a new `launchdarkly_flag_triggers` resource and data source for managing LaunchDarkly flag triggers.

- Added a new `launchdarkly_relay_proxy_configuration` resource and data source for managing configurations for the Relay Proxy's [automatic configuration](https://docs.launchdarkly.com/home/relay-proxy/automatic-configuration#writing-an-inline-policy) feature.

- Added a new `launchdarkly_audit_log_subscription` resource and data source for managing LaunchDarkly audit log integration subscriptions.

ENHANCEMENTS:

- Updated tests to use the constant attribute keys defined in launchdarkly/keys.go.

- Added a pre-commit file with a hook to alphabetize launchdarkly/keys.go

- Improved 409 and 429 retry handling.

## [2.3.0] (January 4, 2022)

FEATURES:

- Added `default_client_side_availability` block to the `launchdarkly_project` resource to specify whether feature flags created under the project should be available to client-side SDKs by default.

BUG FIXES:

- Fixed a bug in the `launchdarkly_project` and `launchdarkly_environment` resources which caused Terraform to crash when environment approvals settings are omitted from the LaunchDarkly API response.

NOTES:

- The `launchdarkly_project` resource's argument `include_in_snippet` has been deprecated in favor of `default_client_side_availability`. Please update your config to use `default_client_side_availability` in order to maintain compatibility with future versions.

- The `launchdarkly_project` data source's attribute `client_side_availability` has been renamed to `default_client_side_availability`. Please update your config to use `default_client_side_availability` in order to maintain compatibility with future versions.

## [2.2.0] (December 23, 2021)

ENHANCEMENTS:

- Upgraded the LaunchDarkly API client to version 7.
- Flag resource creation respects project level SDK availability defaults.

FEATURES:

- Added `client_side_availability` block to the `launchdarkly_feature_flag` resource to allow setting whether this flag should be made available to the client-side JavaScript SDK using the client-side ID, mobile key, or both.

NOTES:

- The `launchdarkly_feature_flag` resource's argument `include_in_snippet` has been deprecated in favor of `client_side_availability`. Please update your config to use `client_side_availability` in order to maintain compatibility with future versions.

ENHANCEMENTS:

- Upgraded the LaunchDarkly API client to version 7.

## [2.1.1] (October 11, 2021)

BUG FIXES:

- Fixed an oversight in the approval settings where the environment `approval_settings` property `can_apply_declined_changes` was defaulting to `false` where it should have been defaulting to `true` in alignment with the LaunchDarkly API.

- Updated an error message.

## [2.1.0] (October 8, 2021)

FEATURES:

- Added `approval_settings` blocks to the `launchdarkly_environment` resource and nested `environments` blocks on the `launchdarkly_project` resource.

- Added a boolean `archive` attribute on the `launchdarkly_feature_flag` resource to allow archiving and unarchiving flags instead of deleting them.

## [2.0.1] (September 20, 2021)

BUG FIXES:

- Fixed [a bug](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/67) resulting in nested environments not being imported on the `launchdarkly_project` resource. As a result, _all_ of a project's environments will be saved to the Terraform state during an import of the `launchdarkly_project` resource. Please keep in mind if you have not added all of the existing environments to your Terraform config before importing a `launchdarkly_project` resource, Terraform will delete these environments from LaunchDarkly during the next `terraform apply`. If you wish to manage project properties with Terraform but not nested environments consider using Terraform's [ignore changes](https://www.terraform.io/docs/language/meta-arguments/lifecycle.html#ignore_changes) lifecycle meta-argument.

## [2.0.0] (August 31, 2021)

ENHANCEMENTS:

- Improved test coverage.

NOTES:

- As part of the ongoing deprecation of Terraform 0.11, the LaunchDarkly provider now only supports Terraform 0.12 and higher.

- This release changes the way LaunchDarkly recommends you manage `launchdarkly_environment` and `launchdarkly_project` resources in tandem. It is recommended you do not manage environments as separate resources _unless_ you wish to manage the encapsulating project externally (not via Terraform). As such, at least one `environments` attribute will now be `Required` on the `launchdarkly_project` resource, but you will also be able to manage environments outside of Terraform on Terraform-managed projects if you do not import them into the Terraform state as a configuration block on the encapsulating project resource.

- The deprecated `launchdarkly_destination` resource `enabled` field has been removed in favor of `on`. `on` now defaults to `false` when not explicitly set.

- The `default_on_variation` and `default_off_variation` properties on the `launchdarkly_feature_flag` resource have now been replaced with a computed `defaults` block containing the properties `on_variation` and `off_variation` that refer to the variations in question by index rather than value.

- The `launchdarkly_feature_flag_environment` resource and data source `target` attribute schema has been modified to include a new `variation` attribute. Here `variation` represents the index of the feature flag variation to serve if a user target is matched.

- The deprecated `launchdarkly_feature_flag_environment` resource `targeting_enabled` field has been removed in favor of `on`. `on` now defaults to `false` when not explicitly set.

- The deprecated `launchdarkly_feature_flag_environment` resource `user_targets` field has been removed in favor of `targets`. `targets` now defaults to null when not explicitly set.

- The deprecated `launchdarkly_feature_flag_environment` resource `flag_fallthrough` field has been removed in favor of `fallthrough`.

- The deprecated `launchdarkly_webhooks` resource `enabled` field has been removed in favor of `on`. `on` is now a required field.

- The deprecated `launchdarkly_webhooks` resource `policy_statements` field has been removed in favor of `statements`.

- `off_variation` and `fallthrough` (previously `flag_fallthrough`) on `launchdarkly_feature_flag_environment` are now `Required` fields.

- Most optional fields will now be removed or revert to their null / false value when not explicitly set and / or when removed, including:

  - `on` on the `launchdarkly_destination` resource

  - `include_in_snippet` on the `launchdarkly_project` resource

  - on the `launchdarkly_environment` resource and in `environment` blocks on the `launchdarkly_project` resource:

    - `secure_mode`

    - `default_track_events`

    - `require_comments`

    - `confirm_changes`

    - `default_ttl` (reverts to `0`)

  - on the `launchdarkly_feature_flag_environment` resource:

    - `on` (previously `targeting_enabled`, reverts to `false`)

    - `rules`

    - `targets` (previously `user_targets`)

    - `prerequisites`

    - `track_events` (reverts to `false`)

BUG FIXES:

- Fixed a bug in the `launchdarkly_webhook` resource where `statements` removed from the configuration were not being deleted in LaunchDarkly.

- The `launchdarkly_feature_flag` resource `maintainer_id` field is now computed and will update the state with the most recently-set value when not explicitly set.

- The `client_side_availability` attribute on the `launchdarkly_feature_flag` and `launchdarkly_project` data sources has been corrected to an array with a single map item. This means that you will need to add an index 0 when accessing this property from the state (ex. `client_side_availability.using_environment_id` will now have to be accessed as `client_side_availability.0.using_environment_id`).

## [1.7.1] (August 24, 2021)

ENHANCEMENTS:

- Improved test coverage.

NOTES:

- As part of the ongoing deprecation of Terraform 0.11, the LaunchDarkly provider now only supports Terraform 0.12 and higher.

- This release changes the way LaunchDarkly recommends you manage `launchdarkly_environment` and `launchdarkly_project` resources in tandem. It is recommended you do not manage environments as separate resources _unless_ you wish to manage the encapsulating project externally (not via Terraform). As such, at least one `environments` attribute will now be `Required` on the `launchdarkly_project` resource, but you will also be able to manage environments outside of Terraform on Terraform-managed projects if you do not import them into the Terraform state as a configuration block on the encapsulating project resource.

- The deprecated `launchdarkly_destination` resource `enabled` field has been removed in favor of `on`. `on` now defaults to `false` when not explicitly set.

- The `default_on_variation` and `default_off_variation` properties on the `launchdarkly_feature_flag` resource have now been replaced with a computed `defaults` block containing the properties `on_variation` and `off_variation` that refer to the variations in question by index rather than value.

- The `launchdarkly_feature_flag_environment` resource and data source `target` attribute schema has been modified to include a new `variation` attribute. Here `variation` represents the index of the feature flag variation to serve if a user target is matched.

- The deprecated `launchdarkly_feature_flag_environment` resource `targeting_enabled` field has been removed in favor of `on`. `on` now defaults to `false` when not explicitly set.

- The deprecated `launchdarkly_feature_flag_environment` resource `user_targets` field has been removed in favor of `targets`. `targets` now defaults to null when not explicitly set.

- The deprecated `launchdarkly_feature_flag_environment` resource `flag_fallthrough` field has been removed in favor of `fallthrough`.

- The deprecated `launchdarkly_webhooks` resource `enabled` field has been removed in favor of `on`. `on` is now a required field.

- The deprecated `launchdarkly_webhooks` resource `policy_statements` field has been removed in favor of `statements`.

- `off_variation` and `fallthrough` (previously `flag_fallthrough`) on `launchdarkly_feature_flag_environment` are now `Required` fields.

- Most optional fields will now be removed or revert to their null / false value when not explicitly set and / or when removed, including:

  - `on` on the `launchdarkly_destination` resource

  - `include_in_snippet` on the `launchdarkly_project` resource

  - on the `launchdarkly_environment` resource and in `environment` blocks on the `launchdarkly_project` resource:

    - `secure_mode`

    - `default_track_events`

    - `require_comments`

    - `confirm_changes`

    - `default_ttl` (reverts to `0`)

  - on the `launchdarkly_feature_flag_environment` resource:

    - `on` (previously `targeting_enabled`, reverts to `false`)

    - `rules`

    - `targets` (previously `user_targets`)

    - `prerequisites`

    - `track_events` (reverts to `false`)

BUG FIXES:

- Fixed a bug in the `launchdarkly_webhook` resource where `statements` removed from the configuration were not being deleted in LaunchDarkly.

- The `launchdarkly_feature_flag` resource `maintainer_id` field is now computed and will update the state with the most recently-set value when not explicitly set.

- The `client_side_availability` attribute on the `launchdarkly_feature_flag` and `launchdarkly_project` data sources has been corrected to an array with a single map item. This means that you will need to add an index 0 when accessing this property from the state (ex. `client_side_availability.using_environment_id` will now have to be accessed as `client_side_availability.0.using_environment_id`).

## [Unreleased]

BUG FIXES:

- Fixes [a bug](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/60) where attempts to create `launchdarkly_feature_flag` variations with an empty string value were throwing a panic.

NOTES:

- The `launchdarkly_feature_flag_environment` resource and data source's `flag_fallthrough` argument has been deprecated in favor of `fallthrough`. Please update your config to use `fallthrough` in order to maintain compatibility with future versions.

- The `launchdarkly_feature_flag_environment` resource and data source's `user_targets` argument has been deprecated in favor of `targets`. Please update your config to use `targets` in order to maintain compatibility with future versions.

## [1.7.0] (August 2, 2021)

FEATURES:

- Added the `creation_date` attribute to the `launchdarkly_segment` data source and resource.

ENHANCEMENTS:

- Upgraded the Terraform plugin SDK to [v1.17.2](https://github.com/hashicorp/terraform-plugin-sdk/blob/v1-maint/CHANGELOG.md#1172-april-27-2021).

- Upgraded the LaunchDarkly API client to v5.3.0.

- Added example team member resource configs in `examples/team_member`.

BUG FIXES:

- Updated the `project_key` attribute on the environment resource to be `Required` in keeping with the API.

- Added validation for `launchdarkly_access_token` resource creation and updates.

- Fixed a bug in the team member resource where changing the email in the configuration would result in no real changes. Changing the email will now force a replacement.

NOTES:

- The `launchdarkly_feature_flag_environment` resource's `targeting_enabled` argument has been deprecated in favor of `on`. Please update your config to use `on` in order to maintain compatibility with future versions.

- The `launchdarkly_access_token` resource's `policy_statements` argument has been deprecated in favor of `inline_roles`. Please update your config to use `inline_roles` in order to maintain compatibility with future versions.

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

- Add tags attribute to `launchdarkly_environment` resource. [#5](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/5)
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
