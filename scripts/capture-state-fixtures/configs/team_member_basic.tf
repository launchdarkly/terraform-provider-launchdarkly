terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "2.29.0"
    }
  }
}

# Note: capturing this fixture creates a real LD team member account. Use a
# disposable address such as `fixture+team-member@example.invalid` mapped to
# your test LD account's allowed-domains list.
resource "launchdarkly_team_member" "fixture" {
  email      = "fixture-team-member-PLACEHOLDER@example.invalid"
  first_name = "Fixture"
  last_name  = "Member"
  role       = "reader"
}
