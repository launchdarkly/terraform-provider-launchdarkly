resource "launchdarkly_custom_role" "example_json" {
  key         = "example-role-key-json"
  name        = "example JSON role"
  description = "Equivalent role expressed as a JSON policy"

  policy_statements_json = jsonencode([
    {
      effect    = "allow"
      resources = ["proj/*:env/production:flag/*"]
      actions   = ["*"]
    },
    {
      effect    = "allow"
      resources = ["proj/*:env/production"]
      actions   = ["*"]
    }
  ])
}
