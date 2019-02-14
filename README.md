# Terraform Provider for LaunchDarkly

Work in Progress.

Currently supported resources:
- project
- environment
- feature flag (partial)

## Development requirements:
- golang 1.11+
- make
- terraform

## To run example:
```bash
 LAUNCHDARKLY_API_KEY=<api key> make build apply

```