resource "launchdarkly_metric_group" "checkout_funnel" {
  project_key = launchdarkly_project.example.key
  key         = "checkout-funnel"
  name        = "Checkout funnel"
  kind        = "funnel"
  description = "Ordered funnel of the steps a customer takes through checkout"

  metrics = [
    {
      key           = launchdarkly_metric.viewed_cart.key
      name_in_group = "Viewed cart"
    },
    {
      key           = launchdarkly_metric.started_checkout.key
      name_in_group = "Started checkout"
    },
    {
      key           = launchdarkly_metric.completed_purchase.key
      name_in_group = "Completed purchase"
    },
  ]

  tags = ["checkout", "experimentation"]
}
