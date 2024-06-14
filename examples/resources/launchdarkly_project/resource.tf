resource "launchdarkly_project" "example" {
  key  = "example-project"
  name = "Example project"

  tags = [
    "terraform",
  ]

  environments {
    key   = "production"
    name  = "Production"
    color = "EEEEEE"
    tags  = ["terraform"]
    approval_settings {
      can_review_own_request     = false
      can_apply_declined_changes = false
      min_num_approvals          = 3
      required_approval_tags     = ["approvals_required"]
    }
  }

  environments {
    key   = "staging"
    name  = "Staging"
    color = "000000"
    tags  = ["terraform"]
  }
}
