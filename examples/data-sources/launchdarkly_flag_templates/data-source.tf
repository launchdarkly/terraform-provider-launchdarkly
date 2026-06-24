# Reads the "Custom" flag template settings (default tags, temporary, and boolean
# variation defaults) for a project. Only project_key is required; the rest are computed.
data "launchdarkly_flag_templates" "example" {
  project_key = "example-project"
}

# Reuse the project's default flag tags elsewhere in your configuration.
output "default_flag_tags" {
  value = data.launchdarkly_flag_templates.example.tags
}
