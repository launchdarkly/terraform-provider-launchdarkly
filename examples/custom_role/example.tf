# This config provides an example of a custom role that prevents management of any flag 
# with a "terraform-managed" tag to ensure these are only managed via terraform.

provider "launchdarkly" {
  version = "~> 1.0"
}
resource "launchdarkly_custom_role" "exclude_terraform" {
  key         = "exclude-terraform"
  name        = "Exclude Terraform"
  description = "Deny access to resources with a 'terraform-managed' tag"

  policy {
    effect = "deny"
    resources = [
      "proj/*:env/:flag/*;terraform-managed"
    ]
    actions = [
      "*"
    ]
  }
}