---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_destination"
description: |-
  Interact with the LaunchDarkly data export destinations API.
---

# launchdarkly_destination

Provides a LaunchDarkly Data Export Destination resource.

Data Export Destinations are locations that receive exported data. This resource allows you to configure destinations for the export of raw analytics data, including feature flag requests, analytics events, custom events, and more.

To learn more about data export, read [Data Export Documentation](https://docs.launchdarkly.com/integrations/data-export).

## Example Usage

Currently the following five types of destinations are available: kinesis, google-pubsub, mparticle, azure-event-hubs, and segment. Please note that config fields will vary depending on which destination you are trying to configure / access.

```hcl
resource "launchdarkly_destination" "example" {
  project_key = "example-project"
  env_key     = "example-env"
  name        = "example-kinesis-dest"
  kind        = "kinesis"
  config = {
    "region" : "us-east-1",
    "role_arn" : "arn:aws:iam::123456789012:role/marketingadmin",
    "stream_name" : "cat-stream"
  }
  on = true
  tags    = ["terraform"]
}
```

```hcl
resource "launchdarkly_destination" "example" {
  project_key = "example-project"
  env_key     = "example-env"
  name        = "example-pubsub-dest"
  kind        = "google-pubsub"
  config = {
    "project" : "example-pub-sub-project",
    "topic" : "example-topic"
  }
  on = true
  tags    = ["terraform"]
}
```

```hcl
resource "launchdarkly_destination" "example" {
  project_key = "example-project"
  env_key     = "example-env"
  name        = "example-mparticle-dest"
  kind        = "mparticle"
  config = {
    "api_key" : "apiKeyfromMParticle"
    "secret" : "mParticleSecret"
    "user_identity" : "customer_id"
    "environment" : "production"
  }
  on = true
  tags    = ["terraform"]
}
```

```hcl
resource "launchdarkly_destination" "example" {
	project_key = "example-project"
	env_key = "example-env"
	name    = "example-azure-event-hubs-dest"
	kind    = "azure-event-hubs"
	config  = {
		namespace = "example-azure-namespace"
		name = "example-azure-name"
		policy_name = "example-policy-name"
		policy_key = "azure-event-hubs-policy-key"
	}
	on = true
	tags = ["terraform"]
}
```

```hcl
resource "launchdarkly_destination" "example" {
  project_key = "example-project"
  env_key     = "example-env"
  name        = "example-segment-dest"
  kind        = "segment"
  config = {
    "write_key": "segment-write-key"
  }
  on = true
  tags    = ["terraform"]
}
```

## Argument Reference

- `project_key` - (Required) - The LaunchDarkly project key.

- `env_key` - (Required) - The environment key.

- `name` - (Required) - A human-readable name for your data export destination.

- `kind` - (Required) - The data export destination type. Available choices are `kinesis`, `google-pubsub`, `mparticle`, `azure-event-hubs`, and `segment`.

- `config` - (Required) - The destination-specific configuration. To learn more, read [Destination-Specific Configs](#destination-specific-configs).

- `enabled` - (Optional, **Deprecated**) - Whether the data export destination is on or not. This field argument is **deprecated** in favor of `on`. Please update your config to use to `on` to maintain compatibility with future versions.

- `on` - (Optional) - Whether the data export destination is on or not.

### Destination-Specific Configs

Depending on the destination kind, the `config` argument should contain the following fields:

#### Kinesis

- `region` - (Required) - AWS region your Kinesis resource resides in.

- `role_arn` - (Required) - Your AWS stream ARN in the format `"arn:aws:iam::{account-id}:role/{role}"`, ex. `"arn:aws:iam::123456789012:role/marketingadmin"`. Follow the directions in the [docs](https://docs.launchdarkly.com/integrations/data-export/kinesis) to set up the necessary roles if need be.

- `stream_name` - (Required) - The name of your Kinesis stream.

#### Google Pub/Sub

- `project` - (Required) - The name of your Pub/Sub project.

- `topic` - (Required) - The name of your Pub/Sub topic.

#### mParticle

- `api_key` - (Required) - Your mParticle API key.

- `secret` - (Required) - Your mParticle secret.

- `user_identity` - (Required) - Your mParticle user ID.

- `environment` - (Required) - The mParticle environment. Must be 'production' or 'development'.

### Azure Event Hubs

- `namespace` - (Required) - The Azure namespace where you want LaunchDarkly to export events.

- `name` - (Required) -

- `policy_name` - (Required) - The name of your Azure policy. Follow the directions in the [docs](https://docs.launchdarkly.com/home/data-export/event-hub#creating-a-policy-and-key-in-azure-event-hub) to set up a policy.

- `policy_key` - (Required) - Your Azure policy key. The name of your Azure policy. Follow the directions in the [docs](https://docs.launchdarkly.com/home/data-export/event-hub#creating-a-policy-and-key-in-azure-event-hub) to set up a policy.

### Segment

- `write_key` - (Required) - Your Segment write key.

## Attributes Reference

In addition to the arguments above, the resource exports the following attribute:

- `id` - The data export destination ID.

## Import

You can import a data export destination using the destination's full ID in the format `project_key/environment_key/id`.

For example:

```
$ terraform import launchdarkly_destination.example example-project/example-env/57c0af609969090743529967
```
