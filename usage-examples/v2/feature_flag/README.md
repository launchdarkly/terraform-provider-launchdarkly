## Example: feature flags

### Introduction

The LaunchDarkly provider provides two resources for configuring feature flags: [`launchdarkly_feature_flag`](https://www.terraform.io/docs/providers/launchdarkly/r/feature_flag.html), which allows you to configure and manipulate project-wide feature flag settings and [`launchdarkly_feature_flag_environment`](https://www.terraform.io/docs/providers/launchdarkly/r/feature_flag_environment.html), which allows you to manage environment-specific feature flag settings, such as [targeting rules](https://docs.launchdarkly.com/home/managing-flags/targeting-users) and [prerequisites](https://docs.launchdarkly.com/home/managing-flags/flag-prerequisites).

This example contains three config files:

- [setup.tf](./setup.tf), which auths the provider and creates a project under which the flags will be created
- [flag_types_example.tf](./flag_types_example.tf), which provides examples of the different ways you can define binary (boolean) and multivariate (string, numeric, and JSON) flag variations using the `launchdarkly_feature_flag` resource
- [targeting_example.tf](./targeting_example.tf), which provides complex examples of user targeting using the `launchdarkly_feature_flag_environment` resource. For more detail on user targeting, see the [official LaunchDarkly documentation](https://docs.launchdarkly.com/home/managing-flags/targeting-users).

### Run

Init your working directory from the CL with `terraform init` and then apply the changes with `terraform apply`. You should see output resembling the following:

```
An execution plan has been generated and is shown below.
Resource actions are indicated with the following symbols:
  + create

Terraform will perform the following actions:

  # launchdarkly_feature_flag.boolean_flag will be created
  + resource "launchdarkly_feature_flag" "boolean_flag" {
      + description        = "An example boolean feature flag that can be turned either on or off"
      + id                 = (known after apply)
      + include_in_snippet = (known after apply)
      + key                = "boolean-flag"
      + maintainer_id      = (known after apply)
      + name               = "Bool feature flag"
      + project_key        = "tf-flag-examples"
      + temporary          = false
      + variation_type     = "boolean"

      + client_side_availability {
          + using_environment_id = (known after apply)
          + using_mobile_key     = (known after apply)
        }

      + defaults {
          + off_variation = (known after apply)
          + on_variation  = (known after apply)
        }

      + variations {
          + description = (known after apply)
          + name        = (known after apply)
          + value       = (known after apply)
        }
    }

  # launchdarkly_feature_flag.json_flag will be created
  + resource "launchdarkly_feature_flag" "json_flag" {
      + description        = "An example of a multivariate feature flag with JSON variations"
      + id                 = (known after apply)
      + include_in_snippet = (known after apply)
      + key                = "json-flag"
      + maintainer_id      = (known after apply)
      + name               = "JSON-based feature flag"
      + project_key        = "tf-flag-examples"
      + tags               = [
          + "terraform-managed",
        ]
      + temporary          = false
      + variation_type     = "json"

      + client_side_availability {
          + using_environment_id = (known after apply)
          + using_mobile_key     = (known after apply)
        }

      + defaults {
          + off_variation = (known after apply)
          + on_variation  = (known after apply)
        }

      + variations {
          + value = jsonencode(
                {
                  + foo = "bar"
                }
            )
        }
      + variations {
          + value = jsonencode(
                {
                  + extra = {
                      + nested = "json"
                    }
                  + foo   = "baz"
                }
            )
        }
    }

  # launchdarkly_feature_flag.number_flag will be created
  + resource "launchdarkly_feature_flag" "number_flag" {
      + description        = "An example of a multivariate feature flag with numeric variations"
      + id                 = (known after apply)
      + include_in_snippet = (known after apply)
      + key                = "number-flag"
      + maintainer_id      = (known after apply)
      + name               = "Number value-based feature flag"
      + project_key        = "tf-flag-examples"
      + tags               = [
          + "terraform-managed",
        ]
      + temporary          = false
      + variation_type     = "number"

      + client_side_availability {
          + using_environment_id = (known after apply)
          + using_mobile_key     = (known after apply)
        }

      + defaults {
          + off_variation = (known after apply)
          + on_variation  = (known after apply)
        }

      + variations {
          + name  = "Big Number Variation"
          + value = "123000000"
        }
      + variations {
          + name  = "Small Number Variation"
          + value = "100"
        }
      + variations {
          + name  = "Float Variation"
          + value = "123.45"
        }
    }

  # launchdarkly_feature_flag.string_flag will be created
  + resource "launchdarkly_feature_flag" "string_flag" {
      + description        = "An example of a multivariate feature flag with string variations"
      + id                 = (known after apply)
      + include_in_snippet = (known after apply)
      + key                = "string-flag"
      + maintainer_id      = (known after apply)
      + name               = "String-based feature flag"
      + project_key        = "tf-flag-examples"
      + tags               = [
          + "terraform-managed",
        ]
      + temporary          = false
      + variation_type     = "string"

      + client_side_availability {
          + using_environment_id = (known after apply)
          + using_mobile_key     = (known after apply)
        }

      + defaults {
          + off_variation = (known after apply)
          + on_variation  = (known after apply)
        }

      + variations {
          + description = "one of three variations"
          + name        = "A String"
          + value       = "string1"
        }
      + variations {
          + name  = "Another String"
          + value = "string2"
        }
      + variations {
          + name  = "Yet Another String"
          + value = "string3"
        }
    }

  # launchdarkly_feature_flag_environment.prereq_flag will be created
  + resource "launchdarkly_feature_flag_environment" "prereq_flag" {
      + env_key       = "production"
      + flag_id       = (known after apply)
      + id            = (known after apply)
      + off_variation = 1
      + on            = true
      + track_events  = false

      + fallthrough {
          + variation = 0
        }

      + prerequisites {
          + flag_key  = "boolean-flag"
          + variation = 1
        }

      + rules {
          + clauses {
              + attribute  = "country"
              + negate     = false
              + op         = "matches"
              + value_type = "string"
              + values     = [
                  + "uk",
                  + "aus",
                  + "usa",
                ]
            }
        }
    }

  # launchdarkly_feature_flag_environment.user_targeting_flag will be created
  + resource "launchdarkly_feature_flag_environment" "user_targeting_flag" {
      + env_key       = "test"
      + flag_id       = (known after apply)
      + id            = (known after apply)
      + off_variation = 0
      + on            = true
      + track_events  = true

      + fallthrough {
          + bucket_by       = "company"
          + rollout_weights = [
              + 60000,
              + 30000,
              + 10000,
            ]
        }

      + rules {
          + variation = 0

          + clauses {
              + attribute  = "name"
              + negate     = false
              + op         = "startsWith"
              + value_type = "string"
              + values     = [
                  + "a",
                  + "b",
                  + "c",
                  + "d",
                  + "e",
                ]
            }
        }

      + targets {
          + values    = [
              + "test_user0",
            ]
          + variation = 0
        }
      + targets {
          + values    = [
              + "test_user1",
              + "test_user2",
            ]
          + variation = 1
        }
      + targets {
          + values    = [
              + "test_user3",
            ]
          + variation = 2
        }
    }

  # launchdarkly_project.tf_flag_examples will be created
  + resource "launchdarkly_project" "tf_flag_examples" {
      + id                 = (known after apply)
      + include_in_snippet = (known after apply)
      + key                = "tf-flag-examples"
      + name               = "Terraform Project for Flag Examples"
      + tags               = [
          + "terraform-managed",
        ]

      + client_side_availability {
          + using_environment_id = (known after apply)
          + using_mobile_key     = (known after apply)
        }
        
      + environments {
          + api_key              = (sensitive value)
          + client_side_id       = (sensitive value)
          + color                = "ababab"
          + confirm_changes      = (known after apply)
          + default_track_events = (known after apply)
          + default_ttl          = (known after apply)
          + key                  = "example-env"
          + mobile_key           = (sensitive value)
          + name                 = "example environment"
          + require_comments     = (known after apply)
          + secure_mode          = (known after apply)
        }
    }

Plan: 7 to add, 0 to change, 0 to destroy.
```

Since Terraform handles all the files in a given directory as a single configuration, all the configurations from the three files in this directory will be applied together. Once you have confirmed the changes to Terraform's prompt, it will apply them with output resembling the following:

```
launchdarkly_project.tf_flag_examples: Creating...
launchdarkly_project.tf_flag_examples: Creation complete after 0s [id=tf-flag-examples]
launchdarkly_feature_flag.boolean_flag: Creating...
launchdarkly_feature_flag.json_flag: Creating...
launchdarkly_feature_flag.number_flag: Creating...
launchdarkly_feature_flag.string_flag: Creating...
launchdarkly_feature_flag.boolean_flag: Creation complete after 1s [id=tf-flag-examples/boolean-flag]
launchdarkly_feature_flag.json_flag: Creation complete after 1s [id=tf-flag-examples/json-flag]
launchdarkly_feature_flag.string_flag: Creation complete after 1s [id=tf-flag-examples/string-flag]
launchdarkly_feature_flag.number_flag: Creation complete after 1s [id=tf-flag-examples/number-flag]
launchdarkly_feature_flag_environment.user_targeting_flag: Creating...
launchdarkly_feature_flag_environment.prereq_flag: Creating...
launchdarkly_feature_flag_environment.user_targeting_flag: Creation complete after 0s [id=tf-flag-examples/test/string-flag]
launchdarkly_feature_flag_environment.prereq_flag: Creation complete after 0s [id=tf-flag-examples/production/number-flag]

Apply complete! Resources: 7 added, 0 changed, 0 destroyed.
```

To view your flags, navigate to the Feature flags section on the left sidebar and search for the "terraform-managed" tag:

!["terraform-managed" tags](../assets/images/feature-flags-variation-types.png)

You should be able to view specific flag policies by clicking into them. Here you can see how the policies for the user_targeting_flag would look:

![user_targeting_flag policies](../assets/images/feature-flag-targeting.png)
