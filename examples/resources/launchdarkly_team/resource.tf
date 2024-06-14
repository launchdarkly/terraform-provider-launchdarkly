resource "launchdarkly_team" "platform_team" {
  key              = "platform_team"
  name             = "Platform team"
  description      = "Team to manage internal infrastructure"
  member_ids       = ["507f1f77bcf86cd799439011", "569f183514f4432160000007"]
  maintainers      = ["12ab3c45de678910abc12345"]
  custom_role_keys = ["platform", "nomad-administrators"]
}
