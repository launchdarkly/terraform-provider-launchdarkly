data "launchdarkly_observability_filter_rules" "logs" {
  project_key  = "example-project"
  product_type = "logs"
}
