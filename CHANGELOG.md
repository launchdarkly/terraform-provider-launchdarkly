# Change log

All notable changes to the LaunchDarkly Terraform Provider will be documented in this file. This project adheres to [Semantic Versioning](http://semver.org).

## [2.23.0](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v2.22.0...v2.23.0) (2025-02-11)


### Features

* add context_kind to targeting rules with percentage rollouts ([#293](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/293)) ([a41f969](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/a41f96963b8cf66b3045e8571e391630895e3b47))

## [2.22.0](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v2.21.5...v2.22.0) (2025-02-06)


### Features

* add role attributes to `launchdarkly_team_member` ([#289](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/289)) ([bc24609](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/bc2460932a50bb81c39dd8ebe11024644e4a3e58))
* add role attributes to custom roles ([#286](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/286)) ([5160b78](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/5160b7885ddeb98421efa93625076275a233e33e))
* add role_attributes to `launchdarkly_team` ([#290](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/290)) ([10ac131](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/10ac13184d8f04a02c98d1f69a1629760e16ed19))


### Bug Fixes

* make deprecated metric is_active field optional and computed ([#285](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/285)) ([afcbdc3](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/afcbdc359552f588a91a609996a313f57db6f2dd))

## [2.21.5](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v2.21.4...v2.21.5) (2025-01-30)


### Bug Fixes

* Bump golang.org/x/crypto from 0.24.0 to 0.31.0 ([#254](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/254)) ([eaea627](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/eaea6272d3ffa93eddbd5c8c1ac34ba94a00e85c))
* Bump golang.org/x/net from 0.26.0 to 0.33.0 ([#267](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/267)) ([505712e](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/505712e0bedeb2f6e5a8171403d83f130358d785))

## [2.21.4](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v2.21.3...v2.21.4) (2025-01-30)


### Bug Fixes

* add random characters to name that keeps conflicting ([#272](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/272)) ([4d5cd7a](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/4d5cd7a637ec862a6926a30097a9da470cbacfa2))
* update LD API version to 20240415 ([#268](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/268)) ([70bef86](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/70bef8676551150ba3ead5286ea6823d2cff0563))


## [2.21.3](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v2.21.2...v2.21.3) (2024-12-17)


### Bug Fixes

* Add missing changelog entry ([#236](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/236)) ([171d3d6](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/171d3d60b42560a23e3033a324efc07de0016046))
* add test for segments with anonymous clauses ([#249](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/249)) ([0274906](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/0274906a3a05dd08f1152517b324ef22e1ec6b7b))

## [2.21.2] - 2024-12-06

### Fixed:
- Fixed and issue with our release process. This release does not contain any code changes to the provider.

## [2.21.1] - 2024-12-06
### Updated:
- Added a note to global `launchdarkly_feature_flag` documentation about managing variations for env-level flag configurations outside of Terraform.

## [2.21.0] - 2024-10-24
### Added:
- Added support for managing the [Kosli integration](https://docs.launchdarkly.com/integrations/kosli/) with the `launchdarkly_audit_log_subscription` resource and data source.
- Add missing fields `analysis_type`, `include_units_without_events`, `percentile_value`, and `unit_aggregation_type` to the `launchdarkly_metric` resource and data provider.
- Added the computed `version` attribute to `launchdarkly_metric`.

### Deprecated:
- Deprecated the `launchdarkly_metric`'s `is_active` attribute. This attribute is no longer used by LaunchDarkly and is safe to remove from Terraform configs.

## [2.20.2] - 2024-09-04
### Fixed:
- Fixed a bug in the `launchdarkly_team` resource that prevented changes to the `key` from requiring the resource to be destroyed and recreated.
- Bump google.golang.org/grpc from 1.64.0 to 1.64.1 to address [CVE-2023-45288](https://nvd.nist.gov/vuln/detail/CVE-2023-45288).

## [2.20.1] - 2024-09-02
### Fixed:
- Fixed a bug that prevented `api_host` from being respected when using the `launchdarkly_team_role_mapping` resource.
- Fixed a bug in the `launchdarkly_team_role_mapping` resource that prevented creating the resource with an empty `custom_role_keys` array.

## [2.20.0] - 2024-07-22
### Added:
- Updated `launchdarkly_audit_log_subscription` resource and data source to support the [Last9 integration](https://docs.launchdarkly.com/integrations/last9/).

## [2.19.0] - 2024-06-14
### Added:
- Added support for the `dynatrace-cloud-automation` `integration_key` on the `launchdarkly_flag_trigger` resource. 

### Updated:
- Documentation for all resources and data sources is now generated from the underlying schema using [tf-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs). 
- Updated various dependencies.

## [2.18.4] - 2024-05-28
### Fixed:
- Do not retry 409s from LaunchDarkly.

### Updated:
- Update LaunchDarkly API client dependency to v16.

## [2.18.3] - 2024-05-16
### Added:
- `critical` field on `launchdarkly_environment`.

### Changed:
- Updated examples to show use of all clause property

## [2.18.2] - 2024-03-25
### Improvements:
- Updated the resource name in our `launchdarkly_feature_flag_environment` docs example to make it more clear that it's a feature flag environment, not an environment.
- Updated the LD API client from v14 to v15.

## [2.18.1] - 2024-03-14
### Fixed:
- Fixed the "Default off variation must be a valid index in the variations list" error for cases where the default variations were defined when creating a `launchdarkly_feature_flag` but no variations were explicitly defined (in the case of a default boolean flag, for example). 
- Adds a default value "HigherThanBaseline" to the `launchdarkly_metric.success_criteria` field to correspond to the same change in the API.

## [2.18.0] - 2024-02-23
### Fixed:
- Fixed a bug surfaced by [issue #198](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/198) where feature flag configuration off variations were being reverted to an incorrect default value when a `launchdarkly_feature_flag_environment` was removed.

### Added:
- Added previously missing support for team maintainers on the `launchdarkly_feature_flag` resource and data source with the new `team_maintainer_key` attribute field.

### Improvements:
- Updated the LD API client to v0.14.

## [2.17.0] - 2023-12-08
### Added:
- Added the `service_kind` and `service_config` attributes the `launchdarkly_environment`'s approval settings. With these settings you can configure the ServiceNow approval system. Thanks, @arhill05 [#191](https://github.com/launchdarkly/terraform-provider-launchdarkly/pull/191)

## [2.16.0] - 2023-10-17
### Added:
- Added additional fields related to big segments to the `launchdarkly_segment` resource and data source. Thanks, @christogav! [#187](https://github.com/launchdarkly/terraform-provider-launchdarkly/pull/187)

### Improvements:
- Generated documentation for the `launchdarkly_segment` data source and resource using [terraform-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs).

## [2.15.2] - 2023-09-26
### Fixed:
- Added 404 retries to the `launchdarkly_team_role_mapping` resource. [#179](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/179)
- Fixed a bug related to how tags are updated in the `launchdarkly_environment` resource.

## [2.15.1] - 2023-08-30
### Fixed:
- Generate `launchdarkly_feature_flag` documentation from schema.
- Fix typo in webhook example code. Thanks, @lachlancooper [#180](https://github.com/launchdarkly/terraform-provider-launchdarkly/pull/180)

## [2.15.0] - 2023-08-11
### Added:
- Adds the new optional `http_timeout` attribute to the provider config. This attribute allows you to configure the HTTP request timeout when the provider makes API calls to LaunchDarkly. [#174](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/174)

### Fixed:
- Minor docs formatting errors.

## [2.14.0] - 2023-08-10
### Features:
- Adds the new optional `http_timeout` attribute to the provider config. This attribute allows you to configure the HTTP request timeout when the provider makes API calls to LaunchDarkly. [#174](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/174)
- Updates some of the public documentation using [terraform-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs).

### Fixed:
- Increased the `launchdarkly_project` key length validation from 20 characters to 100 characters. [#175](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/175)

## [2.13.4] - 2023-08-03
### Fixed:
- Fix YAML frontmatter on `launchdarkly_relay_proxy_configuration` documentation. Thanks, @felixlut [#171](https://github.com/launchdarkly/terraform-provider-launchdarkly/pull/171)

## [2.13.3] - 2023-07-26
### Fixed:
- Fixed a bug in the `launchdarkly_project` resource which prevented importing and managing a project with more than 20 environments. [#154](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/154)

## [2.13.2] - 2023-07-19
### Changed:
- Improved the `launchdarkly_team_role_mapping` documentation to include a reference to the [team sync with SCIM feature](https://docs.launchdarkly.com/home/account-security/sso/scim#team-sync-with-scim). [#152](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/152)

## [2.13.1] - 2023-07-18
### Fixed:
- Fixed an incorrect header in the `launchdarkly_team_role_mapping` resource documentation. [#152](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/152)

## [2.13.0] - 2023-07-18
FEATURES:

- Adds the new `launchdarkly_team_role_mapping` resource to manage the custom roles associated with a LaunchDarkly team. This is useful if the LaunchDarkly team is created and managed externally, such as via [Okta SCIM](https://docs.launchdarkly.com/home/account-security/okta/#using-okta-to-manage-launchdarkly-teams-with-scim). If you wish to create an manage the team using Terraform, we recommend using the [`launchdarkly_team` resource](https://registry.terraform.io/providers/launchdarkly/launchdarkly/latest/docs/resources/team) instead. [#152](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/152)

## [2.12.2] - 2023-06-28

BUG FIXES:

- Fixes [an issue](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/142) in the `launchdarkly_feature_flag_environment` resource where empty `fallthrough` blocks would cause the provider to panic.
- Fixes the release pipeline

## [2.12.1] - 2023-06-21

BUG FIXES:

- Fixes [an issue](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/142) in the `launchdarkly_feature_flag_environment` resource where empty `fallthrough` blocks would cause the provider to panic.

## [2.12.0] - 2023-03-10

FEATURES:

- Adds the optional `randomization_units` attribute to the `launchdarkly_metric` resource and data source. For more information, read [Allocating experiment audiences](https://docs.launchdarkly.com/home/creating-experiments/allocation). Thanks, @goldfrapp04!

- Bumps golang.org/x/net dependency from 0.1.0 to 0.7.0.

- Bumps Go version to 1.19.

NOTES:

- Fixes incorrect attribute casing in `launchdarkly_metric` resource and data source documentation.

## [2.11.0] - 2023-02-28

FEATURES:

- Adds a new `context_targets` attribute to `launchdarkly_feature_flag_environment` resource blocks that allow you to target based on context kinds other than `"user"`.

- Adds a new `context_kind` field to `launchdarkly_feature_flag_environment` fallthrough blocks.

- Adds a new `context_kind` field to `launchdarkly_feature_flag_environment` rule clauses.

- Adds `included_contexts` and `excluded_contexts` attribute to `launchdarkly_segment` resource blocks that allow you to target based on context kinds other than `"user"`.

- Adds a new `rollout_context_kind` field to `launchdarkly_segment` rule blocks.

- Adds `user_identities` field to the `launchdarkly_destination` mParticle configuration attribute that allow you to define mParticle user identities in tandem with their corresponding LaunchDarkly context kind.

## [2.10.0] - 2023-02-27

FEATURES:

- Adds the optional `description` argument to the nested `rules` blocks on the `launchdarkly_feature_flag_environment` resource and data source.

BUG FIXES:

- Fixes an issue on the `launchdarkly_feature_flag` resource affecting some customers where the `client_side_availability` property would sometimes unexpectedly update. Also updates the behavior of that field to not default back to project defaults even if removed, in keeping with [the behavior of the LaunchDarkly API](https://docs.launchdarkly.com/home/organize/projects/?q=project#project-flag-defaults). If a feature flag resource is created for the first time without `client_side_availability` set, it will be set to the project defaults.

## [2.9.5] - 2023-01-30

BUG FIXES:

- Fixes a bug that allowed target blocks to be defined with no values in Terraform, resulting in a plan differential post-apply. A minimum of 1 item has been applied to the `values` field of `launchdarkly_feature_flag_environment` resource blocks.

- Fixes a bug where removal of `tags` on `resource_launchdarkly_segment` was not resulting in the actual deletion of tags.

NOTES:

- Adds a note to the `launchdarkly_feature_flag_environment` documentation to recommend against usage with experimentation.
- Updates links to LaunchDarkly product and REST API documentation.

## [2.9.4] (October 26, 2022)

BUG FIXES:

- Adds `ignore_changes` guide to the documentation sidebar
- Fixes broken link in `account_members` resource documentation

## [2.9.3] (October 3, 2022)

BUG FIXES:

- Correctly set bucketBy to nil when explicitly set to an empty string to avoid API errors [#120](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/120)
- Print error message from API response for `launchdarkly_team` resource

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
