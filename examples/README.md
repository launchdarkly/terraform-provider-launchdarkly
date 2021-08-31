# terraform-provider-launchdarkly examples

Example LaunchDarkly project configured using the LaunchDarkly Terraform provider

## Overview

This repository contains a series of directories containing detailed examples of how to configure LaunchDarkly via the Terraform provider. [Click here](https://www.terraform.io/docs/providers/launchdarkly/index.html) for the official documentation on the Terraform website. A description of the examples included here can be found in the [Contents](#contents) section.

> ! TAKE NOTE! Running `terraform apply` on any of these directories with your auth credentials will result in real resources being created that may cost real money. These are meant to be used as examples only and LaunchDarkly is not responsible for any costs incurred via testing.

Before getting started with the LaunchDarkly Terraform provider, make sure you have Terraform correctly installed and configured. For more on this, see the [Setup](#setup) section.

## Contents

- [v1](./v1) contains examples of configurations compatible with v1.X of the LaunchDarkly provider. Please note that this version is no longer maintained.
  - [v1/full_config](./v1/full_config) contains an example of a simple but fully fleshed-out LaunchDarkly [project](https://docs.launchdarkly.com/home/managing-flags/projects) configuration, including [environments](https://docs.launchdarkly.com/home/managing-flags/environments), [feature flags](https://docs.launchdarkly.com/home/managing-flags), [team members](https://docs.launchdarkly.com/home/account-security/managing-your-team), [roles](https://docs.launchdarkly.com/home/account-security/custom-roles), and [segments](https://docs.launchdarkly.com/home/managing-users/segments). It provides an example of how to organize a more complex configuration with multiple resources.
  - [v1/multiple_projects](./v1/multiple_projects) contains an example of the simultaneous configuration of multiple [projects](https://docs.launchdarkly.com/home/managing-flags/projects) and associated [environments](https://docs.launchdarkly.com/home/managing-flags/environments) and [flags](https://docs.launchdarkly.com/home/managing-flags) in a single file.
  - [v1/feature_flags](./v1/feature_flags) contains a full range of flag examples, covering both [flag variation types](https://docs.launchdarkly.com/home/managing-flags/flag-variations) and complex [targeting rules](https://docs.launchdarkly.com/home/managing-flags/targeting-users).
- [v2](./v2) contains examples of configurations compatible with v2+ of the LaunchDarkly provider.
  - [v2/full_config](./v2/full_config) contains an example of a simple but fully fleshed-out LaunchDarkly [project](https://docs.launchdarkly.com/home/managing-flags/projects) configuration with nested environments, [feature flags](https://docs.launchdarkly.com/home/managing-flags), [team members](https://docs.launchdarkly.com/home/account-security/managing-your-team), [roles](https://docs.launchdarkly.com/home/account-security/custom-roles), and [segments](https://docs.launchdarkly.com/home/managing-users/segments). It provides an example of how to organize a more complex configuration with multiple resources.
  - [v2/feature_flags](./v2/feature_flags) contains a full range of flag examples, covering both [flag variation types](https://docs.launchdarkly.com/home/managing-flags/flag-variations) and complex [targeting rules](https://docs.launchdarkly.com/home/managing-flags/targeting-users).
  - [v2/environment](./v2/environment) contains an example of how to configure a standalone [LaunchDarkly environment](https://docs.launchdarkly.com/home/organize/environments) in Terraform. Please note that this is only recommended in v2 of the provider if you wish to manage the encapsulating project outside of Terraform.
  - [v2/segment](./v2/segment) contains an example of a [LaunchDarkly segment](https://docs.launchdarkly.com/home/data-export/segment) configuration to send LD event notifications to an external endpoint.
  - [v2/webhook](./v2/webhook) contains an example of a [LaunchDarkly webhook](https://docs.launchdarkly.com/integrations/webhooks) configuration to send LD event notifications to an external endpoint.
  - [v2/data_source_destination](./v2/data_source_destination) provides an example of how to configure a [LaunchDarkly data export destination](https://docs.launchdarkly.com/home/data-export) data source for easy reference in other resources.
  - [v2/data_source_webhook](./v2/data_source_webhook) provides an example of how to configure a [LaunchDarkly webhook](https://docs.launchdarkly.com/integrations/webhooks) data source for easy reference in other resources.
- [access_token](./access_token) contains an example of how to configure LaunchDarkly [access tokens](https://docs.launchdarkly.com/home/account-security/api-access-tokens) using Terraform. This configuration is compatible with both v1 and v2 of the provider.
- [custom_role](./custom_role) contains an example of how to configure a [custom role](https://docs.launchdarkly.com/home/account-security/custom-roles) within LaunchDarkly using Terraform. This configuration is compatible with both v1 and v2 of the provider.
- [team_member](./team_member) contains an example of how to configure a [team member](https://docs.launchdarkly.com/home/members/managing) within LaunchDarkly using Terraform. This configuration is compatible with both v1 and v2 of the provider.

- For an example of how to configure your provider if using Terraform version 0.13 or above, please see the [terraform_0.13](./terraform_0.13).

## Setup

### Install Terraform

First and foremost, you need to make sure you have Terraform installed on the machine you will be applying the configurations from and that you meet the requirements listed on the [project readme](https://github.com/launchdarkly/terraform-provider-launchdarkly#requirements). For instructions on how to install Terraform, [see here](https://learn.hashicorp.com/terraform/getting-started/install.html).

### Configure LD Credentials

Before getting started with the LaunchDarkly provider, you need to ensure you have your LaunchDarkly credentials properly set up. All you will need for this is a LaunchDarkly access token, which you can create via the LaunchDarkly platform under Account settings > Authorization.

Once you have your access token in hand, there are several ways to set variables within your Terraform context. For the sake of ease, we've set the access token as an environmental variable named `LAUNCHDARKLY_ACCESS_TOKEN`. The provider configuration will then automatically access it from your environment so that your provider config should only have to contain

```
provider "launchdarkly" {
    version     = "~> 1.0"
}
```

> ! TAKE NOTE! If you are using Terraform version 0.13 or above, your provider block will be nested inside of your `terraform` block as seen below.

```
terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "~> 1.5"
    }
  }
  required_version = ">= 0.13"
}
```

Some resources or attributes, such as [webhook policy_statements](./webhook/example.tf), that were added later may require a provider version later than 1.0; check the [changelog](https://github.com/launchdarkly/terraform-provider-launchdarkly/blob/master/CHANGELOG.md) for more information on versions.

If you would prefer to define your variables some other way, see [Terraform's documentation on input variables](https://learn.hashicorp.com/terraform/getting-started/variables) for some other ways to do so.
