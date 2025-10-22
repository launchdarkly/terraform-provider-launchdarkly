# Modular Views Example

This example demonstrates how to use the `view_keys` field on feature flags and segments to simplify managing view associations in a modular Terraform setup.

## Problem Statement

When using a modular Terraform structure (e.g., one file per flag in team/domain modules), managing view associations with `launchdarkly_view_links` requires:
1. Collecting flag keys as outputs from each module
2. Managing a centralized `view_links` resource at the root level
3. Refactoring existing code to support this pattern

## Solution

The `view_keys` field allows you to specify view associations directly on each flag or segment resource, enabling a truly modular structure where each resource can independently declare which views it belongs to.

## Example Structure

```
.
├── main.tf                    # Root module - creates views
├── modules/
│   ├── payments/              # Payments team module
│   │   ├── flags.tf           # Payment-related flags with view_keys
│   │   └── segments.tf        # Payment-related segments with view_keys
│   ├── frontend/              # Frontend team module
│   │   ├── flags.tf           # Frontend flags with view_keys
│   │   └── segments.tf        # Frontend segments with view_keys
│   └── shared/                # Shared features module
│       └── flags.tf           # Shared flags with view_keys
```

## Key Benefits

1. **No output collection required**: Each flag/segment declares its own view associations
2. **Easy to add/modify**: Simply update the `view_keys` in the flag definition
3. **Module independence**: Teams can manage their flags without coordinating centrally
4. **Backwards compatible**: Can coexist with `launchdarkly_view_links` (but avoid managing the same resource both ways)

## Usage Notes

- The `linked_views` computed field shows ALL views a resource is linked to, regardless of how the association was created
- If you use both `view_keys` and `launchdarkly_view_links` to manage the same resource, they may conflict
- Views must exist before you can link resources to them

