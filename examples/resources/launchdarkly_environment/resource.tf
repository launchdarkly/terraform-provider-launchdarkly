resource "launchdarkly_environment" "staging" {
  name  = "Staging"
  key   = "staging"
  color = "ff00ff"
  tags  = ["terraform", "staging"]

  project_key = launchdarkly_project.example.key
}

resource "launchdarkly_environment" "approvals_example" {
  name  = "Approvals Example Environment"
  key   = "approvals-example"
  color = "ff00ff"
  tags  = ["terraform", "staging"]

  approval_settings {
    required                   = true
    can_review_own_request     = true
    min_num_approvals          = 2
    can_apply_declined_changes = true
  }

  project_key = launchdarkly_project.example.key
}
