<<<<<<< HEAD
## 1.0.0 (Unreleased)
=======
## 1.0.1 (Unreleased)
## 1.0.0 (November 06, 2019)
>>>>>>> master

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
