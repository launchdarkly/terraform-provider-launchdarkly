# CONFIGURE PROJECT-LEVEL FLAGS
# ----------------------------------------------------------------------------------- #

# create a simple on/off flag for a feature
resource "launchdarkly_feature_flag" "binary_flag" {
  project_key    = launchdarkly_project.tf_full_config.key
  key            = "binary-flag"
  name           = "Binary feature flag"
  description    = "A binary flag for a feature that can be turned either on or off"
  variation_type = "boolean"
}

# create a multivariate flag with red, green, and blue variations
resource "launchdarkly_feature_flag" "multivariate_flag" {
  project_key = launchdarkly_project.tf_full_config.key
  key         = "multivariate-flag"
  name        = "Multivariate feature flag"
  description = "A multivariate flag with string variations"

  variation_type = "string"
  variations {
    name  = "Red"
    value = "red"
  }
  variations {
    name  = "Green"
    value = "green"
  }
  variations {
    name  = "Light blue"
    value = "light-blue"
  }
  tags = [
    "terraform"
  ]
}

# create a flag to be used in the staging environment for a feature for LD employees 
# see an example of an env-specific config in "env-staging.tf"
resource "launchdarkly_feature_flag" "ld_internal_tester" {
  project_key = launchdarkly_project.tf_full_config.key
  key         = "ld-internal-tester"
  name        = "LD Internal Tester"
  description = "A flag for LD employees to view pre-release features"

  variation_type = "number"
  variations {
    name  = "zero"
    value = 0
  }
  variations {
    name  = "one"
    value = 1
  }
  variations {
    name  = "two"
    value = 2
  }
}