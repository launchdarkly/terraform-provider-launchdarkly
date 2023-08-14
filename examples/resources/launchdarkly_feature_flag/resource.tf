resource "launchdarkly_feature_flag" "building_materials" {
  project_key = launchdarkly_project.example.key
  key         = "building-materials"
  name        = "Building materials"
  description = "this is a multivariate flag with string variations."

  variation_type = "string"
  variations {
    value       = "straw"
    name        = "Straw"
    description = "Watch out for wind."
  }
  variations {
    value       = "sticks"
    name        = "Sticks"
    description = "Sturdier than straw"
  }
  variations {
    value       = "bricks"
    name        = "Bricks"
    description = "The strongest variation"
  }

  client_side_availability {
    using_environment_id = false
    using_mobile_key     = true
  }

  defaults {
    on_variation  = 2
    off_variation = 0
  }

  tags = [
    "example",
    "terraform",
    "multivariate",
    "building-materials",
  ]
}

resource "launchdarkly_feature_flag" "json_example" {
  project_key = "example-project"
  key         = "json-example"
  name        = "JSON example flag"

  variation_type = "json"
  variations {
    name  = "Single foo"
    value = jsonencode({ "foo" : "bar" })
  }
  variations {
    name  = "Multiple foos"
    value = jsonencode({ "foos" : ["bar1", "bar2"] })
  }

  defaults {
    on_variation  = 1
    off_variation = 0
  }
}
