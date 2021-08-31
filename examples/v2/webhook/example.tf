# This config provides an example for configuring a webhook that posts to a public pastebin service

resource "launchdarkly_webhook" "tf_example_webook" {
  name = "tf-example-webhook"
  url  = "https://enrl3l3jnmnwh.x.pipedream.net"
  tags = [
    "terraform-managed"
  ]
  on = true

  statements {
    actions   = ["*"]
    effect    = "allow"
    resources = ["proj/*:env/*:flag/*;terraform-managed"]
  }
}
