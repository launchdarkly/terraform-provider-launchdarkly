terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "2.29.0"
    }
  }
}

resource "launchdarkly_custom_role" "fixture" {
  key              = "fixture-custom-role-basic"
  name             = "fixture-custom-role-basic"
  description      = "fixture custom role basic"
  base_permissions = "no_access"
  policy_statements {
    effect    = "allow"
    resources = ["proj/*"]
    actions   = ["createProject", "deleteProject"]
  }
}
