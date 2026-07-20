resource "launchdarkly_project" "example" {
  key  = "example-project"
  name = "Example project"

  tags = [
    "terraform",
  ]

  # Require new flags and segments to be associated with a view
  require_view_association_for_new_flags    = false
  require_view_association_for_new_segments = false

  # environments is a map keyed by the environment key (the map key and the
  # nested `key` are the same value). Reordering, adding, or removing one
  # environment does not affect the others. The map is authoritative: an
  # environment removed from it is deleted. To manage environments outside
  # Terraform instead, add `lifecycle { ignore_changes = [environments] }`.
  environments = {
    "production" = {
      key   = "production"
      name  = "Production"
      color = "EEEEEE"
      tags  = ["terraform"]
      approval_settings = {
        can_review_own_request     = false
        can_apply_declined_changes = false
        min_num_approvals          = 3
        required_approval_tags     = ["approvals_required"]
      }
    }
    "staging" = {
      key   = "staging"
      name  = "Staging"
      color = "000000"
      tags  = ["terraform"]
    }
  }
}
