# Change log

All notable changes to the LaunchDarkly Terraform Provider will be documented in this file. This project adheres to [Semantic Versioning](http://semver.org).

## [3.0.0-beta.8](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v3.0.0-beta.7...v3.0.0-beta.8) (2026-07-07)


### ⚠ BREAKING CHANGES

* adopt api-client-go v23 typed clients and field renames ([#494](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/494))
* v3 RC prep — finish single-object and map conversions (REL-13579) ([#486](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/486))
* `launchdarkly_project.environments` is now a map keyed by environment key (`environments = { "production" = { ... } }`) instead of an ordered list. The inner `key` attribute is retained (Optional+Computed) and must equal the map key. The map is authoritative: an environment removed from it is deleted on apply; use `lifecycle { ignore_changes = [environments] }` to manage environments outside Terraform. Rewrite configurations with `migrate-tf-syntax` (it converts the blocks and warns on positional `environments[N]` references with the exact replacement — it does not rewrite them). The v2 -> v3 state upgrade is automatic; `3.0.0-beta.1`–`beta.3` list-shaped state must be re-imported.
* client_side_availability, defaults, default_client_side_availability, and fallthrough now use object syntax (`attr = {...}`) instead of a single-element list (`attr = [{...}]`).
* **access_token:** remove deprecated expire and policy_statements ([#442](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/442))
* **custom_role:** remove deprecated policy attribute ([#441](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/441))
* **project:** remove deprecated include_in_snippet and DS client_side_availability ([#440](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/440))
* **feature_flag:** remove deprecated include_in_snippet attribute ([#439](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/439))
* **metric:** remove deprecated is_active attribute ([#438](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/438))
* all user HCL using block syntax for approval_settings, variations, rules, targets, context_targets, prerequisites, fallthrough, client_side_availability, custom_properties, defaults, environments, policy, policy_statements, inline_roles, statements, role_attributes, included_contexts, excluded_contexts, urls, instructions, boolean_defaults, messages, segments, linked_segments must be rewritten to attribute syntax in v3.0.0.

### Features

* [bot] Regenerate integration configs ([#391](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/391)) ([b772383](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/b7723839b7c4a45f5010c9990c755e28f1272509))
* [REL-12555] Release Views Resources from preview provider into main ([#400](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/400)) ([b718a8c](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/b718a8c489cff57fa5e75fd3b297d42cc69d7d8e))
* [REL-12731] - add support for flag templates ([#403](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/403)) ([927d50b](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/927d50b7113dc21d3a2a4cc48ef686be2d7b49c5))
* [REL-13052] add IP allowlist config and entry resources ([#411](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/411)) ([03a540b](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/03a540beb0dc302ad6d424ddf8ddbabd27cc78ae))
* **access_token:** remove deprecated expire and policy_statements ([#442](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/442)) ([68cb932](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/68cb932441186502dee8a2f467678642c3183155))
* add ai configs resources ([#404](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/404)) ([874bdec](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/874bdecc599212808089b9d5a0d6cced593d20c3))
* add API-coverage drift report (autogen pipeline stage 1) ([#445](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/445)) ([13c2b21](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/13c2b21ab49aa6a8b0a533b9767571ec6622f2bd))
* add deprecated field to feature flag schema ([#410](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/410)) ([87bee57](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/87bee579d3be89de5535a0d68228b2ce8d33b828))
* add scaffold-resource workflow (autogen pipeline stage 2 v0) ([#446](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/446)) ([3025320](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/3025320681b776291cc48e0d04e83e9886043a79))
* add segment_approval_settings to launchdarkly_environment ([#339](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/339)) ([#464](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/464)) ([b553252](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/b553252a406a9fe86a850e7526911b10be8e8862))
* add state upgrade flow, add script to migrate between v2 and v3, add skill ([3694cee](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/3694cee50daaace3d4185faedf41b4001a5ce84e))
* adopt api-client-go v23 typed clients and field renames ([#494](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/494)) ([aefb9a3](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/aefb9a359c22858ed5cc8ae7770fe2e2a3b2ade9))
* **autogen:** add stage-1.5 triage workflow for unclaimed operations ([#474](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/474)) ([32a91d0](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/32a91d0577b64b6aea048e555f606d918e0255ba))
* **custom_role:** remove deprecated policy attribute ([#441](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/441)) ([66327fe](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/66327feaf176fdb83fdc016c1b3fb0e58ab30341))
* expose max_concurrency as an optional provider attribute ([#449](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/449)) ([20fb75e](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/20fb75ee5217fd188c4b7647faddac72452a5641))
* expose max_concurrency as an optional provider attribute (preview-v3) ([#450](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/450)) ([59ed7ad](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/59ed7ad45f9b5fbdefc766ee88531e0b05dcdc46))
* **feature_flag_environment:** make off_variation optional to model "Not set" ([#482](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/482)) ([#483](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/483)) ([11782b0](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/11782b0b6d0bf8657d0666c8f85d3e4be10325c4))
* **feature_flag:** remove deprecated include_in_snippet attribute ([#439](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/439)) ([2c88ac3](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/2c88ac32d29599395c1b44178ce4e4fbbb3dc99f))
* key-address launchdarkly_project environments (REL-14236) ([779e297](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/779e29736e2c2c7114f4d9dcebbc4ed172e1637b))
* **metric:** remove deprecated is_active attribute ([#438](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/438)) ([5ef4be1](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/5ef4be1f9948c060fdf18dc44f5bd85ea5c82bff))
* migrate complex resources (Phase 4) ([#423](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/423)) ([17ebe3b](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/17ebe3bab443ed8b0d914e1b4660da8ab3ad8c58))
* migrate data sources ([#418](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/418)) ([d4fc98b](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/d4fc98b98df8588cdc06f53f36f49a57e7a92199))
* migrate medium resources (Phase 3) ([#422](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/422)) ([c6bbc04](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/c6bbc047cde52fb7e59f1c1efbf3cd0a7562e05e))
* migrate simple resources ([#419](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/419)) ([008314b](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/008314bcd679a3867619ff257d0ce6e4c9b0e46b))
* **migrate-tf-syntax:** auto-synthesize required boolean variations ([866faaa](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/866faaadaaf5907243ed2e6069b9573742bbc587))
* net-new resource candidates in partial families (drift report) ([#458](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/458)) ([1fdf445](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/1fdf4458280697923291515f7e2682d1562f4f76))
* operation-level coverage in API drift report (autogen pipeline stage 1b) ([#456](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/456)) ([52f5ec2](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/52f5ec2bdf33d456ca9411959d2fc0b0893b89c5))
* per-family drift notifications + stage-3 verification agent ([#457](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/457)) ([7ec3be2](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/7ec3be28b032a96c9a15c0492c2a1a1d23d21f49))
* plan-time validation for prerequisite flag destroy ([#372](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/372)) ([#430](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/430)) ([d796a1c](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/d796a1cc14f363532c0d580f3624e5d1abd29e4a))
* port launchdarkly_context_kind to plugin framework ([#433](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/433)) ([db90c70](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/db90c70a608d1ecf39fdd4796aa5d728c37ea437))
* port policy_statements_json to framework custom_role ([#432](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/432)) ([f392bf6](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/f392bf6fd1a8464f0139fd39598e6186bd2bc742))
* port role_attributes on team_role_mapping to plugin framework ([#431](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/431)) ([b2f910f](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/b2f910f1dc7094fc6d8eb474d379838a0782eb9d))
* **project:** remove deprecated include_in_snippet and DS client_side_availability ([#440](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/440)) ([b188650](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/b1886503551820aace0e5cfb634658e2b82055d8))
* scaffold Announcements resource (autogen stage 2) ([#460](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/460)) ([6cd11bb](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/6cd11bbcf46a9a6376e225968e2993364add4a5e))
* scaffold big segment store integration resource (autogen stage 2) ([#468](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/468)) ([a1b799e](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/a1b799ee0f1589a60987448c3bff560290410901))
* scaffold Flag import configurations resource (autogen stage 2) ([#469](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/469)) ([2728f22](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/2728f2223e87d68f5bcd6af29a32e5acffd7a787))
* scaffold integration delivery configurations (beta) resource (autogen stage 2) ([#467](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/467)) ([95cd2c0](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/95cd2c06616a827ca72f94f6666d6158a4fd2fcf))
* scaffold launchdarkly_ai_agent_graph resource (autogen stage 2) ([#475](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/475)) ([8daf542](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/8daf542159d65d616f62b74be37915e78e94ab67))
* scaffold launchdarkly_sdk_key resource (autogen stage 2) ([#495](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/495)) ([7ab40a4](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/7ab40a41e4a23fdfb7f1be6b600a0b23b598874e))
* scaffold Metrics (beta) resource (autogen stage 2) ([#453](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/453)) ([3972d91](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/3972d91dbd5f485bf40dde75e3724bb687b77ec7))
* scaffold OAuth2 Clients resource (autogen stage 2) ([#466](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/466)) ([0cfb297](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/0cfb29706673b81953b8d429d923c2f3ac5a9e69))
* scaffold release policies (beta) resource (autogen stage 2) ([#471](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/471)) ([e4cab28](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/e4cab28a645d482fde61a9a2ea220b6cac2774b3))
* ship migrate-tf-syntax binaries and v3 migration guide ([#448](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/448)) ([644e5d8](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/644e5d8e43587d373015b33e3874c6d8e788c99f))
* use object syntax for single-object flag/project attributes (REL-14237) ([#480](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/480)) ([821d574](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/821d574d276e136ef63c1ca30494c439fe16072e))
* v3 RC prep — finish single-object and map conversions (REL-13579) ([#486](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/486)) ([21acf6d](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/21acf6dcd85712e92c68679dbd5bb04e77291996))


### Bug Fixes

* [REL-11737] Add pagination to teams resource nested fields roles and maintainers ([#375](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/375)) ([a22a7a0](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/a22a7a0d28a7fcdf2ce3d66d3effba6601b1c8db))
* disable Go cache in fork PR workflow to prevent cache poisoning ([#420](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/420)) ([6d0a5cc](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/6d0a5cc6be17adc93ca2b93a0b34e1cf8a828d39))
* **driftreport:** repair mapping.yaml so the drift report parses ([#473](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/473)) ([4bc9db7](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/4bc9db7c6552eed9af8d6bb9511ba57d6fde61d4))
* **feature_flag:** demote prereq destroy plan check to a warning ([#451](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/451)) ([ed6e6b0](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/ed6e6b0679d156ca2d78222bb3ccd5536e1e3573))
* **feature_flag:** preserve variation name and description when omitted from config ([d99725b](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/d99725b18cca78b61d45b3a432a66e94ca937a65))
* fix ip allowlist behaviour/tests ([#421](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/421)) ([5ddbb56](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/5ddbb5648211110e311652282afe7b25f0a107e3))
* handle segment create under segment approvals ([#370](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/370)) ([#463](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/463)) ([80f478b](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/80f478b7051c49cc163d939eb375f0fe1e9f643c))
* improve custom_properties hashing to resolve false / missing diffs ([#373](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/373)) ([ff36941](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/ff3694144eea34d0958b9d4b9d3d376378520f1c))
* **lint:** lowercase capitalized error strings in team_member_helper (ST1005) ([#477](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/477)) ([a3725e4](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/a3725e4195470148a57e3d205da018d50b6a4ebb))
* prevent nil-pointer panics in optional schema attributes and harden embedded-schema (Upjet) compatibility ([#387](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/387)) ([#415](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/415)) ([4844112](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/484411229387ba44ab40ce298f363e515eeb4cf8))
* **release:** ship migrate-tf-syntax binaries in preview releases ([#491](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/491)) ([7a64955](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/7a6495556827d217262fd160438558e443fcc914))
* remove deprecated `generate_sdk_keys` field from beta views resource ([#412](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/412)) ([bdf36e4](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/bdf36e481e26a4576f19e0b82046571d6eaece30))

## [3.0.0-beta.7](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v3.0.0-beta.6...v3.0.0-beta.7) (2026-07-07)


### ⚠ BREAKING CHANGES

* adopt api-client-go v23 typed clients and field renames ([#494](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/494))

### Features

* adopt api-client-go v23 typed clients and field renames ([#494](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/494)) ([aefb9a3](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/aefb9a359c22858ed5cc8ae7770fe2e2a3b2ade9))
* scaffold launchdarkly_sdk_key resource (autogen stage 2) ([#495](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/495)) ([7ab40a4](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/7ab40a41e4a23fdfb7f1be6b600a0b23b598874e))

## [3.0.0-beta.6](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v3.0.0-beta.5...v3.0.0-beta.6) (2026-07-06)


### Bug Fixes

* **release:** ship migrate-tf-syntax binaries in preview releases ([#491](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/491)) ([7a64955](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/7a6495556827d217262fd160438558e443fcc914))

## [3.0.0-beta.5](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v3.0.0-beta.4...v3.0.0-beta.5) (2026-07-02)


### ⚠ BREAKING CHANGES

* v3 RC prep — finish single-object and map conversions (REL-13579) ([#486](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/486))
* `launchdarkly_project.environments` is now a map keyed by environment key (`environments = { "production" = { ... } }`) instead of an ordered list. The inner `key` attribute is retained (Optional+Computed) and must equal the map key. The map is authoritative: an environment removed from it is deleted on apply; use `lifecycle { ignore_changes = [environments] }` to manage environments outside Terraform. Rewrite configurations with `migrate-tf-syntax` (it converts the blocks and warns on positional `environments[N]` references with the exact replacement — it does not rewrite them). The v2 -> v3 state upgrade is automatic; `3.0.0-beta.1`–`beta.3` list-shaped state must be re-imported.
* client_side_availability, defaults, default_client_side_availability, and fallthrough now use object syntax (`attr = {...}`) instead of a single-element list (`attr = [{...}]`).
* **access_token:** remove deprecated expire and policy_statements ([#442](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/442))
* **custom_role:** remove deprecated policy attribute ([#441](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/441))
* **project:** remove deprecated include_in_snippet and DS client_side_availability ([#440](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/440))
* **feature_flag:** remove deprecated include_in_snippet attribute ([#439](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/439))
* **metric:** remove deprecated is_active attribute ([#438](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/438))
* all user HCL using block syntax for approval_settings, variations, rules, targets, context_targets, prerequisites, fallthrough, client_side_availability, custom_properties, defaults, environments, policy, policy_statements, inline_roles, statements, role_attributes, included_contexts, excluded_contexts, urls, instructions, boolean_defaults, messages, segments, linked_segments must be rewritten to attribute syntax in v3.0.0.

### Features

* [bot] Regenerate integration configs ([#346](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/346)) ([15a0ef3](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/15a0ef3afb4d0e8ddcc9f9a0c8f68960b66819bf))
* [bot] Regenerate integration configs ([#391](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/391)) ([b772383](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/b7723839b7c4a45f5010c9990c755e28f1272509))
* [REL-12555] Release Views Resources from preview provider into main ([#400](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/400)) ([b718a8c](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/b718a8c489cff57fa5e75fd3b297d42cc69d7d8e))
* [REL-12731] - add support for flag templates ([#403](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/403)) ([927d50b](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/927d50b7113dc21d3a2a4cc48ef686be2d7b49c5))
* [REL-13052] add IP allowlist config and entry resources ([#411](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/411)) ([03a540b](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/03a540beb0dc302ad6d424ddf8ddbabd27cc78ae))
* **access_token:** remove deprecated expire and policy_statements ([#442](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/442)) ([68cb932](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/68cb932441186502dee8a2f467678642c3183155))
* add ai configs resources ([#404](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/404)) ([874bdec](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/874bdecc599212808089b9d5a0d6cced593d20c3))
* add API-coverage drift report (autogen pipeline stage 1) ([#445](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/445)) ([13c2b21](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/13c2b21ab49aa6a8b0a533b9767571ec6622f2bd))
* add deprecated field to feature flag schema ([#410](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/410)) ([87bee57](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/87bee579d3be89de5535a0d68228b2ce8d33b828))
* add scaffold-resource workflow (autogen pipeline stage 2 v0) ([#446](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/446)) ([3025320](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/3025320681b776291cc48e0d04e83e9886043a79))
* add segment_approval_settings to launchdarkly_environment ([#339](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/339)) ([#464](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/464)) ([b553252](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/b553252a406a9fe86a850e7526911b10be8e8862))
* add state upgrade flow, add script to migrate between v2 and v3, add skill ([3694cee](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/3694cee50daaace3d4185faedf41b4001a5ce84e))
* **autogen:** add stage-1.5 triage workflow for unclaimed operations ([#474](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/474)) ([32a91d0](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/32a91d0577b64b6aea048e555f606d918e0255ba))
* **custom_role:** remove deprecated policy attribute ([#441](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/441)) ([66327fe](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/66327feaf176fdb83fdc016c1b3fb0e58ab30341))
* expose max_concurrency as an optional provider attribute ([#449](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/449)) ([20fb75e](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/20fb75ee5217fd188c4b7647faddac72452a5641))
* expose max_concurrency as an optional provider attribute (preview-v3) ([#450](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/450)) ([59ed7ad](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/59ed7ad45f9b5fbdefc766ee88531e0b05dcdc46))
* **feature_flag_environment:** make off_variation optional to model "Not set" ([#482](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/482)) ([#483](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/483)) ([11782b0](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/11782b0b6d0bf8657d0666c8f85d3e4be10325c4))
* **feature_flag:** remove deprecated include_in_snippet attribute ([#439](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/439)) ([2c88ac3](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/2c88ac32d29599395c1b44178ce4e4fbbb3dc99f))
* key-address launchdarkly_project environments (REL-14236) ([779e297](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/779e29736e2c2c7114f4d9dcebbc4ed172e1637b))
* **metric:** remove deprecated is_active attribute ([#438](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/438)) ([5ef4be1](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/5ef4be1f9948c060fdf18dc44f5bd85ea5c82bff))
* migrate complex resources (Phase 4) ([#423](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/423)) ([17ebe3b](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/17ebe3bab443ed8b0d914e1b4660da8ab3ad8c58))
* migrate data sources ([#418](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/418)) ([d4fc98b](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/d4fc98b98df8588cdc06f53f36f49a57e7a92199))
* migrate medium resources (Phase 3) ([#422](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/422)) ([c6bbc04](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/c6bbc047cde52fb7e59f1c1efbf3cd0a7562e05e))
* migrate simple resources ([#419](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/419)) ([008314b](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/008314bcd679a3867619ff257d0ce6e4c9b0e46b))
* **migrate-tf-syntax:** auto-synthesize required boolean variations ([866faaa](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/866faaadaaf5907243ed2e6069b9573742bbc587))
* net-new resource candidates in partial families (drift report) ([#458](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/458)) ([1fdf445](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/1fdf4458280697923291515f7e2682d1562f4f76))
* operation-level coverage in API drift report (autogen pipeline stage 1b) ([#456](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/456)) ([52f5ec2](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/52f5ec2bdf33d456ca9411959d2fc0b0893b89c5))
* per-family drift notifications + stage-3 verification agent ([#457](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/457)) ([7ec3be2](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/7ec3be28b032a96c9a15c0492c2a1a1d23d21f49))
* plan-time validation for prerequisite flag destroy ([#372](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/372)) ([#430](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/430)) ([d796a1c](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/d796a1cc14f363532c0d580f3624e5d1abd29e4a))
* port launchdarkly_context_kind to plugin framework ([#433](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/433)) ([db90c70](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/db90c70a608d1ecf39fdd4796aa5d728c37ea437))
* port policy_statements_json to framework custom_role ([#432](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/432)) ([f392bf6](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/f392bf6fd1a8464f0139fd39598e6186bd2bc742))
* port role_attributes on team_role_mapping to plugin framework ([#431](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/431)) ([b2f910f](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/b2f910f1dc7094fc6d8eb474d379838a0782eb9d))
* **project:** remove deprecated include_in_snippet and DS client_side_availability ([#440](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/440)) ([b188650](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/b1886503551820aace0e5cfb634658e2b82055d8))
* scaffold Announcements resource (autogen stage 2) ([#460](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/460)) ([6cd11bb](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/6cd11bbcf46a9a6376e225968e2993364add4a5e))
* scaffold big segment store integration resource (autogen stage 2) ([#468](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/468)) ([a1b799e](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/a1b799ee0f1589a60987448c3bff560290410901))
* scaffold Flag import configurations resource (autogen stage 2) ([#469](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/469)) ([2728f22](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/2728f2223e87d68f5bcd6af29a32e5acffd7a787))
* scaffold integration delivery configurations (beta) resource (autogen stage 2) ([#467](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/467)) ([95cd2c0](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/95cd2c06616a827ca72f94f6666d6158a4fd2fcf))
* scaffold launchdarkly_ai_agent_graph resource (autogen stage 2) ([#475](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/475)) ([8daf542](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/8daf542159d65d616f62b74be37915e78e94ab67))
* scaffold Metrics (beta) resource (autogen stage 2) ([#453](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/453)) ([3972d91](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/3972d91dbd5f485bf40dde75e3724bb687b77ec7))
* scaffold OAuth2 Clients resource (autogen stage 2) ([#466](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/466)) ([0cfb297](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/0cfb29706673b81953b8d429d923c2f3ac5a9e69))
* scaffold release policies (beta) resource (autogen stage 2) ([#471](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/471)) ([e4cab28](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/e4cab28a645d482fde61a9a2ea220b6cac2774b3))
* ship migrate-tf-syntax binaries and v3 migration guide ([#448](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/448)) ([644e5d8](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/644e5d8e43587d373015b33e3874c6d8e788c99f))
* use object syntax for single-object flag/project attributes (REL-14237) ([#480](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/480)) ([821d574](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/821d574d276e136ef63c1ca30494c439fe16072e))
* v3 RC prep — finish single-object and map conversions (REL-13579) ([#486](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/486)) ([21acf6d](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/21acf6dcd85712e92c68679dbd5bb04e77291996))


### Bug Fixes

* [REL-10234] Imiller/rel 10234/terraform flag resource does not smoothly switch between rollout weights and variation ([#366](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/366)) ([c42cfa3](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/c42cfa3ee258f33b5d5347db7602d2ef86bfff91))
* [REL-11737] Add pagination to teams resource nested fields roles and maintainers ([#375](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/375)) ([a22a7a0](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/a22a7a0d28a7fcdf2ce3d66d3effba6601b1c8db))
* [REL-8483] limit concurrency on the client to address 429/timeouts issue ([#338](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/338)) ([f38b51f](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/f38b51f5009c955e804dee9f4b5206a344ac41be))
* disable Go cache in fork PR workflow to prevent cache poisoning ([#420](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/420)) ([6d0a5cc](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/6d0a5cc6be17adc93ca2b93a0b34e1cf8a828d39))
* **driftreport:** repair mapping.yaml so the drift report parses ([#473](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/473)) ([4bc9db7](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/4bc9db7c6552eed9af8d6bb9511ba57d6fde61d4))
* **feature_flag:** demote prereq destroy plan check to a warning ([#451](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/451)) ([ed6e6b0](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/ed6e6b0679d156ca2d78222bb3ccd5536e1e3573))
* **feature_flag:** preserve variation name and description when omitted from config ([d99725b](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/d99725b18cca78b61d45b3a432a66e94ca937a65))
* fix ip allowlist behaviour/tests ([#421](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/421)) ([5ddbb56](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/5ddbb5648211110e311652282afe7b25f0a107e3))
* handle segment create under segment approvals ([#370](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/370)) ([#463](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/463)) ([80f478b](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/80f478b7051c49cc163d939eb375f0fe1e9f643c))
* improve custom_properties hashing to resolve false / missing diffs ([#373](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/373)) ([ff36941](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/ff3694144eea34d0958b9d4b9d3d376378520f1c))
* **lint:** lowercase capitalized error strings in team_member_helper (ST1005) ([#477](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/477)) ([a3725e4](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/a3725e4195470148a57e3d205da018d50b6a4ebb))
* prevent nil-pointer panics in optional schema attributes and harden embedded-schema (Upjet) compatibility ([#387](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/387)) ([#415](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/415)) ([4844112](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/484411229387ba44ab40ce298f363e515eeb4cf8))
* remove deprecated `generate_sdk_keys` field from beta views resource ([#412](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/412)) ([bdf36e4](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/bdf36e481e26a4576f19e0b82046571d6eaece30))

## [3.0.0-beta.4](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v3.0.0-beta.3...v3.0.0-beta.4) (2026-07-02)


### ⚠ BREAKING CHANGES

* v3 RC prep — finish single-object and map conversions (REL-13579) ([#486](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/486))
* `launchdarkly_project.environments` is now a map keyed by environment key (`environments = { "production" = { ... } }`) instead of an ordered list. The inner `key` attribute is retained (Optional+Computed) and must equal the map key. The map is authoritative: an environment removed from it is deleted on apply; use `lifecycle { ignore_changes = [environments] }` to manage environments outside Terraform. Rewrite configurations with `migrate-tf-syntax` (it converts the blocks and warns on positional `environments[N]` references with the exact replacement — it does not rewrite them). The v2 -> v3 state upgrade is automatic; `3.0.0-beta.1`–`beta.3` list-shaped state must be re-imported.
* client_side_availability, defaults, default_client_side_availability, and fallthrough now use object syntax (`attr = {...}`) instead of a single-element list (`attr = [{...}]`).

### Features

* **autogen:** add stage-1.5 triage workflow for unclaimed operations ([#474](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/474)) ([32a91d0](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/32a91d0577b64b6aea048e555f606d918e0255ba))
* **feature_flag_environment:** make off_variation optional to model "Not set" ([#482](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/482)) ([#483](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/483)) ([11782b0](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/11782b0b6d0bf8657d0666c8f85d3e4be10325c4))
* key-address launchdarkly_project environments (REL-14236) ([779e297](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/779e29736e2c2c7114f4d9dcebbc4ed172e1637b))
* scaffold launchdarkly_ai_agent_graph resource (autogen stage 2) ([#475](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/475)) ([8daf542](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/8daf542159d65d616f62b74be37915e78e94ab67))
* use object syntax for single-object flag/project attributes (REL-14237) ([#480](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/480)) ([821d574](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/821d574d276e136ef63c1ca30494c439fe16072e))
* v3 RC prep — finish single-object and map conversions (REL-13579) ([#486](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/486)) ([21acf6d](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/21acf6dcd85712e92c68679dbd5bb04e77291996))


### Bug Fixes

* **driftreport:** repair mapping.yaml so the drift report parses ([#473](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/473)) ([4bc9db7](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/4bc9db7c6552eed9af8d6bb9511ba57d6fde61d4))
* **lint:** lowercase capitalized error strings in team_member_helper (ST1005) ([#477](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/477)) ([a3725e4](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/a3725e4195470148a57e3d205da018d50b6a4ebb))

## [3.0.0-beta.3](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v3.0.0-beta.2...v3.0.0-beta.3) (2026-06-24)


### Features

* add API-coverage drift report (autogen pipeline stage 1) ([#445](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/445)) ([13c2b21](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/13c2b21ab49aa6a8b0a533b9767571ec6622f2bd))
* add scaffold-resource workflow (autogen pipeline stage 2 v0) ([#446](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/446)) ([3025320](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/3025320681b776291cc48e0d04e83e9886043a79))
* add segment_approval_settings to launchdarkly_environment ([#339](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/339)) ([#464](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/464)) ([b553252](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/b553252a406a9fe86a850e7526911b10be8e8862))
* expose max_concurrency as an optional provider attribute (preview-v3) ([#450](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/450)) ([59ed7ad](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/59ed7ad45f9b5fbdefc766ee88531e0b05dcdc46))
* **migrate-tf-syntax:** auto-synthesize required boolean variations ([866faaa](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/866faaadaaf5907243ed2e6069b9573742bbc587))
* net-new resource candidates in partial families (drift report) ([#458](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/458)) ([1fdf445](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/1fdf4458280697923291515f7e2682d1562f4f76))
* operation-level coverage in API drift report (autogen pipeline stage 1b) ([#456](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/456)) ([52f5ec2](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/52f5ec2bdf33d456ca9411959d2fc0b0893b89c5))
* scaffold Announcements resource (autogen stage 2) ([#460](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/460)) ([6cd11bb](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/6cd11bbcf46a9a6376e225968e2993364add4a5e))
* scaffold big segment store integration resource (autogen stage 2) ([#468](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/468)) ([a1b799e](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/a1b799ee0f1589a60987448c3bff560290410901))
* scaffold Flag import configurations resource (autogen stage 2) ([#469](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/469)) ([2728f22](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/2728f2223e87d68f5bcd6af29a32e5acffd7a787))
* scaffold integration delivery configurations (beta) resource (autogen stage 2) ([#467](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/467)) ([95cd2c0](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/95cd2c06616a827ca72f94f6666d6158a4fd2fcf))
* scaffold Metrics (beta) resource (autogen stage 2) ([#453](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/453)) ([3972d91](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/3972d91dbd5f485bf40dde75e3724bb687b77ec7))
* scaffold OAuth2 Clients resource (autogen stage 2) ([#466](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/466)) ([0cfb297](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/0cfb29706673b81953b8d429d923c2f3ac5a9e69))
* scaffold release policies (beta) resource (autogen stage 2) ([#471](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/471)) ([e4cab28](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/e4cab28a645d482fde61a9a2ea220b6cac2774b3))
* ship migrate-tf-syntax binaries and v3 migration guide ([#448](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/448)) ([644e5d8](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/644e5d8e43587d373015b33e3874c6d8e788c99f))


### Bug Fixes

* **feature_flag:** demote prereq destroy plan check to a warning ([#451](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/451)) ([ed6e6b0](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/ed6e6b0679d156ca2d78222bb3ccd5536e1e3573))
* **feature_flag:** preserve variation name and description when omitted from config ([d99725b](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/d99725b18cca78b61d45b3a432a66e94ca937a65))
* handle segment create under segment approvals ([#370](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/370)) ([#463](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/463)) ([80f478b](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/80f478b7051c49cc163d939eb375f0fe1e9f643c))

## [3.0.0-beta.2](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v3.0.0-beta.1...v3.0.0-beta.2) (2026-06-10)


### ⚠ BREAKING CHANGES

* **access_token:** remove deprecated expire and policy_statements ([#442](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/442))
* **custom_role:** remove deprecated policy attribute ([#441](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/441))
* **project:** remove deprecated include_in_snippet and DS client_side_availability ([#440](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/440))
* **feature_flag:** remove deprecated include_in_snippet attribute ([#439](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/439))
* **metric:** remove deprecated is_active attribute ([#438](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/438))

### Features

* **access_token:** remove deprecated expire and policy_statements ([#442](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/442)) ([68cb932](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/68cb932441186502dee8a2f467678642c3183155))
* add state upgrade flow, add script to migrate between v2 and v3, add skill ([3694cee](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/3694cee50daaace3d4185faedf41b4001a5ce84e))
* **custom_role:** remove deprecated policy attribute ([#441](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/441)) ([66327fe](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/66327feaf176fdb83fdc016c1b3fb0e58ab30341))
* **feature_flag:** remove deprecated include_in_snippet attribute ([#439](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/439)) ([2c88ac3](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/2c88ac32d29599395c1b44178ce4e4fbbb3dc99f))
* **metric:** remove deprecated is_active attribute ([#438](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/438)) ([5ef4be1](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/5ef4be1f9948c060fdf18dc44f5bd85ea5c82bff))
* **project:** remove deprecated include_in_snippet and DS client_side_availability ([#440](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/440)) ([b188650](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/b1886503551820aace0e5cfb634658e2b82055d8))

## [3.0.0-beta.1](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v3.0.0-beta.0...v3.0.0-beta.1) (2026-05-22)

First preview release of the v3.0 line.

### ⚠ BREAKING CHANGES

* The provider has been fully migrated to the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework). All schema previously expressed as configuration blocks (`variations`, `rules`, `targets`, `context_targets`, `prerequisites`, `fallthrough`, `client_side_availability`, `custom_properties`, `defaults`, `environments`, `policy`, `policy_statements`, `inline_roles`, `statements`, `role_attributes`, `included_contexts`, `excluded_contexts`, `urls`, `instructions`, `boolean_defaults`, `messages`, `segments`, `linked_segments`, `approval_settings`) is now expressed as nested attributes. Existing HCL must be updated from block syntax to attribute syntax before upgrading.

### Highlights

* Full migration from the legacy Terraform SDKv2 to the Terraform Plugin Framework across all resources and data sources.
* All block-based schema replaced with nested attribute syntax, enabling clearer plan output, stronger validation, and better editor support.
* Plan-time validation for prerequisite flag destroy, surfacing invalid removals before apply rather than at the API.
* A handful of long-standing issues addressed along the way.

## [2.30.0](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v2.29.0...v2.30.0) (2026-06-11)


### Features

* expose max_concurrency as an optional provider attribute ([#449](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/449)) ([20fb75e](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/20fb75ee5217fd188c4b7647faddac72452a5641))


### Bug Fixes

* disable Go cache in fork PR workflow to prevent cache poisoning ([#420](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/420)) ([6d0a5cc](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/6d0a5cc6be17adc93ca2b93a0b34e1cf8a828d39))
* fix ip allowlist behaviour/tests ([#421](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/421)) ([5ddbb56](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/5ddbb5648211110e311652282afe7b25f0a107e3))

## [2.29.0](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v2.28.0...v2.29.0) (2026-05-08)


### Features

* [REL-13052] add IP allowlist config and entry resources ([#411](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/411)) ([03a540b](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/03a540beb0dc302ad6d424ddf8ddbabd27cc78ae))
* add deprecated field to feature flag schema ([#410](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/410)) ([87bee57](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/87bee579d3be89de5535a0d68228b2ce8d33b828))


### Bug Fixes

* prevent nil-pointer panics in optional schema attributes and harden embedded-schema (Upjet) compatibility ([#387](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/387)) ([#415](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/415)) ([4844112](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/484411229387ba44ab40ce298f363e515eeb4cf8))
* remove deprecated `generate_sdk_keys` field from beta views resource ([#412](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/412)) ([bdf36e4](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/bdf36e481e26a4576f19e0b82046571d6eaece30))

## [2.28.0](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v2.27.0...v2.28.0) (2026-04-20)


### Features

* [REL-12731] - add support for flag templates ([#403](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/403)) ([927d50b](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/927d50b7113dc21d3a2a4cc48ef686be2d7b49c5))
* add ai configs resources ([#404](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/404)) ([874bdec](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/874bdecc599212808089b9d5a0d6cced593d20c3))

## [2.27.0](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v2.26.2...v2.27.0) (2026-03-05)


### Features

* [bot] Regenerate integration configs ([#391](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/391)) ([b772383](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/b7723839b7c4a45f5010c9990c755e28f1272509))
* [REL-12555] Release Views Resources from preview provider into main ([#400](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/400)) ([b718a8c](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/b718a8c489cff57fa5e75fd3b297d42cc69d7d8e))

## [2.26.2](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v2.26.1...v2.26.2) (2026-01-22)


### Bug Fixes

* improve custom_properties hashing to resolve false / missing diffs ([#373](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/373)) ([ff36941](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/ff3694144eea34d0958b9d4b9d3d376378520f1c))

## [2.26.1](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v2.26.0...v2.26.1) (2026-01-20)


### Bug Fixes

* [REL-11737] Add pagination to teams resource nested fields roles and maintainers ([#375](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/375)) ([a22a7a0](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/a22a7a0d28a7fcdf2ce3d66d3effba6601b1c8db))

## [2.26.0](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v2.25.3...v2.26.0) (2025-11-10)


### Features

* [bot] Regenerate integration configs ([#346](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/346)) ([15a0ef3](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/15a0ef3afb4d0e8ddcc9f9a0c8f68960b66819bf))


### Bug Fixes

* [REL-10234] Imiller/rel 10234/terraform flag resource does not smoothly switch between rollout weights and variation ([#366](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/366)) ([c42cfa3](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/c42cfa3ee258f33b5d5347db7602d2ef86bfff91))

## [2.25.3](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v2.25.2...v2.25.3) (2025-07-18)


### Bug Fixes

* [REL-8483] limit concurrency on the client to address 429/timeouts issue ([#338](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/338)) ([f38b51f](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/f38b51f5009c955e804dee9f4b5206a344ac41be))
* [REL-8605] add documentation note on discrepancy in default base permissions with current API version ([#336](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/336)) ([53733ee](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/53733ee7eb2e1e8a62257faa6ef01369c5dd435c))

## [2.25.2](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v2.25.1...v2.25.2) (2025-07-01)


### Bug Fixes

* [REL-8490] remove ConflictsWith for unbounded and rules, included, excluded ([#324](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/324)) ([14a1980](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/14a1980deeffb1e9124c859ed80e9f082a89a279))

## [2.25.1](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v2.25.0...v2.25.1) (2025-05-27)


### Bug Fixes

* [REL-7954] update error messages to return properly ([#317](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/317)) ([755f43d](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/755f43dc700f14e00c75dc7616191e14e0110e0b))

## [2.25.0](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v2.24.0...v2.25.0) (2025-03-19)


### Features

* Add support for PagerDuty Events integration ([#305](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/305)) ([15dfb9d](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/15dfb9df42aa048bf78171cde520f885c59f029c))

## [2.24.0](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v2.23.1...v2.24.0) (2025-03-04)


### Features

* add auto apply to env approvals ([#295](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/295)) ([c546fbe](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/c546fbee93d58964472c2723ad14bbb3ce07a1b3))

## [2.23.1](https://github.com/launchdarkly/terraform-provider-launchdarkly/compare/v2.23.0...v2.23.1) (2025-02-17)


### Bug Fixes

* set `critical` property on environment resource ([#296](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/296)) ([3e3cd70](https://github.com/launchdarkly/terraform-provider-launchdarkly/commit/3e3cd70cb69211a3c689241c71a019e6d9b8b9fb))

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
