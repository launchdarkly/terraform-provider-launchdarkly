Terraform Provider for LaunchDarkly
=====

- http://www.launchdarkly.com
- https://www.terraform.io/

This terraform provider covers the resources found in the [LaunchDarkly API docs](https://apidocs.launchdarkly.com/reference). 
The API client used is the generated [api-client-go](https://github.com/launchdarkly/api-client-go) which is based on the OpenAPI spec found [here](https://github.com/launchdarkly/ld-openapi). 

Requirements:
-------
- golang 1.12+ (1.11 should also work)
- make
- terraform 0.11.13+ (earlier versions may work)

Building The Provider
---------------------
This project uses [go modules](https://github.com/golang/go/wiki/Modules)

1. Clone the project: `git clone https://github.com/launchdarkly/terraform-provider-launchdarkly`
1. `make build`

Run unit tests:
`make test`

Run acceptance tests (Be sure to use a test account as this will create/destroy real resources!):
```
LAUNCHDARKLY_API_KEY=YOUR_API_KEY make testacc
```

Run example:
```
LAUNCHDARKLY_API_KEY=YOUR_API_KEY make apply
```