resource "launchdarkly_custom_role" "example" {
  key         = "example-role-key-1"
  name        = "example role"
  description = "This is an example role"

  policy_statements {
    effect    = "allow"
    resources = ["proj/*:env/production:flag/*"]
    actions   = ["*"]
  }
  policy_statements {
    effect    = "allow"
    resources = ["proj/*:env/production"]
    actions   = ["*"]
  }
}
