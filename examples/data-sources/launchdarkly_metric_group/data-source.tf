data "launchdarkly_metric_group" "checkout_funnel" {
  project_key = "example-project"
  key         = "checkout-funnel"
}
