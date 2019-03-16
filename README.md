# Terraform Provider for LaunchDarkly

Work in Progress.

Currently supported resources:
- project
- environment
- feature flag (partial)
- Custom Roles

## Development requirements:
- golang 1.12+ (1.11 should also work)
- make
- terraform 0.11.13+

#  WARNING: Use a test/demo account since various make targets/acceptance tests may wipe out all existing projects/settings!!

## To run example:
```bash
 LAUNCHDARKLY_API_KEY=<api key> make build apply

```