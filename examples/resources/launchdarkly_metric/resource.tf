resource "launchdarkly_metric" "example" {
  project_key = launchdarkly_project.example.key
  key         = "example-metric"
  name        = "Example Metric"
  description = "Metric description."
  kind        = "pageview"
  tags        = ["example"]
  urls {
    kind      = "substring"
    substring = "foo"
  }
}
