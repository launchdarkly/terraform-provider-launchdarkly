## Example: webhook

### Introduction

LaunchDarkly webhooks allow you to build your own integrations that subscribe to changes in LaunchDarkly. For more information, see the [official LaunchDarkly documentation](https://docs.launchdarkly.com/integrations/webhooks).

This directory contains a very simple example of a webhook configuration using the [`launchdarkly_webhook` Terraform resource](https://www.terraform.io/docs/providers/launchdarkly/r/webhook.html). The created webhook posts changes in LaunchDarkly to a public bin located at [https://requestbin.com/r/enrl3l3jnmnwh](https://requestbin.com/r/enrl3l3jnmnwh) using the endpoint configured in the [`example.tf`](./example.tf) file.

### Run

Init your working directory from the CL with `terraform init` and then apply the changes with `terraform apply`. You should see output resembling the following:

```
An execution plan has been generated and is shown below.
Resource actions are indicated with the following symbols:
  + create

Terraform will perform the following actions:

  # launchdarkly_webhook.tf_example_webook will be created
  + resource "launchdarkly_webhook" "tf_example_webook" {
      + on = true
      + id      = (known after apply)
      + name    = "tf-example-webhook"
      + tags    = [
          + "terraform-managed",
        ]
      + url     = "https://enrl3l3jnmnwh.x.pipedream.net"

      + statements {
          + actions   = [
              + "*",
            ]
          + effect    = "allow"
          + resources = [
              + "proj/*:env/*:flag/*;terraform-managed",
            ]
        }
    }

Plan: 1 to add, 0 to change, 0 to destroy.
```

Terraform will then ask you for your confirmation to apply the changes:

```
launchdarkly_webhook.tf_example_webook: Creating...
launchdarkly_webhook.tf_example_webook: Creation complete after 1s [id=5e8f2e8393945a0848ab7b7c]

Apply complete! Resources: 1 added, 0 changed, 0 destroyed.
```
