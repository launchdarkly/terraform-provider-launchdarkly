# Generated framework variant of launchdarkly_team_role_mapping.
resource "launchdarkly_team" "base_team" {
  key  = "generated-team-role-mapping-example"
  name = "Generated Team Role Mapping Example"
}

resource "launchdarkly_custom_role" "role_0" {
  key              = "generated-mapping-role-0"
  name             = "Generated Mapping Role 0"
  base_permissions = "no_access"

  policy {
    actions   = ["*"]
    effect    = "deny"
    resources = ["proj/*:env/production"]
  }
}

resource "launchdarkly_custom_role" "role_1" {
  key              = "generated-mapping-role-1"
  name             = "Generated Mapping Role 1"
  base_permissions = "no_access"

  policy {
    actions   = ["*"]
    effect    = "deny"
    resources = ["proj/*:env/test"]
  }
}

resource "launchdarkly_generated_team_role_mapping" "example" {
  team_key = launchdarkly_team.base_team.key
  custom_role_keys = [
    launchdarkly_custom_role.role_0.key,
    launchdarkly_custom_role.role_1.key,
  ]
}
