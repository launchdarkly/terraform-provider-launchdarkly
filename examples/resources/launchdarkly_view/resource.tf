resource "launchdarkly_view" "example" {
  project_key = "example-project"
  key         = "example-view"
  name        = "Example View"
  description = "An example view for demonstration purposes"

  tags = [
    "terraform",
    "example"
  ]

  generate_sdk_keys = true
  maintainer_id     = "507f1f77bcf86cd799439011"
}

# Alternative example with team maintainer instead of individual maintainer
resource "launchdarkly_view" "team_maintained" {
  project_key         = "example-project"
  key                 = "team-view"
  name                = "Team Maintained View"
  description         = "A view maintained by a team"
  maintainer_team_key = "platform-team"

  tags = ["team-managed"]
}

# To import an existing view, use:
# terraform import launchdarkly_view.example example-project/example-view
