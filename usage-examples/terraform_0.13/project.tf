resource "launchdarkly_project" "tf_13_example" {
  key  = "tf-13-example"
  name = "Terraform 13 Example"

  tags = [
    "terraform"
  ]
}