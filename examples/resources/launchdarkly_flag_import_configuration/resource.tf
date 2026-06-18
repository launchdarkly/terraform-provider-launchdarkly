resource "launchdarkly_flag_import_configuration" "split_import" {
  project_key     = launchdarkly_project.example.key
  integration_key = "split"
  name            = "Split flag import"

  # The accepted keys vary by integration and are described by the
  # integration's manifest formVariables. The keys below are those required by
  # the `split` integration.
  config = jsonencode({
    workspaceApiKey = var.split_workspace_api_key
    workspaceId     = var.split_workspace_id
    environmentId   = var.split_environment_id
    ldApiKey        = var.launchdarkly_api_key
  })

  tags = ["imported", "split"]
}
