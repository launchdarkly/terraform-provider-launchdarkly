## Example: full LD config

### Introduction
The LaunchDarkly Terraform provider allows you to configure a full suite of LaunchDarkly features using Terraform.

The sample configuration in this directory provides an example of how to organize a full project configuration that takes advantage of the following LaunchDarkly resource types:
- [launchdarkly_project](https://www.terraform.io/docs/providers/launchdarkly/r/project.html) in [project.tf](./project.tf)
- [launchdarkly_environment](https://www.terraform.io/docs/providers/launchdarkly/r/environment.html) in [env-staging.tf](./env-staging.tf) and [env-dev.tf](./env-dev.tf)
- [launchdarkly_custom_role](https://www.terraform.io/docs/providers/launchdarkly/r/custom_role.html) and [launchdarkly_team_member](https://www.terraform.io/docs/providers/launchdarkly/r/team_member.html) in [roles.tf](./roles.tf)
- [launchdarkly_feature_flag](https://www.terraform.io/docs/providers/launchdarkly/r/feature_flag.html) and [launchdarkly_feature_flag_environment](https://www.terraform.io/docs/providers/launchdarkly/r/feature_flag_environment.html) in [flags.tf](./flags.tf) and [env-staging.tf](./env-staging.tf), respectively
- [launchdarkly_segment](https://www.terraform.io/docs/providers/launchdarkly/r/segment.html) in [env-dev.tf](./env-dev.tf)

Resources not included in this example are:
- [launchdarkly_webhook](https://www.terraform.io/docs/providers/launchdarkly/r/webhook.html), an example of which can be found in the [webhook](../webhook) directory
- [launchdarkly_destination](https://www.terraform.io/docs/providers/launchdarkly/r/destination.html)


### Important notes
- If you want to use Terraform to configure existing LaunchDarkly resources, you will first need to import them from the CL using `terraform import`. For details on the precise syntax, select the relevant resource [from this page](https://www.terraform.io/docs/providers/launchdarkly/index.html) and scroll to the bottom. For example, if you want to configure the automatically-created production environment after creating the project, run `terraform import launchdarkly_environment.production tf-full-config/production` before running your env config files.


### Run
Init your working directory from the CL with `terraform init` and then apply the changes with `terraform apply`. If you are creating it for the first time, the output should end with something like `Plan: 11 to add, 0 to change, 0 to destroy.`