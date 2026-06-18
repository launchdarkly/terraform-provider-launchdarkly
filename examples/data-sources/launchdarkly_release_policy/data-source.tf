data "launchdarkly_release_policy" "guarded_checkout" {
  project_key = "example-project"
  key         = "guarded-checkout"
}
