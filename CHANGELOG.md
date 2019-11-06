## 1.0.0 (Unreleased)

FEATURES:

- Add tags to environments. [#5](https://github.com/terraform-providers/terraform-provider-launchdarkly/issues/5)

ENHANCEMENTS:

- Add `maintainer_id` input validation.
- Improved `tags` input validaiton.
-

BUG FIXES:

- Allow flag `maintainer_id` to be unset. [#6](https://github.com/terraform-providers/terraform-provider-launchdarkly/issues/6)
- Fix typo in initialization error message. Thanks @jen20
- Flags created with invalid schema are deleted instead of left dangling.

## 0.0.1 (October 21, 2019)

NOTES:

- First release.
