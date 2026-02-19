---
page_title: "Migrating to multi-resource approval settings"
description: |-
  This guide explains how to use the approval_settings block to configure approval requirements for flags, segments, and AI configs within the same environment. Learn about the new resource_kind attribute, validation rules, and how to migrate existing configurations.
---

# Approval Settings Migration Guide

## Overview

The `approval_settings` block now supports multiple resource kinds (flags, segments, and AI configs) while maintaining full backwards compatibility with existing configurations.

## What's New

### Multiple Approval Settings Blocks

You can now specify approval settings for different resource types within the same environment:

**Important**: `service_kind` and `service_config` are **only supported for flag approval settings** (`resource_kind = "flag"`). Using these fields with `resource_kind = "segment"` or `resource_kind = "aiconfig"` will result in a validation error.

```hcl
resource "launchdarkly_environment" "example" {
  name        = "Production"
  key         = "production"
  color       = "FF0000"
  project_key = "my-project"

  # Flag approval settings
  approval_settings {
    resource_kind              = "flag"
    required                   = true
    min_num_approvals          = 2
    can_review_own_request     = false
    can_apply_declined_changes = false
  }

  # Segment approval settings
  # Note: service_kind and service_config are not supported for segment approvals
  approval_settings {
    resource_kind              = "segment"
    required                   = true
    min_num_approvals          = 1
    can_apply_declined_changes = true
  }

  # AI Config approval settings
  # Note: service_kind and service_config are not supported for aiconfig approvals
  approval_settings {
    resource_kind              = "aiconfig"
    required                   = false
    min_num_approvals          = 1
  }
}
```

### New Attribute: `resource_kind`

- **Type**: `string`
- **Optional**: Yes (defaults to `"flag"`)
- **Valid values**: `"flag"`, `"segment"`, `"aiconfig"`
- **Description**: Specifies which resource type the approval settings apply to

### Validation

- Each `resource_kind` can only be specified once per environment
- Attempting to configure duplicate `resource_kind` values will result in a validation error

## Backwards Compatibility

Existing configurations without the `resource_kind` attribute will continue to work exactly as before. The attribute defaults to `"flag"` when not specified:

```hcl
# This configuration is still valid and equivalent to resource_kind = "flag"
resource "launchdarkly_environment" "example" {
  name        = "Production"
  key         = "production"
  color       = "FF0000"
  project_key = "my-project"

  approval_settings {
    required          = true
    min_num_approvals = 1
  }
}
```

## API Mapping

### Reading from API
- `approvalSettings` (root level) → `approval_settings` with `resource_kind = "flag"`
- `resourceApprovalSettings.segment` → `approval_settings` with `resource_kind = "segment"`
- `resourceApprovalSettings.aiconfig` → `approval_settings` with `resource_kind = "aiconfig"`

### Writing to API
- `resource_kind = "flag"` → PATCH operations to `/approvalSettings/*`
- `resource_kind = "segment"` → PATCH operations to `/resourceApprovalSettings/segment/*`
- `resource_kind = "aiconfig"` → PATCH operations to `/resourceApprovalSettings/aiconfig/*`

## Migration Examples

### Before: Flag Approvals Only
```hcl
resource "launchdarkly_environment" "prod" {
  name        = "Production"
  key         = "production"
  project_key = "my-project"

  approval_settings {
    required          = true
    min_num_approvals = 2
  }
}
```

### After: Adding Segment Approvals
```hcl
resource "launchdarkly_environment" "prod" {
  name        = "Production"
  key         = "production"
  project_key = "my-project"

  # Existing flag approvals (add resource_kind for clarity, but optional)
  approval_settings {
    resource_kind     = "flag"
    required          = true
    min_num_approvals = 2
  }

  # New segment approvals
  approval_settings {
    resource_kind     = "segment"
    required          = true
    min_num_approvals = 1
  }
}
```

## Implementation Details

### Code Changes

1. **New constant**: `RESOURCE_KIND` added to `keys.go`
2. **Schema updated**: `approval_settings` block now supports `resource_kind` attribute
3. **Validation added**: `validateUniqueResourceKinds` ensures no duplicate resource kinds
4. **Patch generation**: `approvalPatchFromSettings` generates correct API paths based on resource kind
5. **Data reading**: `environmentApprovalSettingsToResourceData` handles both API structures

### Testing

Comprehensive unit tests ensure:
- Multiple resource kinds can be configured simultaneously
- Correct API paths are generated for each resource kind
- Backwards compatibility with existing configurations
- Proper handling of adding/removing resource kinds

Run tests with:
```bash
go test -v ./launchdarkly -run TestApproval
```

## Troubleshooting

### Error: "duplicate resource_kind found"

**Cause**: Multiple `approval_settings` blocks with the same `resource_kind`

**Solution**: Ensure each `approval_settings` block has a unique `resource_kind` value

```hcl
# ❌ Invalid - duplicate "flag" resource_kind
approval_settings {
  resource_kind = "flag"
  required      = true
}
approval_settings {
  resource_kind = "flag"  # ERROR: duplicate
  required      = false
}

# ✅ Valid - unique resource_kind values
approval_settings {
  resource_kind = "flag"
  required      = true
}
approval_settings {
  resource_kind = "segment"
  required      = false
}
```

### Error: "service_kind cannot be set for resource_kind 'segment'"

**Cause**: Attempting to use `service_kind` with a non-default value (anything other than `"launchdarkly"`) for segment or aiconfig approval settings

**Solution**: Remove the `service_kind` field or set it to the default value `"launchdarkly"` (or omit it entirely)

```hcl
# ❌ Invalid - service_kind not supported for segment
approval_settings {
  resource_kind = "segment"
  service_kind  = "servicenow"  # ERROR: not supported for segments
  required      = true
}

# ✅ Valid - service_kind only used with flag resource_kind
approval_settings {
  resource_kind = "flag"
  service_kind  = "servicenow"
  required      = true
}

approval_settings {
  resource_kind = "segment"
  required      = true
  # service_kind omitted or set to default "launchdarkly"
}
```

### Error: "service_config cannot be set for resource_kind 'segment'"

**Cause**: Attempting to use `service_config` for segment or aiconfig approval settings

**Solution**: Remove the `service_config` field from non-flag approval settings

```hcl
# ❌ Invalid - service_config not supported for segment
approval_settings {
  resource_kind = "segment"
  service_config = {
    template      = "template-id"
    detail_column = "justification"
  }  # ERROR: not supported for segments
  required = true
}

# ✅ Valid - service_config only used with flag resource_kind
approval_settings {
  resource_kind = "flag"
  service_kind  = "servicenow"
  service_config = {
    template      = "template-id"
    detail_column = "justification"
  }
  required = true
}

approval_settings {
  resource_kind = "segment"
  required      = true
  # service_config omitted
}
```


### Import Considerations

When you import an existing environment (using `terraform import`), Terraform will read all configured approval settings from the API and represent them as separate `approval_settings` blocks in the state, each with the appropriate `resource_kind` value.

For example, if you import an environment that has both flag and segment approvals configured:

```bash
terraform import launchdarkly_environment.example project-key/env-key
```

The resulting state will contain multiple `approval_settings` blocks. You'll need to add corresponding blocks to your Terraform configuration to match the imported state.

## Additional Resources

- [LaunchDarkly Approval Settings Documentation](https://docs.launchdarkly.com/home/feature-workflows/approvals)
- [Terraform Provider Documentation](https://registry.terraform.io/providers/launchdarkly/launchdarkly/latest/docs)
