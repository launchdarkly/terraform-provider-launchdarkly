package launchdarkly

// approvals_framework.go is the terraform-plugin-framework analogue of
// approvals_helper.go (the schema + conversion helpers for
// approval_settings blocks). Used by environment + project data sources
// and (later) by the resource migrations of those types.
//
// Block-style nesting preserved: `approval_settings { ... }` stays a
// block, not a nested attribute, per the CLAUDE.md convention.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// frameworkApprovalSettingsObjectAttrTypes is the attribute-type map
// every approval_settings list-nested-block element conforms to. Single-
// element list shape preserves the SDKv2 TypeList{MaxItems:1} pattern.
var frameworkApprovalSettingsObjectAttrTypes = map[string]attr.Type{
	REQUIRED:                    types.BoolType,
	CAN_REVIEW_OWN_REQUEST:      types.BoolType,
	MIN_NUM_APPROVALS:           types.Int64Type,
	CAN_APPLY_DECLINED_CHANGES:  types.BoolType,
	REQUIRED_APPROVAL_TAGS:      types.ListType{ElemType: types.StringType},
	SERVICE_KIND:                types.StringType,
	SERVICE_CONFIG:              types.MapType{ElemType: types.StringType},
	AUTO_APPLY_APPROVED_CHANGES: types.BoolType,
}

// frameworkApprovalSettingsDataSourceBlock returns a ListNestedBlock
// schema for use in datasource.Schema. All inner attrs are Computed.
func frameworkApprovalSettingsDataSourceBlock() dsschema.ListNestedBlock {
	return dsschema.ListNestedBlock{
		Description: "Approval settings for this environment / project.",
		NestedObject: dsschema.NestedBlockObject{
			Attributes: map[string]dsschema.Attribute{
				REQUIRED: dsschema.BoolAttribute{
					Computed:    true,
					Description: "Whether changes require approval.",
				},
				CAN_REVIEW_OWN_REQUEST: dsschema.BoolAttribute{
					Computed:    true,
					Description: "Whether requesters can approve their own requests.",
				},
				MIN_NUM_APPROVALS: dsschema.Int64Attribute{
					Computed:    true,
					Description: "Minimum approvers required (1-5).",
				},
				CAN_APPLY_DECLINED_CHANGES: dsschema.BoolAttribute{
					Computed:    true,
					Description: "Whether changes can be applied with the minimum number of approvals despite declines.",
				},
				REQUIRED_APPROVAL_TAGS: dsschema.ListAttribute{
					Computed:    true,
					ElementType: types.StringType,
					Description: "Flag tags requiring approval (only one of required / required_approval_tags is set).",
				},
				SERVICE_KIND: dsschema.StringAttribute{
					Computed:    true,
					Description: "Approval service (e.g. servicenow, launchdarkly).",
				},
				SERVICE_CONFIG: dsschema.MapAttribute{
					Computed:    true,
					ElementType: types.StringType,
					Description: "Service-specific approval config.",
				},
				AUTO_APPLY_APPROVED_CHANGES: dsschema.BoolAttribute{
					Computed:    true,
					Description: "Whether to auto-apply changes once all approvers have approved.",
				},
			},
		},
	}
}

// frameworkApprovalSettingsValue converts an LD-API ApprovalSettings
// into a single-element types.List (matching the SDKv2 TypeList:1 shape
// that approvalSettingsToResourceData produced). Empty / unset settings
// produce an empty list, not null, so wire shape is stable.
func frameworkApprovalSettingsValue(ctx context.Context, settings *ldapi.ApprovalSettings) (basetypes.ListValue, diag.Diagnostics) {
	objectType := types.ObjectType{AttrTypes: frameworkApprovalSettingsObjectAttrTypes}
	// Framework blocks can't be Computed at the block level (SDKv2 had
	// Optional+Computed on the TypeList), so emit empty for LD's
	// zero-default struct to keep omitted-config plans empty.
	if settings == nil || isZeroApprovalSettings(settings) {
		return types.ListValue(objectType, []attr.Value{})
	}

	requiredTagsList, diags := listFromStringSlice(ctx, settings.RequiredApprovalTags)

	serviceConfig := make(map[string]string, len(settings.ServiceConfig))
	for k, v := range settings.ServiceConfig {
		if s, ok := v.(string); ok {
			serviceConfig[k] = s
		} else {
			// LD's ServiceConfig is map[string]interface{}; coerce to
			// the framework Map<String> by string-formatting non-string
			// values. Reviewers should treat unexpected shapes as a bug
			// on the API side.
			serviceConfig[k] = ""
		}
	}
	serviceConfigVal, d := types.MapValueFrom(ctx, types.StringType, serviceConfig)
	diags.Append(d...)

	// Mirror schema Default(false) so plan-vs-apply matches when LD
	// returns nil here.
	autoApply := types.BoolValue(false)
	if settings.AutoApplyApprovedChanges != nil {
		autoApply = types.BoolValue(*settings.AutoApplyApprovedChanges)
	}

	obj, d := types.ObjectValue(frameworkApprovalSettingsObjectAttrTypes, map[string]attr.Value{
		REQUIRED:                    types.BoolValue(settings.Required),
		CAN_REVIEW_OWN_REQUEST:      types.BoolValue(settings.CanReviewOwnRequest),
		MIN_NUM_APPROVALS:           types.Int64Value(int64(settings.MinNumApprovals)),
		CAN_APPLY_DECLINED_CHANGES:  types.BoolValue(settings.CanApplyDeclinedChanges),
		REQUIRED_APPROVAL_TAGS:      requiredTagsList,
		SERVICE_KIND:                types.StringValue(settings.ServiceKind),
		SERVICE_CONFIG:              serviceConfigVal,
		AUTO_APPLY_APPROVED_CHANGES: autoApply,
	})
	diags.Append(d...)

	list, d := types.ListValue(objectType, []attr.Value{obj})
	diags.Append(d...)
	return list, diags
}

// frameworkApprovalSettingsResourceBlock returns the resource-side
// ListNestedBlock schema for approval_settings. Shared between
// project's nested-environments block, segment, FFE, and the standalone
// environment resource going forward.
func frameworkApprovalSettingsResourceBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Approval settings (single element).",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				REQUIRED:                    schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
				CAN_REVIEW_OWN_REQUEST:      schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
				MIN_NUM_APPROVALS:           schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(1)},
				CAN_APPLY_DECLINED_CHANGES:  schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
				REQUIRED_APPROVAL_TAGS:      schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType},
				SERVICE_KIND:                schema.StringAttribute{Optional: true, Computed: true},
				SERVICE_CONFIG:              schema.MapAttribute{Optional: true, Computed: true, ElementType: types.StringType},
				AUTO_APPLY_APPROVED_CHANGES: schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
			},
		},
	}
}

// approvalSettingsModel matches frameworkApprovalSettingsObjectAttrTypes.
type approvalSettingsModel struct {
	Required                 types.Bool   `tfsdk:"required"`
	CanReviewOwnRequest      types.Bool   `tfsdk:"can_review_own_request"`
	MinNumApprovals          types.Int64  `tfsdk:"min_num_approvals"`
	CanApplyDeclinedChanges  types.Bool   `tfsdk:"can_apply_declined_changes"`
	RequiredApprovalTags     types.List   `tfsdk:"required_approval_tags"`
	ServiceKind              types.String `tfsdk:"service_kind"`
	ServiceConfig            types.Map    `tfsdk:"service_config"`
	AutoApplyApprovedChanges types.Bool   `tfsdk:"auto_apply_approved_changes"`
}

// approvalPatchesFromModels mirrors approvalPatchFromSettings in
// approvals_helper.go but operates on framework List values directly.
// Returns the patch operations needed to apply the difference between
// the planned and prior state of an approval_settings block.
func approvalPatchesFromModels(ctx context.Context, planList, stateList types.List) ([]ldapi.PatchOperation, diag.Diagnostics) {
	var diags diag.Diagnostics
	planEmpty := planList.IsNull() || planList.IsUnknown() || len(planList.Elements()) == 0
	stateEmpty := stateList.IsNull() || stateList.IsUnknown() || len(stateList.Elements()) == 0
	if planEmpty && stateEmpty {
		return nil, diags
	}
	if planEmpty {
		// Remove gates so LD returns to default approval state.
		return []ldapi.PatchOperation{
			patchRemove("/approvalSettings/required"),
			patchRemove("/approvalSettings/requiredApprovalTags"),
		}, diags
	}
	var models []approvalSettingsModel
	d := planList.ElementsAs(ctx, &models, false)
	diags.Append(d...)
	if diags.HasError() || len(models) == 0 {
		return nil, diags
	}
	m := models[0]
	requiredApprovalTags, d := stringSliceFromList(ctx, m.RequiredApprovalTags)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	if m.Required.ValueBool() && len(requiredApprovalTags) > 0 {
		diags.AddError("invalid approval_settings config", "required and required_approval_tags cannot be set simultaneously")
		return nil, diags
	}
	serviceKind := m.ServiceKind.ValueString()
	autoApply := m.AutoApplyApprovedChanges.ValueBool()
	if serviceKind == "launchdarkly" && autoApply {
		diags.AddError("invalid approval_settings config", "auto_apply_approved_changes cannot be set to true for service_kind of launchdarkly")
		return nil, diags
	}
	serviceConfig := map[string]interface{}{}
	raw, d := mapStringFromAttr(ctx, m.ServiceConfig)
	diags.Append(d...)
	for k, v := range raw {
		serviceConfig[k] = v
	}
	return []ldapi.PatchOperation{
		patchReplace("/approvalSettings/required", m.Required.ValueBool()),
		patchReplace("/approvalSettings/canReviewOwnRequest", m.CanReviewOwnRequest.ValueBool()),
		patchReplace("/approvalSettings/minNumApprovals", m.MinNumApprovals.ValueInt64()),
		patchReplace("/approvalSettings/canApplyDeclinedChanges", m.CanApplyDeclinedChanges.ValueBool()),
		patchReplace("/approvalSettings/requiredApprovalTags", requiredApprovalTags),
		patchReplace("/approvalSettings/serviceKind", serviceKind),
		patchReplace("/approvalSettings/serviceConfig", serviceConfig),
		patchReplace("/approvalSettings/autoApplyApprovedChanges", autoApply),
	}, diags
}

// isZeroApprovalSettings reports whether LD's approval-settings doc is
// effectively unconfigured. LD returns a struct (with API defaults
// like minNumApprovals=1) for envs without approvals; we treat the
// doc as absent when no approval gate is active and no service
// integration is wired up.
func isZeroApprovalSettings(s *ldapi.ApprovalSettings) bool {
	if s == nil {
		return true
	}
	if s.Required {
		return false
	}
	if len(s.RequiredApprovalTags) > 0 {
		return false
	}
	if s.ServiceKind != "" && s.ServiceKind != "launchdarkly" {
		return false
	}
	if len(s.ServiceConfig) > 0 {
		return false
	}
	if s.AutoApplyApprovedChanges != nil && *s.AutoApplyApprovedChanges {
		return false
	}
	return true
}
