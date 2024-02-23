## Example: Configure multiple LaunchDarkly projects

### Introduction

LaunchDarkly projects allow you to manage multiple business objectives from a single LaunchDarkly account. Every project has its own unique set of associated environments and feature flags. For more, please see the [LaunchDarkly official documentation](https://docs.launchdarkly.com/home/managing-flags/projects).

This directory contains an example of how one might configure multiple projects, each with their own environments and feature flags, in one go. Specifically, this example will create the following:

- a "Terraform Example Project 1" with the key `tf-project-1` containing:
  - a "Terraform Test Environment" with key `tf-test`
  - a "Terraform Production Environment" with key `tf-production`
  - a boolean "Basic feature flag" with key "basic-flag"
    - a `tf-test`-specific flag configuration attached to the "Basic feature flag"
- a "Terraform Example Project 2" with the key `tf-project-2` containing:
  - an "Example Environment A" with key `tf-example-env-a`
  - an "Example Environment B" with key `tf-example-env-b`
  - (automatically-created "Test" and "Production" environments will also be created since this example does not use nested environments in the `launchdarkly_project` resource)

Project 1 also contains an example of a feature flag that is configured differently in different environments using the `launchdarkly_feature_flag_environment` resource. You can see these differences from within the LaunchDarkly UI. When viewing the flag in the `tf_test` environment, the flag will look like this:

![basic flag in the tf_test env](../assets/images/multiple-proj-basic-env-flag.png)

From all other environments, it will simply look like this:

![basic flag](../assets/images/multiple-proj-basic-flag.png)

### Important notes

- Keep in mind that all new projects that don't use nested `environments` blocks will automatically come with 'test' and 'production' environments, so it may be redundant to create those!
- Project keys MUST be unique
- Environment keys must be unique within their given project

### Run

Init your working directory from the CL with `terraform init` and then apply the changes with `terraform apply`. You should see output resembling the following:

```
An execution plan has been generated and is shown below.
Resource actions are indicated with the following symbols:
  + create

Terraform will perform the following actions:

  # launchdarkly_environment.tf_env_a will be created
  + resource "launchdarkly_environment" "tf_env_a" {
      + api_key        = (sensitive value)
      + client_side_id = (sensitive value)
      + color          = "ff00ff"
      + id             = (known after apply)
      + key            = "tf-example-env-a"
      + mobile_key     = (sensitive value)
      + name           = "Example Environment A"
      + project_key    = "tf-project-2"
      + tags           = [
          + "rollouts",
          + "terraform-managed",
        ]
    }


...


Plan: 8 to add, 0 to change, 0 to destroy.
```

where all of the changes to be made should be described in the format seen above. Terraform will then ask you for your confirmation and then begin applying the changes:

```
launchdarkly_project.tf_project_1: Creating...
launchdarkly_project.tf_project_2: Creating...
launchdarkly_project.tf_project_2: Creation complete after 0s [id=example-project-2]
launchdarkly_environment.tf_env_a: Creating...
launchdarkly_environment.tf_env_b: Creating...
```

etc. Your projects should now be up, running, and accessible from your LaunchDarkly UI.
