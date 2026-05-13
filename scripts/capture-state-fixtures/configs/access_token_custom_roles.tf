terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "2.29.0"
    }
  }
}

resource "launchdarkly_custom_role" "fixture_role" {
  key              = "fixture-role-a"
  name             = "fixture-role-a"
  description      = "fixture role A"
  base_permissions = "no_access"
  policy_statements {
    effect    = "allow"
    resources = ["proj/*"]
    actions   = ["*"]
  }
}

resource "launchdarkly_custom_role" "fixture_role_b" {
  key              = "fixture-role-b"
  name             = "fixture-role-b"
  description      = "fixture role B"
  base_permissions = "no_access"
  policy_statements {
    effect    = "allow"
    resources = ["proj/*"]
    actions   = ["*"]
  }
}

resource "launchdarkly_access_token" "fixture" {
  name         = "fixture-access-token-custom-roles"
  custom_roles = [launchdarkly_custom_role.fixture_role.key, launchdarkly_custom_role.fixture_role_b.key]
}
