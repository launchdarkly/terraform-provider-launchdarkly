# Development Environment Setup

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) 0.12.x (to run acceptance tests)
- [Go](https://golang.org/doc/install) 1.16 (to build the provider plugin)

## Quick Start

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (please check the [requirements](#requirements) before proceeding).

_Note:_ This project uses [Go Modules](https://blog.golang.org/using-go-modules) making it safe to work with it outside of your existing [GOPATH](http://golang.org/doc/code.html#GOPATH). The instructions that follow assume a directory in your home directory outside of the standard GOPATH (i.e `$HOME/development/terraform-providers/`).

## Building The Provider

Clone repository to: `$HOME/development/terraform-providers/`

```sh
$ mkdir -p $HOME/development/terraform-providers/; cd $HOME/development/terraform-providers/
$ git clone git@github.com:launchdarkly/terraform-provider-launchdarkly
```

To compile the provider, run `make build`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

```sh
$ make build
```

## Testing the provider

In order to test the provider, you can simply run `make test`.

```sh
$ make test
```

In order to run the full suite of Acceptance tests, run `make testacc`.

_Note:_ Acceptance tests create real LaunchDarkly resources, and require an enterprise account.

## Using the provider

With Terraform v0.14 and later, [development overrides for provider developers](https://www.terraform.io/docs/cli/config/config-file.html#development-overrides-for-provider-developers) can be leveraged in order to use the provider built from source.

To do this, populate a Terraform CLI configuration file (`~/.terraformrc` for all platforms other than Windows; `terraform.rc` in the `%APPDATA%` directory when using Windows) with at least the following options:

```hcl
provider_installation {
  dev_overrides {
    "launchdarkly/launchdarkly" = "[REPLACE WITH GOPATH]/bin"
  }
  direct {}
}
```
