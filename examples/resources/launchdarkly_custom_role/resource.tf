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

# Alternative example with view membership both directly and via role attribute
resource "launchdarkly_custom_role" "view_membership_example" {
  key         = "example-role-key-2"
  name        = "example role with view membership"
  description = "This is an example role with view membership"

  policy_statements {
    effect    = "allow"
    resources = ["proj/*:env/production:flag/*;view:example-view"]
    actions   = ["*"]
  }
  policy_statements {
    effect    = "allow"
    resources = ["proj/*:env/production:flag/*;view:$${roleAttribute/view}"]
    actions   = ["*"]
  }
  policy_statements {
    effect    = "allow"
    resources = ["proj/*:env/production"]
    actions   = ["*"]
  }
}

