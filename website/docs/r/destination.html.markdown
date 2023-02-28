---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_destination"
description: |-
  Interact with the LaunchDarkly data export destinations API.
---

# launchdarkly_destination

Provides a LaunchDarkly Data Export Destination resource.

-> **Note:** Data Export is available to customers on an Enterprise LaunchDarkly plan. To learn more, read about our pricing. To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).

Data Export Destinations are locations that receive exported data. This resource allows you to configure destinations for the export of raw analytics data, including feature flag requests, analytics events, custom events, and more.

To learn more about data export, read [Data Export Documentation](https://docs.launchdarkly.com/integrations/data-export).

## Example Usage

Currently the following five types of destinations are available: kinesis, google-pubsub, mparticle, azure-event-hubs, and segment. Please note that config fields will vary depending on which destination you are trying to configure / access.

```hcl
resource "launchdarkly_destination" "kinesis_example" {
  project_key = "example-project"
  env_key     = "example-env"
  name        = "example-kinesis-dest"
  kind        = "kinesis"
  config = {
    region = "us-east-1"
    role_arn = "arn:aws:iam::123456789012:role/marketingadmin"
    stream_name = "cat-stream"
  }
  on = true
  tags = ["terraform"]
}
```

```hcl
resource "launchdarkly_destination" "pubsub_example" {
  project_key = "example-project"
  env_key     = "example-env"
  name        = "example-pubsub-dest"
  kind        = "google-pubsub"
  config = {
    project = "example-pub-sub-project"
    topic = "example-topic"
  }
  on = true
  tags = ["terraform"]
}
```

```hcl
resource "launchdarkly_destination" "mparticle_example" {
  project_key = "example-project"
  env_key     = "example-env"
  name        = "example-mparticle-dest"
  kind        = "mparticle"
  config = {
    api_key = "apiKeyfromMParticle"
    secret = "mParticleSecret"
    user_identities = jsonencode([
      {"ldContextKind":"user","mparticleUserIdentity":"customer_id"},
      {"ldContextKind":"device","mparticleUserIdentity":"google"}]
		)
    environment = "production"
  }
  on = true
  tags = ["terraform"]
}
```

```hcl
resource "launchdarkly_destination" "azure_example" {
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
resource "launchdarkly_destination" "segment_example" {
  project_key = "example-project"
  env_key     = "example-env"
  name        = "example-segment-dest"
  kind        = "segment"
  config = {
    write_key = "segment-write-key"
    user_id_context_kind = "user"
    anonymous_id_context_kind = "anonymousUser"
  }
  on = true
  tags    = ["terraform"]
}
```

## Argument Reference

- `project_key` - (Required) - The LaunchDarkly project key. A change in this field will force the destruction of the existing resource and the creation of a new one.

- `env_key` - (Required) - The environment key. A change in this field will force the destruction of the existing resource and the creation of a new one.

- `name` - (Required) - A human-readable name for your data export destination.

- `kind` - (Required) - The data export destination type. Available choices are `kinesis`, `google-pubsub`, `mparticle`, `azure-event-hubs`, and `segment`. A change in this field will force the destruction of the existing resource and the creation of a new one.

- `config` - (Required) - The destination-specific configuration. To learn more, read [Destination-Specific Configs](#destination-specific-configs).

- `on` - (Optional, previously `enabled`) - Whether the data export destination is on or not.

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

- `user_identity` - (Optional) - Your mParticle user ID as a string. If defined, the LaunchDarkly context kind will be implicitly assumed to be "user". At least one of `user_identity` or `user_identities` must be defined.

- `user_identities` - (Optional) - A json-encoded list of objects associating mParticle user identities with LaunchDarkly context kinds. At least one of `user_identity` or `user_identities` must be defined.

- `environment` - (Required) - The mParticle environment. Must be 'production' or 'development'.

### Azure Event Hubs

- `namespace` - (Required) - The Azure namespace where you want LaunchDarkly to export events.

- `name` - (Required) -

- `policy_name` - (Required) - The name of your Azure policy. Follow the directions in the [docs](https://docs.launchdarkly.com/home/data-export/event-hub#creating-a-policy-and-key-in-azure-event-hub) to set up a policy.

- `policy_key` - (Required) - Your Azure policy key. The name of your Azure policy. Follow the directions in the [docs](https://docs.launchdarkly.com/home/data-export/event-hub#creating-a-policy-and-key-in-azure-event-hub) to set up a policy.

### Segment

- `write_key` - (Required) - Your Segment write key.

- `user_id_context_kind` - (Required) - The context kind you would like to associated with the data exported to segment.

- `anonymous_id_context_kind` - (Required) - The context kind you would like to associated with anonymous user data exported to segment.

## Attributes Reference

In addition to the arguments above, the resource exports the following attribute:

- `id` - The data export destination ID.

## Import

You can import a data export destination using the destination's full ID in the format `project_key/environment_key/id`.

For example:

```
$ terraform import launchdarkly_destination.example example-project/example-env/57c0af609969090743529967
```
