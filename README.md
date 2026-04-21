# Terraform Provider for LaunchDarkly

[LaunchDarkly](https://launchdarkly.com) is a continuous delivery platform that provides feature flags as a service and allows developers to iterate quickly and safely. Use the LaunchDarkly provider to interact with LaunchDarkly resources, such as projects, environments, feature flags, and more.

## Quick Starts

- [Using the provider](https://registry.terraform.io/providers/launchdarkly/launchdarkly/latest/docs)
- [Provider development](DEVELOPMENT.md)

## Documentation

Full, comprehensive documentation is available on the Terraform website:

https://registry.terraform.io/providers/launchdarkly/launchdarkly/latest/docs

## OpenAPI Generation

This repository includes an OpenAPI-driven generation pipeline for framework/mux artifacts and pilot acceptance tests.

- Generator docs: [docs/openapi-provider-generation.md](docs/openapi-provider-generation.md)
- Config: `templates/openapi-provider-gen/config.json`
- Catalog: `templates/openapi-provider-gen/catalog.auto.json`
- Generator command: `go run ./scripts/openapi-provider-gen --overlay ./templates/openapi-provider-gen/config.json --catalog ./templates/openapi-provider-gen/catalog.auto.json --template-dir ./templates/openapi-provider-gen --out-dir ./launchdarkly --tests-out-dir ./launchdarkly/tests`
- Generated framework resource types for v2 comparison: `launchdarkly_generated_team`, `launchdarkly_generated_team_role_mapping`, `launchdarkly_generated_project`
