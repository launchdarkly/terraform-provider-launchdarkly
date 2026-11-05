resource "launchdarkly_observability_filter_rule" "drop_health_checks" {
  project_key  = launchdarkly_project.example.key
  key          = "drop-health-checks"
  product_type = "logs"
  name         = "Drop health check logs"
  query        = "service_name=checkout-api route=/healthz"
  operation    = "exclude"
}

resource "launchdarkly_observability_filter_rule" "sample_staging_traces" {
  project_key  = launchdarkly_project.example.key
  key          = "sample-staging-traces"
  product_type = "traces"
  name         = "Keep 10% of staging traces"
  query        = "environment=staging"
  operation    = "include"
  rate         = 0.1
}

resource "launchdarkly_observability_filter_rule" "strip_pod_labels" {
  project_key  = launchdarkly_project.example.key
  key          = "strip-pod-labels"
  product_type = "metrics"
  name         = "Strip per-pod labels"
  query        = "service_name=checkout-api"
  operation    = "exclude"
  attributes   = ["k8s.pod.name", "k8s.pod.uid"]
}
