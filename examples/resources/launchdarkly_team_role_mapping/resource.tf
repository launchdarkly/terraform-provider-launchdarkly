resource "launchdarkly_team_role_mapping" "platform_team" {
  team_key         = "platform_team"
  custom_role_keys = ["platform", "nomad-administrators"]
}

# Per-team role scoping: the same shared custom role can be assigned to
# multiple teams with different attribute values, so a single role definition
# (whose policy references e.g. `proj/*;$${roleAttribute/domain}`) is reused
# without forking.
resource "launchdarkly_team_role_mapping" "team_x" {
  team_key         = "team-x"
  custom_role_keys = ["my-shared-role"]
  role_attributes = {
    domain = ["DomainX"]
  }
}

resource "launchdarkly_team_role_mapping" "team_y" {
  team_key         = "team-y"
  custom_role_keys = ["my-shared-role"]
  role_attributes = {
    domain = ["DomainY"]
  }
}
