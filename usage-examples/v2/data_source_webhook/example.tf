// Use the webhook datasource to grab existing webhook information from LaunchDarkly. 
// All you need to provide is the id of the webhook.

terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "~> 2.0"
    }
  }
  required_version = ">= 0.13"
}

// Get data from LaunchDarkly on an existing webhook, all you need to do is provide the id
data "launchdarkly_webhook" "example" {
  id = "60f004b957922d2639124f6d"
}

// Print out the name of the "example" webhook we just recieved from LaunchDarkly
output "launchdarkly_webhook_print" {
  value     = data.launchdarkly_webhook.example.name
  sensitive = false
}
