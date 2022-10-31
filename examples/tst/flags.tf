# resource "launchdarkly_feature_flag" "boolean_flag" {
#   project_key    = "default"
#   key            = "a-customer-test-flag"
#   name           = "A customer test flag"
#   description    = "An example boolean feature flag that can be turned either on or off"
#   variation_type = "boolean"

#   defaults {
#     on_variation  = 1
#     off_variation = 1
#   }

#   client_side_availability {
#     using_environment_id = true
#   }
# }

# resource "launchdarkly_feature_flag" "completeTheLook" {
#   project_key    = launchdarkly_project.a_new_test_proj.key
#   key            = "completeTheLook"
#   name           = "Complete The Look"
#   description    = "Dunelm Web Complete The Look Flag"
#   variation_type = "boolean"

#   variations {
#     value = true
#   }

#   variations {
#     value = false
#   }

#   tags = [
#     "terraform"
#   ]

# }

# resource "launchdarkly_feature_flag_environment" "completeTheLook_dundev" {
#   flag_id       = launchdarkly_feature_flag.completeTheLook.id
#   env_key       = "production"
#   off_variation = 0
#   on            = true

#   fallthrough {
#     variation = 0
#   }
#   #   lifecycle {
#   #     ignore_changes = [
#   #         fallthrough
#   #     ]
#   #   }
# }
