terraform {
  required_providers {
    launchdarkly = {
      version = "~> 2.6.1"
      source  = "launchdarkly/launchdarkly"
    }
  }
}

provider "launchdarkly" {
  access_token = "api-c72ff490-9db2-40a5-8e58-fe886df57776"
  api_host     = "https://app.launchdarkly.com"
}

resource "launchdarkly_team" "terraform_test" {
  key              = "isabelle-test-team"
  name             = "Isabelle Test Team"
  description      = "testing out terraform"
  custom_role_keys = ["approvals-only"]
  member_ids       = ["610180aa0755fa2648b36856", "61a6483d333d961037f43f09"]
}
