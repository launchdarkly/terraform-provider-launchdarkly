# Example of environment with approval settings for both flags and segments

resource "launchdarkly_environment" "example" {
  name       = "Example Environment"
  key        = "example-env"
  color      = "FFFFFF"
  project_key = "example-project"

  # Flag approval settings (resource_kind defaults to "flag")
  approval_settings {
    resource_kind              = "flag"  # Optional, defaults to "flag"
    required                   = true
    can_review_own_request     = false
    min_num_approvals          = 2
    can_apply_declined_changes = false
    service_kind               = "launchdarkly"
  }

  # Segment approval settings
  approval_settings {
    resource_kind              = "segment"
    required                   = true
    can_review_own_request     = false
    min_num_approvals          = 1
    can_apply_declined_changes = true
    service_kind               = "launchdarkly"
  }

  # AI Config approval settings (if needed)
  # approval_settings {
  #   resource_kind              = "aiconfig"
  #   required                   = false
  #   can_review_own_request     = true
  #   min_num_approvals          = 1
  #   can_apply_declined_changes = true
  #   service_kind               = "launchdarkly"
  # }
}

# Example maintaining backwards compatibility (without resource_kind)
# This will default to flag approval settings
resource "launchdarkly_environment" "backwards_compatible" {
  name       = "Backwards Compatible Environment"
  key        = "backwards-compat"
  color      = "000000"
  project_key = "example-project"

  approval_settings {
    # No resource_kind specified - defaults to "flag"
    required                   = true
    can_review_own_request     = false
    min_num_approvals          = 1
    can_apply_declined_changes = true
    service_kind               = "launchdarkly"
  }
}
