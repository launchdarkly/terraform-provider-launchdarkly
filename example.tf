resource "launchdarkly_project" "example" {
  name = "example-project"
  key  = "example-project"

  tags = [
    "terraform",
  ]
}

resource "launchdarkly_environment" "staging" {
  name                 = "Staging"
  key                  = "staging"
  color                = "ff00ff"
  secure_mode          = true
  default_track_events = false
  default_ttl          = 10

  project_key = launchdarkly_project.example.key
}

resource "launchdarkly_feature_flag" "basic" {
  project_key = launchdarkly_project.example.key
  key         = "basic-flag"
  name        = "Basic feature flag"

  variation_type = "boolean"
  variations {
    name  = "The true variation"
    value = true
  }
  variations {
    value = false
  }
}

resource "launchdarkly_feature_flag" "number" {
  project_key = launchdarkly_project.example.key
  key         = "numeric-flag"
  name        = "Number feature flag"

  variation_type = "number"
  variations {
    name  = "The first variation"
    value = 12.5
  }
  variations {
    value = 10
  }
  variations {
    value = 99
  }
}

resource "launchdarkly_feature_flag_environment" "number_env" {
  flag_id = launchdarkly_feature_flag.number.id
  env_key = launchdarkly_environment.staging.key

  targeting_enabled = true


  prerequisites {
    flag_key  = launchdarkly_feature_flag.basic.key
    variation = 0
  }
  user_targets {
    values = ["user0"]
  }
  user_targets {
    values = ["user1", "user2"]
  }
  user_targets {
    values = []
  }

  rules {
    clauses {
      attribute = "country"
      op        = "startsWith"
      values    = ["great", "de", "united"]
      negate    = false
    }
    clauses {
      attribute = "segmentMatch"
      op        = "segmentMatch"
      values    = [launchdarkly_segment.example.key]
      negate    = false
    }
    variation = 0
  }

  flag_fallthrough {
    rollout_weights = [60000, 40000, 0]
  }
}

resource "launchdarkly_feature_flag" "json" {
  project_key = launchdarkly_project.example.key
  key         = "json-flag"
  name        = "JSON feature flag"

  variation_type = "json"
  variations {
    value = <<EOF
    {"foo": "bar"}
    EOF
  }
  variations {
    value = <<EOF
    {
      "foo": "baz",
      "extra": {"nested": "json"}
    }
    EOF
  }
}

resource "launchdarkly_feature_flag" "boolean" {
  project_key    = launchdarkly_project.example.key
  key            = "boolean-flag-1"
  name           = "boolean-flag-1 name"
  variation_type = "boolean"
  description    = "this is a boolean flag by default because we omitted the variations field"
}

resource "launchdarkly_feature_flag" "multivariate" {
  project_key = launchdarkly_project.example.key
  key         = "multivariate-flag"
  name        = "multivariate-flag name"
  description = "this is a multivariate flag because we explicitly define the variations"

  variation_type = "string"
  variations {
    name        = "variation1"
    description = "a description"
    value       = "string1"
  }
  variations {
    value = "string2"
  }
  variations {
    value = "another option"
  }

  tags = [
    "this",
    "is",
    "unordered",
  ]

  custom_properties {
    key  = "some.property"
    name = "Some Property"

    value = [
      "value1",
      "value2",
      "value3",
    ]
  }
  custom_properties {
    key   = "some.property2"
    name  = "Some Property"
    value = ["very special custom property"]
  }
}

resource "launchdarkly_custom_role" "example" {
  key         = "example-role-key-1"
  name        = "example role"
  description = "This is an example role"

  policy {
    actions   = ["*"]
    effect    = "allow"
    resources = ["proj/*:env/production"]
  }
}

resource "launchdarkly_segment" "example" {
  key         = "segment-key1"
  project_key = launchdarkly_project.example.key
  env_key     = launchdarkly_environment.staging.key
  name        = "segment name"
  description = "segment description"
  tags        = ["segmentTag1", "segmentTag2"]
  included    = ["user1", "user2"]
  excluded    = ["user3", "user4"]
}

resource "launchdarkly_webhook" "example" {
  name    = "Example Webhook"
  url     = "http://webhooks.com/webhook"
  tags    = ["terraform"]
  enabled = false
}

output "api_key" {
  value = launchdarkly_environment.staging.api_key
}

output "mobile_key" {
  value = launchdarkly_environment.staging.mobile_key
}

output "client_side_id" {
  value = launchdarkly_environment.staging.client_side_id
}
