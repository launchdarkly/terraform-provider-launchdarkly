# This config provides an example for configuring a webhook that posts to a public pastebin service

provider "launchdarkly" {
  version = "~> 1.2.0"
}

resource "launchdarkly_webhook" "tf_example_webook" {
  name = "tf-example-webhook"
  url  = "https://enrl3l3jnmnwh.x.pipedream.net"
  tags = [
    "terraform-managed"
  ]
  enabled = true

  policy_statements {
    actions   = ["*"]
    effect    = "allow"
    resources = ["proj/*:env/*:flag/*;terraform-managed"]
  }
}