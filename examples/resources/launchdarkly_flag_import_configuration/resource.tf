resource "launchdarkly_flag_import_configuration" "split_import" {
  project_key     = launchdarkly_project.example.key
  integration_key = "split"
  name            = "Split flag import"

  # The accepted keys vary by integration and are described by the
  # integration's manifest formVariables. The example below targets `split`.
  config = jsonencode({
    apiToken = var.split_admin_token
    source   = "production"
  })

  tags = ["imported", "split"]
}
