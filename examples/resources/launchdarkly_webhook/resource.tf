resource "launchdarkly_webhook" "example" {
  url  = "http://webhooks.com/webhook"
  name = "Example Webhook"
  tags = ["terraform"]
  on   = true

  statements {
    actions   = ["*"]
    effect    = "allow"
    resources = ["proj/*:env/production:flag/*"]
  }
  statements {
    actions   = ["*"]
    effect    = "allow"
    resources = ["proj/test:env/production:segment/*"]
  }
}
