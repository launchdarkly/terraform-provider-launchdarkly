resource "launchdarkly_project" "example" {
  key  = "example-project"
  name = "metrics example project"
  environments {
    name  = "example environment"
    key   = "example-env"
    color = "010101"
  }
}

resource "launchdarkly_metric" "pageview_example" {
  project_key    = launchdarkly_project.example.key
  key            = "pageview-metric"
  name           = "Pageview Metric"
  description    = "example pageview metric"
  kind           = "pageview"
  is_active      = false
  tags           = [
    "example",
  ]
  randomization_units = [
    "user",
    "request",
  ]
  urls {
    kind = "substring"
    substring = "foo"
  }
  urls {
    kind = "regex"
    pattern = "`foo`gm"
  }
}

resource "launchdarkly_metric" "click_example" {
  project_key    = launchdarkly_project.example.key
  key            = "click-metric"
  name           = "click Metric"
  description    = "example click metric"
  kind           = "click"
  selector       = ".foo"
  tags           = [
    "example",
  ]
  randomization_units = [
    "user",
    "request",
  ]
  urls {
    kind = "exact"
    url = "https://example.com/example/"
  }
}

resource "launchdarkly_metric" "custom_example" {
  project_key    = launchdarkly_project.example.key
  key            = "custom-metric"
  name           = "custom Metric"
  description    = "example custom metric"
  kind           = "custom"
  event_key      = "foo"
  tags           = [
    "example",
  ]
  randomization_units = [
    "user",
    "request",
  ]
}

resource "launchdarkly_metric" "numeric_example" {
  project_key      = launchdarkly_project.example.key
  key              = "numeric-metric"
  name             = "numeric Metric"
  description      = "example numeric metric"
  kind             = "custom"
  is_numeric       = true
  unit             = "bar"
  success_criteria = "HigherThanBaseline"
  event_key        = "foo"
  tags             = [
    "example",
  ]
  randomization_units = [
    "user",
    "request",
  ]
}
