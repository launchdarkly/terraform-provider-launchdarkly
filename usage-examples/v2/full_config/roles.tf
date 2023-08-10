# create a role that grants all permissions on this project / 
# resources tagged with "terraform"
resource "launchdarkly_custom_role" "terraform" {
  key         = "terraform-v2"
  name        = "Terraform"
  description = "allow access to Terraform Full Example Configuration Project"

  policy_statements {
    effect = "allow"
    resources = [
      "proj/tf_full_config:env/*:flag/*;terraform"
    ]
    actions = [
      "*"
    ]
  }
}

# configure team members and associated roles
resource "launchdarkly_team_member" "jane_doe" {
  email      = "jane.doe@ourcompany.com"
  first_name = "Jane"
  last_name  = "Doe"
  custom_roles = [
    launchdarkly_custom_role.terraform.key
  ]
}

resource "launchdarkly_team_member" "john_doe" {
  email      = "john.doe@ourcompany.com"
  first_name = "John"
  last_name  = "Doe"
  custom_roles = [
    launchdarkly_custom_role.terraform.key
  ]
}