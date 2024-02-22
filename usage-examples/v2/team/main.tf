resource "launchdarkly_project" "team_project" {
  key  = "proj-team-maintainer"
  name = "Terraform Team Maintainer Project"
  # configure a production environment
  environments {
    name  = "A Production Environment"
    key   = "production"
    color = "123456"
    tags = [
      "terraform"
    ]
  }

  tags = [
    "terraform"
  ]
}

resource "launchdarkly_feature_flag" "team_flag" {
  project_key    = launchdarkly_project.team_project.key
  key            = "team-flag"
  name           = "team-maintained flag"
  description    = "A basic boolean flag maintained by a team"
  variation_type = "boolean"

  maintainer_team_key = launchdarkly_team.test_team.id
}

resource "launchdarkly_custom_role" "test_team" {
  key         = "test-team-custom-role"
  name        = "test team role"

  policy_statements {
    effect    = "allow"
    resources = ["proj/proj-team-maintainer:env/production:flag/*"]
    actions   = ["*"]
  }
}

resource "launchdarkly_team_member" "member_1" {
  email        = "member1@company.com"
  first_name   = "Katie"
  last_name    = "Lee"
  role         = "writer"
}

resource "launchdarkly_team_member" "member_2" {
  email        = "member2@company.com"
  first_name   = "Meishan"
  last_name    = "Xie"
  role         = "writer"
}

resource "launchdarkly_team" "test_team" {
  key                   = "test-team"
  name                  = "test team"
  description           = "Team to manage team project"
  member_ids            = [launchdarkly_team_member.member_1.id, launchdarkly_team_member.member_2.id]
  custom_role_keys      = ["test-team-custom-role"]
}

