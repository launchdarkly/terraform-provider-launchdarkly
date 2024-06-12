resource "launchdarkly_relay_proxy_configuration" "example" {
  name = "example-config"
  policy {
    actions   = ["*"]
    effect    = "allow"
    resources = ["proj/*:env/*"]
  }
}
