## 1.1.0 (Unreleased)

FEATURES:

- Add `bucket_by` argument to `launchdarkly_feature_flag_environment` to enable custom attributes for percentage rollouts.
- Add `require_comments` and `confirm_changes` arguments to `launchdarkly_environment`.

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
