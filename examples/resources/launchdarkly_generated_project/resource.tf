# Generated framework variant of launchdarkly_project for side-by-side comparison.
resource "launchdarkly_generated_project" "example" {
  key                = "generated-project-example"
  name               = "Generated Project Example"
  include_in_snippet = false
  tags               = ["generated", "comparison"]

  environments {
    key   = "generated-env"
    name  = "Generated Environment"
    color = "010101"
  }
}
