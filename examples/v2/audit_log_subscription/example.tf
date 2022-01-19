terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "~> 2.0"
    }
  }
  required_version = ">= 0.13"
}

resource "launchdarkly_audit_log_subscription" "datadog_example" {
  integration_key = "datadog"
  name            = "Example Terraform Subscription"
  config = {
    api_key  = "thisisasecretkey"
    host_url = "https://api.datadoghq.com"
  }
  on   = false
  tags = ["terraform-managed"]
  statements {
    actions   = ["*"]
    effect    = "deny"
    resources = ["proj/*:env/*:flag/*"]
  }
}

resource "launchdarkly_audit_log_subscription" "dynatrace_example" {
  integration_key = "dynatrace"
  name            = "Example Terraform Subscription"
  config = {
    api_token = "verysecrettoken"
    url       = "https://launchdarkly.appdynamics.com"
    entity    = "APPLICATION_METHOD"
  }
  tags = ["terraform-managed"]
  on   = true
  statements {
    actions   = ["*"]
    effect    = "deny"
    resources = ["proj/*:env/test:flag/*"]
  }
}

resource "launchdarkly_audit_log_subscription" "splunk_example" {
  integration_key = "splunk"
  name            = "Example Terraform Subscription"
  config = {
    base_url             = "https://launchdarkly.splunk.com"
    token                = "averysecrettoken"
    skip_ca_verification = true
  }
  tags = ["terraform-managed"]
  on   = true
  statements {
    actions   = ["*"]
    effect    = "allow"
    resources = ["proj/*:env/production:flag/*"]
  }
}

