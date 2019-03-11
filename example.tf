resource "launchdarkly_project" "exampleproject1" {
  name = "example-project"
  key = "example-project"
  tags = [
    "terraform"]
  environments = [
    {
      name = "defined in project post"
      key = "projDefinedEnv"
      color = "0000f0"
      default_ttl = 100.0
      secure_mode = true
      default_track_events = false
    }
  ]
}

resource "launchdarkly_environment" "staging" {
  name = "Staging"
  key = "staging"
  color = "ff00ff"
  secure_mode = true
  default_track_events = false
  tags = [
    "tags",
    "are",
    "not",
    "ordered",
  ],
  default_ttl = 100.0

  project_key = "${launchdarkly_project.exampleproject1.key}"
}

resource "launchdarkly_feature_flag" "boolean-flag-1" {
  project_key = "${launchdarkly_project.exampleproject1.key}"
  key = "boolean-flag-1"
  name = "boolean-flag-1 name"
  description = "this is a boolean flag by default because we omitted the variations field"
}


resource "launchdarkly_feature_flag" "multivariate-flag-2" {
  project_key = "${launchdarkly_project.exampleproject1.key}"
  key = "multivariate-flag-2"
  name = "multivariate-flag-2 name"
  description = "this is a multivariate flag because we explicitly define the variations"
  variation_type = "string"
  variations = [
    {
      name = "variation1"
      description = "a description"
      value = "string1"
    },
    {
      value = "string2"
    },
    {
      value = "another option"
    },
  ]
  tags = [
    "this",
    "is",
    "unordered"
  ]
  custom_properties = [
    {
      key = "some.property"
      name = "Some Property"
      value = [
        "value1",
        "value2",
        "value3"]
    },
    {
      key = "some.property2"
      name = "Some Property"
      value = ["very special custom property"]
    }]
}


resource "launchdarkly_feature_flag" "multivariate-flag-3" {
  project_key = "${launchdarkly_project.exampleproject1.key}"
  key = "multivariate-flag-3"
  name = "multivariate-flag-3 with json variations"
  description = "this is a multivariate flag because we explicitly define the variations"
  variation_type = "json"
  variations = [
    {
      name = "variation1"
      description = "a description"
      value = "{\"key1\": 3}"
    },
    {
      value = "{\"key1\": \"sdfsdf\"}"
    },
  ]
}

output "api_key" {
  value = "${launchdarkly_environment.staging.api_key}"
}

output "mobile_key" {
  value = "${launchdarkly_environment.staging.mobile_key}"
}