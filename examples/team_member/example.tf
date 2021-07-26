provider "launchdarkly" {
  version = ">= 1.6.0"
}

resource "launchdarkly_team_member" "joe" {
    email = "joe@example.com"
    first_name = "Joe"
    last_name = "Joeman"
    custom_roles = ["modified-writer", "test-role"]
}

resource "launchdarkly_team_member" "meishi" {
    email = "meishi@example.com"
    first_name = "Meishi"
    last_name = "Xie"
    role = "writer"
}
