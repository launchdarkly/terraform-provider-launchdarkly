terraform {
  required_providers {
    launchdarkly = {
      source = "launchdarkly/launchdarkly"
    }
  }
}

provider "launchdarkly" {
  access_token       = var.ld_access_token
  observability_host = var.observability_host
}

variable "ld_access_token" {
  description = "LaunchDarkly personal access token (api-...)"
  type        = string
  sensitive   = true
}

variable "observability_host" {
  description = "Observability backend base URL (e.g. http://localhost:8082)"
  type        = string
}

variable "project_id" {
  description = "Observability project string ID"
  type        = string
}

resource "launchdarkly_alert" "test" {
  project_id    = var.project_id
  name          = "Terraform E2E Test Alert"
  product_type  = "Errors"
  function_type = "Count"

  slack_channels = ["#alerts-channel"]

  emails = ["oncall@example.com"]

  triggers {
    type        = "Constant"
    condition   = "Above"
    alert_value = 100
    warn_value  = 50
  }

  query    = "level=error"
  disabled = false
}

output "alert_id" {
  value = launchdarkly_alert.test.id
}
