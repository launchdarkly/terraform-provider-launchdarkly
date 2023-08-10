data "launchdarkly_team_member" "spongebob" {
  email = "spongebob@squarepants.net"
}

resource "launchdarkly_team" "krusty_krab_staff" {
  key         = "krusty_krab_staff"
  name        = "Krusty Krab staff"
  description = "Team serving Krabby patties"
  members     = [data.launchdarkly_team_member.spongebob.id]

  lifecycle {
    ignore_changes = [member_ids]
  }
}
