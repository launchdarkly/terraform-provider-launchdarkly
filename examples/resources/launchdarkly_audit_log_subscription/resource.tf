resource "launchdarkly_audit_log_subscription" "example" {
  integration_key = "datadog"
  name            = "Example Datadog Subscription"
  config {
    api_key  = "yoursecretkey"
    host_url = "https://api.datadoghq.com"
  }
  tags = [
    "integrations",
    "terraform"
  ]
  statements {
    actions   = ["*"]
    effect    = "allow"
    resources = ["proj/*:env/*:flag/*"]
  }
}
