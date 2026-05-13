terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = "2.29.0"
    }
  }
}

resource "launchdarkly_project" "fixture" {
  key  = "fixture-project-1"
  name = "fixture-project-1"
  environments {
    key   = "fixture-env-test"
    name  = "fixture-env-test"
    color = "AABBCC"
  }
}

# The capture script must set view_maintainer_id to a real member ID from the
# test LD account, then the sanitiser replaces it with a placeholder.
variable "view_maintainer_id" {
  type        = string
  default     = "000000000000000000000000"
  description = "Member ID of the view's maintainer (24 hex chars)."
}

resource "launchdarkly_view" "fixture" {
  project_key   = launchdarkly_project.fixture.key
  key           = "fixture-view-basic"
  name          = "fixture-view-basic"
  description   = "fixture view basic"
  maintainer_id = var.view_maintainer_id
  tags          = ["fixture-tag"]
}
