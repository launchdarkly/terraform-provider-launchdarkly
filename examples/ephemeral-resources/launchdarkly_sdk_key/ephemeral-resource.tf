# Ephemeral resources require Terraform 1.10 or later. Write-only arguments,
# such as value_wo below, require Terraform 1.11 or later.

ephemeral "launchdarkly_sdk_key" "mobile_analytics" {
  project_key     = "example-project"
  environment_key = "production"
  key             = "mobile-analytics-key"
}

# Store the SDK key in AWS SSM Parameter Store without writing it to the
# Terraform plan or state. Increment value_wo_version to write a new value.
resource "aws_ssm_parameter" "mobile_analytics_sdk_key" {
  name             = "/example-app/production/mobile-analytics-sdk-key"
  type             = "SecureString"
  value_wo         = ephemeral.launchdarkly_sdk_key.mobile_analytics.value
  value_wo_version = 1
}
