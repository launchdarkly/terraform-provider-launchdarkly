resource "launchdarkly_flag_defaults" "example" {
  project_key = "my-project"

  tags      = ["terraform"]
  temporary = false

  boolean_defaults {
    true_display_name  = "True"
    false_display_name = "False"
    true_description   = ""
    false_description  = ""
    on_variation       = 0
    off_variation      = 1
  }
}
