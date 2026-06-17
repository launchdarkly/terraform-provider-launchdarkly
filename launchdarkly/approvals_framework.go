package launchdarkly

// approvals_framework.go provides the shared approval_settings schema
// + conversion helpers. Used by the environment, project, segment, and
// feature_flag_environment resources and data sources.
//
// HCL surface: `approval_settings = [{ ... }]` — a single-element list.

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
// every approval_settings list element conforms to.
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

// frameworkApprovalSettingsDataSourceAttribute returns a ListNestedAttribute
// schema for use in datasource.Schema. All inner attrs are Computed.
func frameworkApprovalSettingsDataSourceAttribute() dsschema.ListNestedAttribute {
	return dsschema.ListNestedAttribute{
		Computed:    true,
		Description: "Approval settings for this environment / project.",
		NestedObject: dsschema.NestedAttributeObject{
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
// into a single-element types.List, mirroring the prior state's
// attribute presence. The `prior` argument carries the plan's view of
// the attribute (during Create/Update) or the previous state (during
// Refresh) so the read can emit a null list when the user did not
// declare the attribute and a populated single-element list when they
// did, even when both branches resolve to LD-API "default" approval
// values. Returning null (not an empty list) is important for plan
// parity: the framework treats `attr = null` and `attr = []`
// differently in the plan/apply consistency check, especially when
// the parent object contains sensitive fields.
func frameworkApprovalSettingsValue(ctx context.Context, settings *ldapi.ApprovalSettings, prior basetypes.ListValue) (basetypes.ListValue, diag.Diagnostics) {
	objectType := types.ObjectType{AttrTypes: frameworkApprovalSettingsObjectAttrTypes}
	priorEmpty := prior.IsNull() || prior.IsUnknown() || len(prior.Elements()) == 0
	if settings == nil || priorEmpty {
		return types.ListNull(objectType), nil
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

// frameworkApprovalSettingsResourceAttribute returns the resource-side
// ListNestedAttribute schema for approval_settings. Shared between
// project's nested-environments attribute, segment, FFE, and the
// standalone environment resource.
func frameworkApprovalSettingsResourceAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Optional: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				REQUIRED: schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "Set to `true` for changes to flags in this environment to require approval. You may only set `required` to true if `required_approval_tags` is not set and vice versa. Defaults to `false`.",
				},
				CAN_REVIEW_OWN_REQUEST: schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "Set to `true` if requesters can approve or decline their own request. They may always comment. Defaults to `false`.",
				},
				MIN_NUM_APPROVALS: schema.Int64Attribute{
					Optional:    true,
					Computed:    true,
					Default:     int64default.StaticInt64(1),
					Description: "The number of approvals required before an approval request can be applied. This number must be between 1 and 5. Defaults to 1.",
				},
				CAN_APPLY_DECLINED_CHANGES: schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(true),
					Description: "Set to `true` if changes can be applied as long as the `min_num_approvals` is met, regardless of whether any reviewers have declined a request. Defaults to `true`.",
				},
				REQUIRED_APPROVAL_TAGS: schema.ListAttribute{
					Optional:    true,
					Computed:    true,
					ElementType: types.StringType,
					Description: "An array of tags used to specify which flags with those tags require approval. You may only set `required_approval_tags` if `required` is set to `false` and vice versa.",
				},
				SERVICE_KIND: schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "The kind of service associated with this approval. This determines which platform is used for requesting approval. Valid values are `servicenow`, `launchdarkly`. If you use a value other than `launchdarkly`, you must have already configured the integration in the LaunchDarkly UI or your apply will fail.",
				},
				SERVICE_CONFIG: schema.MapAttribute{
					Optional:    true,
					Computed:    true,
					ElementType: types.StringType,
					Description: "The configuration for the service associated with this approval. This is specific to each approval service. For a `service_kind` of `servicenow`, the following fields apply:\n\n\t - `template` (String) The sys_id of the Standard Change Request Template in ServiceNow that LaunchDarkly will use when creating the change request.\n\t - `detail_column` (String) The name of the ServiceNow Change Request column LaunchDarkly uses to populate detailed approval request information. This is most commonly \"justification\".",
				},
				AUTO_APPLY_APPROVED_CHANGES: schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "Automatically apply changes that have been approved by all reviewers. This field is only applicable for approval service kinds other than `launchdarkly`.",
				},
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

// approvalPatchesFromModels returns the patch operations needed to apply
// the difference between the planned and prior state of an
// approval_settings block.
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

// frameworkApprovalSettingsDataSourceValue is the data-source variant
// that always emits the populated block when LD returns approval
// settings, since data source attrs are Computed-only and don't go
// through the resource block-presence consistency check.
func frameworkApprovalSettingsDataSourceValue(ctx context.Context, settings *ldapi.ApprovalSettings) (basetypes.ListValue, diag.Diagnostics) {
	// Synthetic "non-empty prior" so the helper emits populated.
	objectType := types.ObjectType{AttrTypes: frameworkApprovalSettingsObjectAttrTypes}
	if settings == nil {
		return types.ListValue(objectType, []attr.Value{})
	}
	priorObj, _ := types.ObjectValue(frameworkApprovalSettingsObjectAttrTypes, map[string]attr.Value{
		REQUIRED:                    types.BoolValue(false),
		CAN_REVIEW_OWN_REQUEST:      types.BoolValue(false),
		MIN_NUM_APPROVALS:           types.Int64Value(0),
		CAN_APPLY_DECLINED_CHANGES:  types.BoolValue(false),
		REQUIRED_APPROVAL_TAGS:      types.ListNull(types.StringType),
		SERVICE_KIND:                types.StringValue(""),
		SERVICE_CONFIG:              types.MapNull(types.StringType),
		AUTO_APPLY_APPROVED_CHANGES: types.BoolValue(false),
	})
	priorList, _ := types.ListValue(objectType, []attr.Value{priorObj})
	return frameworkApprovalSettingsValue(ctx, settings, priorList)
}

// isZeroApprovalSettings reports whether LD's approval-settings doc is
// effectively unconfigured (no approval gate active and no service
// integration wired up). Used on the Import-equivalent path inside
// project envs where there's no prior state to anchor attribute
// presence.
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

// ----------------------------------------------------------------------
// segment_approval_settings
//
// Segment approval settings live on a separate, beta API surface than
// flag approval settings: GET/PATCH /api/v2/approval-requests/projects/
// {projectKey}/settings (LD-API-Version: beta), with a resourceKind of
// "segment" and an environmentKey in the body. They share the same
// object shape (frameworkApprovalSettingsObjectAttrTypes) and conversion
// helpers as flag approval_settings; only the transport differs.
// ----------------------------------------------------------------------

// segmentResourceKind is the value LD's approval-request settings API uses
// to scope settings to segment changes (as opposed to "flag").
const segmentResourceKind = "segment"

// segmentApprovalSettingsWarning is surfaced on the resource attribute so
// users understand the sequencing footgun (issue #370): enabling segment
// approvals while managing segments with Terraform makes every subsequent
// segment apply require manual approval before it can be applied.
const segmentApprovalSettingsWarning = "\n\n~> **Warning:** Enabling segment approvals (`required = true`) while you manage `launchdarkly_segment` resources in Terraform will cause every subsequent segment change to require manual approval before it can be applied, so your applies will not complete until a reviewer approves them. This is a known limitation tracked in [issue #370](https://github.com/launchdarkly/terraform-provider-launchdarkly/issues/370). Only enable this if you are prepared to approve segment changes out of band."

// frameworkSegmentApprovalSettingsResourceAttribute returns the
// resource-side ListNestedAttribute schema for segment_approval_settings.
// It mirrors frameworkApprovalSettingsResourceAttribute but with
// segment-specific wording and the #370 warning.
func frameworkSegmentApprovalSettingsResourceAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Optional:    true,
		Description: "Configure approval settings for segment changes in this environment. This is configured via LaunchDarkly's beta approvals API, separate from flag `approval_settings`." + segmentApprovalSettingsWarning,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				REQUIRED: schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "Set to `true` for changes to segments in this environment to require approval. You may only set `required` to true if `required_approval_tags` is not set and vice versa. Defaults to `false`.",
				},
				CAN_REVIEW_OWN_REQUEST: schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "Set to `true` if requesters can approve or decline their own request. They may always comment. Defaults to `false`.",
				},
				MIN_NUM_APPROVALS: schema.Int64Attribute{
					Optional:    true,
					Computed:    true,
					Default:     int64default.StaticInt64(1),
					Description: "The number of approvals required before an approval request can be applied. This number must be between 1 and 5. Defaults to 1.",
				},
				CAN_APPLY_DECLINED_CHANGES: schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(true),
					Description: "Set to `true` if changes can be applied as long as the `min_num_approvals` is met, regardless of whether any reviewers have declined a request. Defaults to `true`.",
				},
				REQUIRED_APPROVAL_TAGS: schema.ListAttribute{
					Optional:    true,
					Computed:    true,
					ElementType: types.StringType,
					Description: "An array of tags used to specify which segments with those tags require approval. You may only set `required_approval_tags` if `required` is set to `false` and vice versa.",
				},
				SERVICE_KIND: schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Description: "The kind of service associated with this approval. This determines which platform is used for requesting approval. Valid values are `servicenow`, `launchdarkly`. If you use a value other than `launchdarkly`, you must have already configured the integration in the LaunchDarkly UI or your apply will fail.",
				},
				SERVICE_CONFIG: schema.MapAttribute{
					Optional:    true,
					Computed:    true,
					ElementType: types.StringType,
					Description: "The configuration for the service associated with this approval. This is specific to each approval service. For a `service_kind` of `servicenow`, the following fields apply:\n\n\t - `template` (String) The sys_id of the Standard Change Request Template in ServiceNow that LaunchDarkly will use when creating the change request.\n\t - `detail_column` (String) The name of the ServiceNow Change Request column LaunchDarkly uses to populate detailed approval request information. This is most commonly \"justification\".",
				},
				AUTO_APPLY_APPROVED_CHANGES: schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "Automatically apply changes that have been approved by all reviewers. This field is only applicable for approval service kinds other than `launchdarkly`.",
				},
			},
		},
	}
}

// frameworkSegmentApprovalSettingsDataSourceAttribute returns the
// data-source-side schema for segment_approval_settings.
func frameworkSegmentApprovalSettingsDataSourceAttribute() dsschema.ListNestedAttribute {
	return dsschema.ListNestedAttribute{
		Computed:    true,
		Description: "Approval settings for segment changes in this environment.",
		NestedObject: dsschema.NestedAttributeObject{
			Attributes: map[string]dsschema.Attribute{
				REQUIRED: dsschema.BoolAttribute{
					Computed:    true,
					Description: "Whether segment changes require approval.",
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
					Description: "Segment tags requiring approval (only one of required / required_approval_tags is set).",
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

// approvalSettingsFromRequestSetting maps a beta-API ApprovalRequestSetting
// (the segment-approvals read shape) onto the flag-approvals
// ldapi.ApprovalSettings shape so the existing conversion helpers
// (frameworkApprovalSettingsValue / ...DataSourceValue) can be reused.
func approvalSettingsFromRequestSetting(s *ldapi.ApprovalRequestSetting) *ldapi.ApprovalSettings {
	if s == nil {
		return nil
	}
	out := &ldapi.ApprovalSettings{
		Required:                         s.Required,
		BypassApprovalsForPendingChanges: s.BypassApprovalsForPendingChanges,
		MinNumApprovals:                  s.MinNumApprovals,
		CanReviewOwnRequest:              s.CanReviewOwnRequest,
		CanApplyDeclinedChanges:          s.CanApplyDeclinedChanges,
		ServiceKind:                      s.ServiceKind,
		ServiceConfig:                    s.ServiceConfig,
		RequiredApprovalTags:             s.RequiredApprovalTags,
	}
	if s.AutoApplyApprovedChanges.IsSet() {
		out.AutoApplyApprovedChanges = s.AutoApplyApprovedChanges.Get()
	}
	return out
}

// segmentApprovalSettingFromGET extracts the segment approval-request
// setting for envKey from the beta GET-settings response. The response
// is keyed by resourceKind ("segment"); per-environment overrides live
// under Environments, falling back to the account-level _default.
func segmentApprovalSettingFromGET(resp *map[string]ldapi.ApprovalRequestSettingWithEnvs, envKey string) *ldapi.ApprovalRequestSetting {
	if resp == nil {
		return nil
	}
	withEnvs, ok := (*resp)[segmentResourceKind]
	if !ok {
		return nil
	}
	if withEnvs.Environments != nil {
		if s, ok := (*withEnvs.Environments)[envKey]; ok {
			return &s
		}
	}
	return withEnvs.Default
}

// segmentApprovalSettingsPatch builds the beta approval-request settings
// PATCH body for the segment resourceKind from a segment_approval_settings
// plan list. A null/empty list yields a "disable" patch (required=false,
// tags cleared) since LD has no delete operation for these settings.
func segmentApprovalSettingsPatch(ctx context.Context, planList types.List, envKey string) (ldapi.ApprovalRequestSettingsPatch, diag.Diagnostics) {
	var diags diag.Diagnostics
	patch := ldapi.ApprovalRequestSettingsPatch{
		EnvironmentKey: envKey,
		ResourceKind:   segmentResourceKind,
	}

	planEmpty := planList.IsNull() || planList.IsUnknown() || len(planList.Elements()) == 0
	if planEmpty {
		required := false
		patch.Required = &required
		patch.RequiredApprovalTags = []string{}
		return patch, diags
	}

	var models []approvalSettingsModel
	diags.Append(planList.ElementsAs(ctx, &models, false)...)
	if diags.HasError() || len(models) == 0 {
		return patch, diags
	}
	m := models[0]

	requiredApprovalTags, d := stringSliceFromList(ctx, m.RequiredApprovalTags)
	diags.Append(d...)
	if diags.HasError() {
		return patch, diags
	}
	required := m.Required.ValueBool()
	if required && len(requiredApprovalTags) > 0 {
		diags.AddError("invalid segment_approval_settings config", "required and required_approval_tags cannot be set simultaneously")
		return patch, diags
	}
	serviceKind := m.ServiceKind.ValueString()
	autoApply := m.AutoApplyApprovedChanges.ValueBool()
	if serviceKind == "launchdarkly" && autoApply {
		diags.AddError("invalid segment_approval_settings config", "auto_apply_approved_changes cannot be set to true for service_kind of launchdarkly")
		return patch, diags
	}

	canReviewOwn := m.CanReviewOwnRequest.ValueBool()
	canApplyDeclined := m.CanApplyDeclinedChanges.ValueBool()
	minNum := int32(m.MinNumApprovals.ValueInt64())

	patch.Required = &required
	patch.CanReviewOwnRequest = &canReviewOwn
	patch.CanApplyDeclinedChanges = &canApplyDeclined
	patch.MinNumApprovals = &minNum
	patch.RequiredApprovalTags = requiredApprovalTags
	patch.AutoApplyApprovedChanges = *ldapi.NewNullableBool(&autoApply)

	// serviceKind/serviceConfig are Computed with no default: they are
	// Unknown (empty) until LD fills them. Omit them when unset so the
	// partial PATCH does not clobber LD's defaulted values.
	if serviceKind != "" {
		patch.ServiceKind = &serviceKind
	}
	if !m.ServiceConfig.IsNull() && !m.ServiceConfig.IsUnknown() {
		raw, d := mapStringFromAttr(ctx, m.ServiceConfig)
		diags.Append(d...)
		if len(raw) > 0 {
			serviceConfig := make(map[string]interface{}, len(raw))
			for k, v := range raw {
				serviceConfig[k] = v
			}
			patch.ServiceConfig = serviceConfig
		}
	}

	return patch, diags
}
