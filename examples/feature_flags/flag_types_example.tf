# This config provides examples of various flag variation types: boolean, string, numeric, and JSON.

# ----------------------------------------------------------------------------------- #
# BINARY FLAG
resource "launchdarkly_feature_flag" "boolean_flag" {
  project_key    = launchdarkly_project.tf_flag_examples.key
  key            = "boolean-flag"
  name           = "Bool feature flag"
  description    = "An example boolean feature flag that can be turned either on or off"
  variation_type = "boolean"
}

# ----------------------------------------------------------------------------------- #
# MULTIVARIATE FLAGS
# For multivariate flags, each variation must be described in a separate 'variations' block
# with required 'value' field and optional 'name' and 'description' fields.

# create a multivariate flag with string-value variations
resource "launchdarkly_feature_flag" "string_flag" {
  project_key = launchdarkly_project.tf_flag_examples.key
  key         = "string-flag"
  name        = "String-based feature flag"
  description = "An example of a multivariate feature flag with string variations"

  variation_type = "string"
  variations {
    name        = "A String"
    description = "one of three variations"
    value       = "string1"
  }
  variations {
    name  = "Another String"
    value = "string2"
  }
  variations {
    name  = "Yet Another String"
    value = "string3"
  }
  tags = [
    "terraform-managed"
  ]
}

# create a multivariate flag with number-value variations
# Both ints and floats are acceptable, but please note that trailing zeroes
# will be trimmed off of floats, i.e. both 123 and 123.00 will return output 123.
resource "launchdarkly_feature_flag" "number_flag" {
  project_key = launchdarkly_project.tf_flag_examples.key
  key         = "number-flag"
  name        = "Number value-based feature flag"
  description = "An example of a multivariate feature flag with numeric variations"

  variation_type = "number"
  variations {
    name  = "Big Number Variation"
    value = 123000000
  }
  variations {
    name  = "Small Number Variation"
    value = 100
  }
  variations {
    name  = "Float Variation"
    value = 123.45
  }
  tags = [
    "terraform-managed"
  ]
}

# create a multivariate flag with JSON-value variations
# Please note that since terraform evaluates all input as strings, 
# multi-line input such as jsons must use a marker like "<<EOF".
resource "launchdarkly_feature_flag" "json_flag" {
  project_key = launchdarkly_project.tf_flag_examples.key
  key         = "json-flag"
  name        = "JSON-based feature flag"
  description = "An example of a multivariate feature flag with JSON variations"

  variation_type = "json"
  variations {
    value = <<EOF
	  {"foo": "bar"}
	  EOF
  }
  variations {
    value = <<EOF
	  {
		"foo": "baz",
		"extra": {"nested": "json"}
	  }
	  EOF
  }
  tags = [
    "terraform-managed"
  ]
}