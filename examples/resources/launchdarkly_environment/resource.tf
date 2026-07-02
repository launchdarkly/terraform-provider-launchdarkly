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

  approval_settings = {
    required                   = true
    can_review_own_request     = true
    min_num_approvals          = 2
    can_apply_declined_changes = true
  }

  project_key = launchdarkly_project.example.key
}

# Segment approvals are configured separately from flag approval_settings,
# via LaunchDarkly's beta approvals API. Note: enabling segment approvals
# while you manage launchdarkly_segment resources in Terraform will make
# every subsequent segment change require manual approval before it can be
# applied. See https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/370.
resource "launchdarkly_environment" "segment_approvals_example" {
  name  = "Segment Approvals Example Environment"
  key   = "segment-approvals-example"
  color = "ff00ff"
  tags  = ["terraform", "staging"]

  segment_approval_settings = {
    required                   = true
    can_review_own_request     = true
    min_num_approvals          = 2
    can_apply_declined_changes = true
  }

  project_key = launchdarkly_project.example.key
}
