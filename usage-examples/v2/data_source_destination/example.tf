resource "launchdarkly_project" "data_source_destination_project" {
  key  = "tf-dest-project"
  name = "Terraform Data Source Destination Project"

  tags = [
    "terraform"
  ]
}

resource "launchdarkly_destination" "segment" {
	project_key = launchdarkly_project.data_source_destination_project.key
	env_key = "production"
	name    = "Example Segment Destination"
	kind    = "segment"
	config  = {
		write_key = "this-is-a-secret"
	}
	on = false
	tags = [ "terraform" ]
}

