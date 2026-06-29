resource "launchdarkly_experimentation_settings" "example" {
  project_key = launchdarkly_project.example.key

  randomization_units = [
    {
      randomization_unit = "user"
      default            = true
    },
    {
      randomization_unit = "account"
    },
  ]
}
