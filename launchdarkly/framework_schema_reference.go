package launchdarkly

// framework_schema_reference.go is a documentation-only file.
//
// It exists to anchor the convention that resources migrated from SDKv2
// to terraform-plugin-framework must preserve the user-facing block
// syntax (e.g. `environments { approval_settings { ... } }`) rather
// than re-shaping the schema with nested attributes
// (`environments = [{ approval_settings = { ... } }]`). The reasoning,
// in short:
//
//   - The migration ships in v2.x minor releases as a non-breaking
//     internal SDK swap (see .claude/MIGRATION_PLAN_NON_BREAKING.md).
//   - Existing HCL configs must continue to apply unchanged. Block ->
//     nested attribute is a config-rewrite for users.
//   - terraform-plugin-framework's schema.Blocks (in
//     schema.Schema.Blocks) is the deliberate parity escape hatch for
//     exactly this case.
//
// The worked example below mirrors how
// launchdarkly_project's environments + approval_settings nesting would
// look in framework. Compare against the SDKv2 source in
// resource_launchdarkly_project.go for the 1:1 mapping.
//
// This file declares no symbols; it carries example code in comments
// that the migration owner can copy-paste-and-adapt when porting a
// resource.

/*

Example: project.environments[*].approval_settings as a framework
schema.Blocks tree.

  import (
      "github.com/hashicorp/terraform-plugin-framework/resource/schema"
      "github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
      "github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
      "github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
      "github.com/hashicorp/terraform-plugin-framework/types"
  )

  func (r *ProjectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
      resp.Schema = schema.Schema{
          Attributes: map[string]schema.Attribute{
              "key": schema.StringAttribute{
                  Required:    true,
                  Description: "The project's unique key.",
                  Validators:  []validator.String{keyAndLengthValidator(1, 100)},
                  PlanModifiers: []planmodifier.String{
                      stringplanmodifier.RequiresReplace(),
                  },
              },
              "name": schema.StringAttribute{
                  Required:    true,
                  Description: "A human-readable name for the project.",
              },
              "tags": schema.SetAttribute{
                  Optional:    true,
                  Computed:    true,
                  ElementType: types.StringType,
                  Validators: []validator.Set{
                      // per-element tag validator
                  },
              },
              "include_in_snippet": schema.BoolAttribute{
                  Optional:    true,
                  Computed:    true,
                  Description: "Deprecated: use client_side_availability instead.",
                  DeprecationMessage: "Use client_side_availability instead. " +
                      "This attribute remains for backwards-compat; configs that set it continue to apply.",
              },
          },
          Blocks: map[string]schema.Block{
              // environments { ... } stays a block — block syntax
              // preservation is the whole point of this file. The
              // SDKv2 TypeList becomes ListNestedBlock; SingleNested
              // wrapping for approval_settings inside it maps to
              // SingleNestedBlock.
              "environments": schema.ListNestedBlock{
                  NestedObject: schema.NestedBlockObject{
                      Attributes: map[string]schema.Attribute{
                          "key":   schema.StringAttribute{Required: true},
                          "name":  schema.StringAttribute{Required: true},
                          "color": schema.StringAttribute{Required: true},
                      },
                      Blocks: map[string]schema.Block{
                          "approval_settings": schema.SingleNestedBlock{
                              Attributes: map[string]schema.Attribute{
                                  "required": schema.BoolAttribute{
                                      Optional: true,
                                      Computed: true,
                                      // matches SDKv2 Default: false
                                      Default: booldefault.StaticBool(false),
                                  },
                                  "service_kind": schema.StringAttribute{
                                      Optional: true,
                                      Computed: true,
                                  },
                              },
                          },
                      },
                  },
              },
          },
      }
  }

Mapping cheatsheet, SDKv2 -> framework:

  TypeList Elem=Resource{} MaxItems=N       -> ListNestedBlock
  TypeSet  Elem=Resource{}                  -> SetNestedBlock
  TypeList MaxItems=1 Elem=Resource{}       -> SingleNestedBlock
  TypeList MaxItems=N Elem=&Schema{string}  -> ListAttribute{ElementType:String}
  TypeSet  Elem=&Schema{string}             -> SetAttribute{ElementType:String}
  Required: true                            -> Required: true        (verbatim)
  Optional: true                            -> Optional: true
  Computed: true                            -> Computed: true
  ForceNew: true                            -> stringplanmodifier.RequiresReplace()
  Default: X                                -> <type>default.StaticX(X)
  ValidateDiagFunc: validateKey()           -> Validators: []validator.String{keyValidator()}
  ConflictsWith / ExactlyOneOf              -> ConfigValidators (resource.ConfigValidator)
  Deprecated: "msg"                         -> DeprecationMessage: "msg"
  CustomizeDiff                             -> ModifyPlan method on the resource
  SchemaVersion + StateUpgraders            -> UpgradeState method returning UpgradeStateMap

Anti-patterns to avoid:

  - Don't convert blocks to nested attributes when migrating. It's a
    breaking config change.
  - Don't bump SchemaVersion in a migration PR. The wire format
    stays put.
  - Don't drop a Deprecated attribute "while we're in there". Use the
    DeprecationMessage carry-forward; deletion is a future major-version
    move (out of scope for v2.x).

*/
