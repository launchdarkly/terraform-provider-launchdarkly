# Payments team flags - each flag declares which views it belongs to

resource "launchdarkly_feature_flag" "stripe_integration" {
  project_key = var.project_key
  key         = "stripe-integration-v2"
  name        = "Stripe Integration V2"
  description = "New Stripe payment processing integration"
  
  variation_type = "boolean"
  
  # This flag belongs to the payments team view
  view_keys = [var.team_view_key]
  
  tags = ["payments", "stripe", "integration"]
}

resource "launchdarkly_feature_flag" "payment_retries" {
  project_key = var.project_key
  key         = "automatic-payment-retries"
  name        = "Automatic Payment Retries"
  description = "Automatically retry failed payments"
  
  variation_type = "boolean"
  
  # This flag belongs to the payments team view
  view_keys = [var.team_view_key]
  
  tags = ["payments", "retry-logic"]
}

resource "launchdarkly_feature_flag" "checkout_timeout" {
  project_key = var.project_key
  key         = "checkout-timeout-duration"
  name        = "Checkout Timeout Duration"
  description = "How long before checkout sessions expire"
  
  variation_type = "number"
  
  variations {
    value = 300  # 5 minutes
    name  = "Short"
  }
  variations {
    value = 900  # 15 minutes
    name  = "Medium"
  }
  variations {
    value = 1800 # 30 minutes
    name  = "Long"
  }
  
  defaults {
    on_variation  = 1
    off_variation = 0
  }
  
  # This flag is relevant to both payments team and shared features
  view_keys = [
    var.team_view_key,
    var.shared_view_key
  ]
  
  tags = ["payments", "checkout", "timeout"]
}

