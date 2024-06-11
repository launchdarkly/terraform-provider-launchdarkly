data "launchdarkly_flag_trigger" "example" {
  id          = "61d490757f7821150815518f"
  flag_key    = "example-flag"
  project_key = "the-big-project"
  env_key     = "production"
}
