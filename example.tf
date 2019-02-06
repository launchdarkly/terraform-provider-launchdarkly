resource "launchdarkly_project" "example1" {
  name = "Example Project with tags"
  key = "example1"
  tags = [
    "terraform",
    "is",
    "cool"]
}

resource "launchdarkly_environment" "staging" {
  name = "Staging"
  key = "staging"
  color = "0000f0"
  tags = [
    "terraform",
    "is",
    "cool"]

  project_key = "${launchdarkly_project.example1.key}"
}

output "api_key" {
  value = "${launchdarkly_environment.staging.api_key}"
}

output "mobile_key" {
  value = "${launchdarkly_environment.staging.mobile_key}"
}