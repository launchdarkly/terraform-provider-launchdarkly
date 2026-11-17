resource "launchdarkly_observability_filter_rule" "drop_health_checks" {
  project_key  = launchdarkly_project.example.key
  key          = "drop-health-checks"
  product_type = "logs"
  name         = "Drop health check logs"
  query        = "route=/healthz"
  operation    = "exclude"
}

resource "launchdarkly_observability_filter_rule" "sample_debug_logs" {
  project_key  = launchdarkly_project.example.key
  key          = "sample-debug-logs"
  product_type = "logs"
  name         = "Drop 90% of debug logs"
  query        = "level=debug"
  operation    = "exclude"
  rate         = 0.9
}

resource "launchdarkly_observability_filter_settings" "example" {
  project_key = launchdarkly_project.example.key

  # Omitted products have no limit.
  minute_rate_limits = {
    logs   = 100000
    traces = 20000
  }

  # Terraform owns the logs order: only these rules run, in this order.
  # Products that are not listed keep the order set in LaunchDarkly.
  rule_order = {
    logs = [
      launchdarkly_observability_filter_rule.drop_health_checks.key,
      launchdarkly_observability_filter_rule.sample_debug_logs.key,
    ]
  }
}
