terraform {
  required_providers {
    launchdarkly = {
      source  = "launchdarkly/launchdarkly"
      version = ">= 2.7.0"
    }
  }
  required_version = ">= 0.13"
}


resource "launchdarkly_project" "dina-project" {
  key  = "dina-web"
  name = "Dina Web"

  tags = [
    "terraform",
  ]

  environments {
    key   = "production"
    name  = "Production"
    color = "CE3146"
    tags  = ["terraform", ]
    # approval_settings {
    #   required                   = true
    #   can_review_own_request     = false
    #   can_apply_declined_changes = false
    #   min_num_approvals          = 2
    # }
  }

  environments {
    key   = "staging"
    name  = "Staging"
    color = "000000"
    tags  = ["terraform", ]
    # approval_settings {
    #     required                   = true
    #   can_review_own_request     = false
    #   can_apply_declined_changes = false
    #   min_num_approvals          = 2
    # }
  }

  environments {
    key   = "demo"
    name  = "Demo"
    color = "F1CF3B"
    tags  = ["terraform", ]
  }

  environments {
    key   = "qa"
    name  = "QA"
    color = "FF1493"
    tags  = ["terraform", ]
  }

  environments {
    key   = "dev"
    name  = "Dev"
    color = "4682B4"
    tags  = ["terraform", ]
  }

  default_client_side_availability {
    using_environment_id = true
    using_mobile_key     = false
  }
}

resource "launchdarkly_environment" "test_production" {
  project_key = "new-test-project"
  key         = "production"
  name        = "prod"
  color       = "ff0012"
  approval_settings {
    # required                   = true
    # can_review_own_request     = false
    # can_apply_declined_changes = false
    # min_num_approvals          = 2
  }
}
