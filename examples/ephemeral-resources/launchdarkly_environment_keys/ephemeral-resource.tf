# Ephemeral resources require Terraform 1.10 or later. Write-only arguments,
# such as value_wo below, require Terraform 1.11 or later.

resource "launchdarkly_environment" "production" {
  name        = "Production"
  key         = "production"
  color       = "ff0000"
  project_key = "example-project"

  # Optional: also keep the SDK key and mobile key out of this resource's state.
  exclude_keys_from_state = true
}

ephemeral "launchdarkly_environment_keys" "production" {
  project_key = launchdarkly_environment.production.project_key
  env_key     = launchdarkly_environment.production.key
}

# Store the SDK key in AWS SSM Parameter Store without writing it to the
# Terraform plan or state. Increment value_wo_version to write a new value,
# for example after you rotate the SDK key in LaunchDarkly.
resource "aws_ssm_parameter" "launchdarkly_sdk_key" {
  name             = "/example-app/production/launchdarkly-sdk-key"
  type             = "SecureString"
  value_wo         = ephemeral.launchdarkly_environment_keys.production.api_key
  value_wo_version = 1
}
