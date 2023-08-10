# create a segment associated with the production environment for users that 
# have signed up to see experimental features
resource "launchdarkly_segment" "experimental_features" {
  key         = "experimental-features"
  project_key = launchdarkly_project.tf_full_config.key
  env_key     = "production"
  name        = "Experimental Feature Testers"
  description = "the set of users that will see experimental features"
  tags = [
    "terraform",
    "testing"
  ]
  rules {
    clauses {
      attribute = "tester"
      op        = "matches"
      values    = ["true"]
      negate    = false
    }
    weight    = 50000
    bucket_by = "region"
  }
}