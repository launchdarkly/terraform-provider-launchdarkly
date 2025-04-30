---
layout: "launchdarkly"
page_title: "LaunchDarkly: launchdarkly_ai_config"
description: |-
  Create and manage LaunchDarkly AI configurations.
---

# launchdarkly_ai_config

Provides a LaunchDarkly AI Config resource.

This resource allows you to create and manage AI configurations within your LaunchDarkly organization.

## Example Usage

```hcl
resource "launchdarkly_ai_config" "example" {
  project_key = "example-project"
  key         = "example-ai-config"
  name        = "Example AI Config"
  description = "This is an example AI configuration"
  tags        = ["example", "terraform"]
  
  variations {
    key         = "variation-1"
    name        = "GPT-4"
    description = "Uses GPT-4 model with standard settings"
    model       = "gpt-4"
    parameters  = {
      "temperature" = "0.7"
      "max_tokens"  = "1000"
    }
  }
  
  variations {
    key         = "variation-2"
    name        = "GPT-3.5 Turbo"
    description = "Uses GPT-3.5 Turbo model with lower temperature"
    model       = "gpt-3.5-turbo"
    parameters  = {
      "temperature" = "0.5"
      "max_tokens"  = "500"
    }
  }
}
```

## Argument Reference

- `project_key` - (Required) The key of the project to which the AI Config belongs.
- `key` - (Required) The unique key that references the AI Config.
- `name` - (Required) The human-friendly name for the AI Config.
- `description` - (Optional) The description of the AI Config's purpose.
- `tags` - (Optional) Set of tags for the AI Config.
- `variations` - (Required) List of variations for the AI Config. Each variation block supports:
  - `key` - (Required) The unique key for the variation.
  - `name` - (Required) The name of the variation.
  - `description` - (Optional) The description of the variation.
  - `model` - (Required) The AI model to use for this variation.
  - `parameters` - (Optional) Parameters for the AI model as key-value pairs.

## Import

AI Configs can be imported using the project key and AI Config key, e.g.

```
$ terraform import launchdarkly_ai_config.example example-project/example-ai-config
```
