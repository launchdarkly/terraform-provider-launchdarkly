provider "launchdarkly" {
  version = "~> 1.7.0"
}

# Create a token with a built in role
resource "launchdarkly_access_token" "reader_token" {
  role = "reader"
}

# Create a named token
resource "launchdarkly_access_token" "reader_token_with_name" {
  role = "reader"
  name = "Reader access token created by terraform"
}

# Create a token with custom roles
resource "launchdarkly_access_token" "token_with_custom_role" {
  custom_roles = ["terraform-project-reader"]
}

# Create a token with inline custom roles
resource "launchdarkly_access_token" "token_with_inline_roles" {
  inline_roles {
    actions   = ["*"]
    effect    = "deny"
    resources = ["proj/*:env/production"]
  }
}

# Create a token with inline custom roles (previously called policy_statements) **DEPRECATED**
resource "launchdarkly_access_token" "token_with_policy_statements" {
  inline_roles {
    actions   = ["*"]
    effect    = "deny"
    resources = ["proj/*:env/production"]
  }
}

# Create a service token
resource "launchdarkly_access_token" "service_token" {
  role          = "reader"
  service_token = true
}

# Create a token with default api version configured
# Your token will be using the latest api version by default. 
# However, you can also specify an api version as needed: https://apidocs.launchdarkly.com/reference#versioning
# Note: Some accounts are restricted to only use the latest API version (20240415)
resource "launchdarkly_access_token" "token_with_default_api_version" {
  role                = "reader"
  default_api_version = 20240415
}
