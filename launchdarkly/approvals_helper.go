package launchdarkly

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go/v17"
)

type approvalSchemaOptions struct {
	isDataSource bool
}

func approvalSchema(options approvalSchemaOptions) *schema.Schema {
	elemSchema := map[string]*schema.Schema{
		RESOURCE_KIND: {
			Type:             schema.TypeString,
			Optional:         !options.isDataSource,
			Computed:         options.isDataSource,
			Description:      "The kind of resource for which approval settings should apply. Valid values are `flag`, `segment`, and `aiconfig`. Defaults to `flag`.",
			Default:          "flag",
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"flag", "segment", "aiconfig"}, false)),
		},
		REQUIRED: {
			Type:        schema.TypeBool,
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Description: "Set to `true` for changes to resources of this kind in this environment to require approval. You may only set `required` to true if `required_approval_tags` is not set and vice versa. Defaults to `false`.",
			Default:     false,
		},
		CAN_REVIEW_OWN_REQUEST: {
			Type:        schema.TypeBool,
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Description: "Set to `true` if requesters can approve or decline their own request. They may always comment. Defaults to `false`.",
			Default:     false,
		},
		MIN_NUM_APPROVALS: {
			Type:             schema.TypeInt,
			Optional:         !options.isDataSource,
			Computed:         options.isDataSource,
			Description:      "The number of approvals required before an approval request can be applied. This number must be between 1 and 5. Defaults to 1.",
			ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(1, 5)),
			Default:          1,
		},
		CAN_APPLY_DECLINED_CHANGES: {
			Type:        schema.TypeBool,
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Description: "Set to `true` if changes can be applied as long as the `min_num_approvals` is met, regardless of whether any reviewers have declined a request. Defaults to `true`.",
			Default:     true,
		},
		REQUIRED_APPROVAL_TAGS: {
			Type:        schema.TypeList,
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Description: "An array of tags used to specify which flags with those tags require approval. You may only set `required_approval_tags` if `required` is set to `false` and vice versa.",
			Elem: &schema.Schema{
				Type: schema.TypeString,
				// Can't use validation.ToDiagFunc converted validators on TypeList at the moment
				// https://github.com/hashicorp/terraform-plugin-sdk/issues/734
				ValidateFunc: validateTagsNoDiag(),
			},
		},
		SERVICE_KIND: {
			Type:             schema.TypeString,
			Optional:         !options.isDataSource,
			Computed:         options.isDataSource,
			Description:      "The kind of service associated with this approval. This determines which platform is used for requesting approval. Valid values are `servicenow`, `launchdarkly`. **Note:** This field is only supported for `resource_kind = \"flag\"`. Using this field with `resource_kind = \"segment\"` or `resource_kind = \"aiconfig\"` will result in an error. If you use a value other than `launchdarkly`, you must have already configured the integration in the LaunchDarkly UI or your apply will fail.",
			Default:          "launchdarkly",
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"servicenow", "launchdarkly"}, false)),
		},
		SERVICE_CONFIG: {
			Type:        schema.TypeMap,
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Description: "The configuration for the service associated with this approval. **Note:** This field is only supported for `resource_kind = \"flag\"`. Using this field with `resource_kind = \"segment\"` or `resource_kind = \"aiconfig\"` will result in an error. For a `service_kind` of `servicenow`, the following fields apply:\n\n\t - `template` (String) The sys_id of the Standard Change Request Template in ServiceNow that LaunchDarkly will use when creating the change request.\n\t - `detail_column` (String) The name of the ServiceNow Change Request column LaunchDarkly uses to populate detailed approval request information. This is most commonly \"justification\".",
		},
		AUTO_APPLY_APPROVED_CHANGES: {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Automatically apply changes that have been approved by all reviewers. **Note:** This field is only supported for `resource_kind = \"flag\"` and is only applicable for approval service kinds other than `launchdarkly`.",
			Default:     false,
		},
	}

	if options.isDataSource {
		elemSchema = removeInvalidFieldsForDataSource(elemSchema)
	}

	approvalSettingsSchema := &schema.Schema{
		Type:     schema.TypeList,
		Optional: !options.isDataSource,
		Computed: true,
		Elem: &schema.Resource{
			Schema: elemSchema,
		},
	}

	return approvalSettingsSchema
}

// approvalSettingWithKind wraps approval settings with its resource kind
type approvalSettingWithKind struct {
	ResourceKind string
	Settings     ldapi.ApprovalSettings
}

func approvalSettingFromMap(approvalSettingsMap map[string]interface{}) (approvalSettingWithKind, error) {
	resourceKind, ok := approvalSettingsMap[RESOURCE_KIND].(string)
	if !ok || resourceKind == "" {
		resourceKind = "flag" // default value
	}

	autoApply := approvalSettingsMap[AUTO_APPLY_APPROVED_CHANGES].(bool)
	settings := ldapi.ApprovalSettings{
		CanReviewOwnRequest:      approvalSettingsMap[CAN_REVIEW_OWN_REQUEST].(bool),
		MinNumApprovals:          int32(approvalSettingsMap[MIN_NUM_APPROVALS].(int)),
		CanApplyDeclinedChanges:  approvalSettingsMap[CAN_APPLY_DECLINED_CHANGES].(bool),
		ServiceKind:              approvalSettingsMap[SERVICE_KIND].(string),
		ServiceConfig:            approvalSettingsMap[SERVICE_CONFIG].(map[string]interface{}),
		AutoApplyApprovedChanges: &autoApply,
	}
	// Required and RequiredApprovalTags should never be defined simultaneously
	// unfortunately since they default to their null values and are nested we cannot tell if the
	// user has put a value in their config, so we'll check this way
	required := approvalSettingsMap[REQUIRED].(bool)
	tags := approvalSettingsMap[REQUIRED_APPROVAL_TAGS].([]interface{})
	if len(tags) > 0 {
		if required {
			return approvalSettingWithKind{}, fmt.Errorf("invalid approval_settings config: required and required_approval_tags cannot be set simultaneously")
		}
		stringTags := make([]string, len(tags))
		for i := range tags {
			stringTags[i] = tags[i].(string)
		}
		settings.RequiredApprovalTags = stringTags
	} else {
		settings.Required = required
	}

	settings.ServiceKind = approvalSettingsMap[SERVICE_KIND].(string)
	settings.ServiceConfig = approvalSettingsMap[SERVICE_CONFIG].(map[string]interface{})

	// Validate that service_kind and service_config are only used with flag resource kinds
	if resourceKind != "flag" {
		// Error if service_kind is set to non-default value
		serviceKind := settings.ServiceKind
		if serviceKind != "launchdarkly" {
			return approvalSettingWithKind{}, fmt.Errorf("invalid approval_settings config: service_kind cannot be set for resource_kind '%s'. This field is only supported for resource_kind 'flag'", resourceKind)
		}

		// Error if service_config has any values
		if len(settings.ServiceConfig) > 0 {
			return approvalSettingWithKind{}, fmt.Errorf("invalid approval_settings config: service_config cannot be set for resource_kind '%s'. This field is only supported for resource_kind 'flag'", resourceKind)
		}
	}

	// Validate that auto_apply cannot be used with launchdarkly service_kind (existing validation)
	if settings.ServiceKind == "launchdarkly" && settings.AutoApplyApprovedChanges != nil && *settings.AutoApplyApprovedChanges {
		return approvalSettingWithKind{}, fmt.Errorf("invalid approval_settings config: auto_apply_approved_changes cannot be set to true for service_kind of launchdarkly")
	}

	return approvalSettingWithKind{ResourceKind: resourceKind, Settings: settings}, nil
}

func approvalSettingToResourceData(settings ldapi.ApprovalSettings, resourceKind string) map[string]interface{} {
	return map[string]interface{}{
		RESOURCE_KIND:               resourceKind,
		CAN_REVIEW_OWN_REQUEST:      settings.CanReviewOwnRequest,
		MIN_NUM_APPROVALS:           settings.MinNumApprovals,
		CAN_APPLY_DECLINED_CHANGES:  settings.CanApplyDeclinedChanges,
		REQUIRED_APPROVAL_TAGS:      settings.RequiredApprovalTags,
		REQUIRED:                    settings.Required,
		SERVICE_KIND:                settings.ServiceKind,
		SERVICE_CONFIG:              settings.ServiceConfig,
		AUTO_APPLY_APPROVED_CHANGES: settings.AutoApplyApprovedChanges,
	}
}

// environmentApprovalSettingsToResourceData converts environment approval settings from API response
// It handles both approvalSettings (flags) and resourceApprovalSettings (segment, aiconfig, etc.)
func environmentApprovalSettingsToResourceData(env ldapi.Environment) []map[string]interface{} {
	result := make([]map[string]interface{}, 0)

	// Handle flag approval settings (from root approvalSettings)
	// Always include flag approval settings if present for backwards compatibility
	if env.ApprovalSettings != nil {
		result = append(result, approvalSettingToResourceData(*env.ApprovalSettings, "flag"))
	}

	// Handle resource approval settings (from resourceApprovalSettings)
	if env.ResourceApprovalSettings != nil {
		resourceApprovalSettings := *env.ResourceApprovalSettings

		// Handle segment approval settings - only include if actually configured
		if segmentSettings, ok := resourceApprovalSettings["segment"]; ok {
			// Skip if not configured (just defaults from API)
			if segmentSettings.Required || len(segmentSettings.RequiredApprovalTags) > 0 {
				result = append(result, approvalSettingToResourceData(segmentSettings, "segment"))
			}
		}

		// Handle aiconfig approval settings - only include if actually configured
		if aiconfigSettings, ok := resourceApprovalSettings["aiconfig"]; ok {
			// Skip if not configured (just defaults from API)
			if aiconfigSettings.Required || len(aiconfigSettings.RequiredApprovalTags) > 0 {
				result = append(result, approvalSettingToResourceData(aiconfigSettings, "aiconfig"))
			}
		}
	}

	return result
}

// approvalPatchForResourceKind generates patch operations for a specific resource kind
func approvalPatchForResourceKind(settings approvalSettingWithKind) []ldapi.PatchOperation {
	// Determine the base path based on resource kind
	var basePath string
	if settings.ResourceKind == "flag" {
		basePath = "/approvalSettings"
	} else {
		basePath = fmt.Sprintf("/resourceApprovalSettings/%s", settings.ResourceKind)
	}

	patch := []ldapi.PatchOperation{
		patchReplace(basePath+"/required", settings.Settings.Required),
		patchReplace(basePath+"/canReviewOwnRequest", settings.Settings.CanReviewOwnRequest),
		patchReplace(basePath+"/minNumApprovals", settings.Settings.MinNumApprovals),
		patchReplace(basePath+"/canApplyDeclinedChanges", settings.Settings.CanApplyDeclinedChanges),
		patchReplace(basePath+"/requiredApprovalTags", settings.Settings.RequiredApprovalTags),
	}

	// serviceKind, serviceConfig, and autoApplyApprovedChanges are only supported for flag approval settings
	if settings.ResourceKind == "flag" {
		patch = append(patch,
			patchReplace(basePath+"/serviceKind", settings.Settings.ServiceKind),
			patchReplace(basePath+"/serviceConfig", settings.Settings.ServiceConfig),
		)
		if settings.Settings.AutoApplyApprovedChanges != nil {
			patch = append(patch, patchReplace(basePath+"/autoApplyApprovedChanges", *settings.Settings.AutoApplyApprovedChanges))
		}
	}

	return patch
}

func approvalPatchFromSettings(oldApprovalSettings, newApprovalSettings []interface{}) ([]ldapi.PatchOperation, error) {
	new := newApprovalSettings
	old := oldApprovalSettings

	if len(new) == 0 && len(old) == 0 {
		return []ldapi.PatchOperation{}, nil
	}

	// Validate that there are no duplicate resource_kind values
	seenKinds := make(map[string]bool)
	for _, rawSetting := range new {
		if rawSetting == nil {
			continue
		}
		setting := rawSetting.(map[string]interface{})
		kind, ok := setting[RESOURCE_KIND].(string)
		if !ok || kind == "" {
			kind = "flag"
		}
		if seenKinds[kind] {
			return []ldapi.PatchOperation{}, fmt.Errorf("duplicate resource_kind '%s' found in approval_settings. Each resource_kind can only be specified once", kind)
		}
		seenKinds[kind] = true
	}

	patches := []ldapi.PatchOperation{}

	// Track which resource kinds exist in old and new
	oldKinds := make(map[string]bool)
	newKinds := make(map[string]bool)

	for _, rawSetting := range old {
		if rawSetting == nil {
			continue
		}
		setting := rawSetting.(map[string]interface{})
		kind, ok := setting[RESOURCE_KIND].(string)
		if !ok || kind == "" {
			kind = "flag"
		}
		oldKinds[kind] = true
	}

	for _, rawSetting := range new {
		if rawSetting == nil {
			continue
		}
		setting := rawSetting.(map[string]interface{})
		kind, ok := setting[RESOURCE_KIND].(string)
		if !ok || kind == "" {
			kind = "flag"
		}
		newKinds[kind] = true
	}

	// Handle removals - resource kinds that were in old but not in new
	for kind := range oldKinds {
		if !newKinds[kind] {
			if kind == "flag" {
				patches = append(patches,
					patchRemove("/approvalSettings/required"),
					patchRemove("/approvalSettings/requiredApprovalTags"),
				)
			} else {
				basePath := fmt.Sprintf("/resourceApprovalSettings/%s", kind)
				patches = append(patches,
					patchRemove(basePath+"/required"),
					patchRemove(basePath+"/requiredApprovalTags"),
				)
			}
		}
	}

	// Handle additions and updates - resource kinds that are in new
	for _, rawSetting := range new {
		if rawSetting == nil {
			continue
		}
		settingMap := rawSetting.(map[string]interface{})
		setting, err := approvalSettingFromMap(settingMap)
		if err != nil {
			return []ldapi.PatchOperation{}, err
		}

		patches = append(patches, approvalPatchForResourceKind(setting)...)
	}

	return patches, nil
}
