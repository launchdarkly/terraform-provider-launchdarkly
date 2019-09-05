# Terraform Provider for LaunchDarkly

- http://www.launchdarkly.com
- https://www.terraform.io/

This terraform provider covers the resources found in the [LaunchDarkly API docs](https://apidocs.launchdarkly.com/reference).
The API client used is the generated [api-client-go](https://github.com/launchdarkly/api-client-go) which is based on the OpenAPI spec found [here](https://github.com/launchdarkly/ld-openapi).

## Requirements:

- golang 1.12+ (1.11 should also work)
- make
- terraform 0.12.0+

## Building The Provider

This project uses [go modules](https://github.com/golang/go/wiki/Modules)

1. Clone the project: `git clone https://github.com/launchdarkly/terraform-provider-launchdarkly`
1. `make build`

Run unit tests:
`make test`

Run acceptance tests (Be sure to use a test account as this will create/destroy real resources!):

```
LAUNCHDARKLY_API_KEY=YOUR_API_KEY make testacc
```

Note: you may need to clean your account before running the acceptance tests.
Do this by commenting out the `t.SkipNow()` line in [launchdarkly/account_cleaner_test.go](launchdarkly/account_cleaner_test.go)

Run [example.tf](example.tf):

```
LAUNCHDARKLY_API_KEY=YOUR_API_KEY make apply
```

## More examples:

See acceptance tests in `launchdarkly/resource_launchdarkly_*_test.go` for many examples.

## Known issues/Next steps:

1. Tags for environments is not yet supported. Stay tuned.
1. ~~Update terraform to [0.12](https://www.terraform.io/upgrade-guides/0-12.html)~~ This may help address tags for environments!
1. Add CI build
