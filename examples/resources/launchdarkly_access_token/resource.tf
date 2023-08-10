resource "launchdarkly_access_token" "reader_token" {
  name = "Reader token managed by Terraform"
  role = "reader"
}

# With a custom role
resource "launchdarkly_access_token" "custom_role_token" {
  name         = "DevOps"
  custom_roles = ["ops"]
}

# With an inline custom role (policy statements)
resource "launchdarkly_access_token" "token_with_policy_statements" {
  name = "Integration service token"
  inline_roles {
    actions   = ["*"]
    effect    = "deny"
    resources = ["proj/*:env/production"]
  }
  service_token = true
}
