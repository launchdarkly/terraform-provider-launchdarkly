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
		REQUIRED: {
			Type:        schema.TypeBool,
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Description: "Set to `true` for changes to flags in this environment to require approval. You may only set `required` to true if `required_approval_tags` is not set and vice versa. Defaults to `false`.",
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
			Description: "An array of tags used to specify which flags with those tags require approval. You may only set `required_approval_tags` if `required` is not set to `true` and vice versa.",
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
			Description:      "The kind of service associated with this approval. This determines which platform is used for requesting approval. Valid values are `servicenow`, `launchdarkly`. If you use a value other than `launchdarkly`, you must have already configured the integration in the LaunchDarkly UI or your apply will fail.",
			Default:          "launchdarkly",
			ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"servicenow", "launchdarkly"}, false)),
		},
		SERVICE_CONFIG: {
			Type:        schema.TypeMap,
			Optional:    !options.isDataSource,
			Computed:    options.isDataSource,
			Description: "The configuration for the service associated with this approval. This is specific to each approval service. For a `service_kind` of `servicenow`, the following fields apply:\n\n\t - `template` (String) The sys_id of the Standard Change Request Template in ServiceNow that LaunchDarkly will use when creating the change request.\n\t - `detail_column` (String) The name of the ServiceNow Change Request column LaunchDarkly uses to populate detailed approval request information. This is most commonly \"justification\".",
		},
		AUTO_APPLY_APPROVED_CHANGES: {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Automatically apply changes that have been approved by all reviewers. This field is only applicable for approval service kinds other than `launchdarkly`.",
			Default:     false,
		},
	}

	if options.isDataSource {
		elemSchema = removeInvalidFieldsForDataSource(elemSchema)
	}

	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: !options.isDataSource,
		Computed: true,
		Elem: &schema.Resource{
			Schema: elemSchema,
		},
	}
}

func approvalSettingsFromResourceData(val interface{}) (ldapi.ApprovalSettings, error) {
	raw := val.([]interface{})
	if len(raw) == 0 {
		return ldapi.ApprovalSettings{}, nil
	}
	approvalSettingsMap := raw[0].(map[string]interface{})
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
			return ldapi.ApprovalSettings{}, fmt.Errorf("invalid approval_settings config: required and required_approval_tags cannot be set simultaneously")
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
	if settings.ServiceKind == "launchdarkly" && settings.AutoApplyApprovedChanges != nil && *settings.AutoApplyApprovedChanges {
		return ldapi.ApprovalSettings{}, fmt.Errorf("invalid approval_settings config: auto_apply_approved_changes cannot be set to true for service_kind of launchdarkly")
	}
	return settings, nil
}

func approvalSettingsToResourceData(settings ldapi.ApprovalSettings) interface{} {
	transformed := map[string]interface{}{
		CAN_REVIEW_OWN_REQUEST:      settings.CanReviewOwnRequest,
		MIN_NUM_APPROVALS:           settings.MinNumApprovals,
		CAN_APPLY_DECLINED_CHANGES:  settings.CanApplyDeclinedChanges,
		REQUIRED_APPROVAL_TAGS:      settings.RequiredApprovalTags,
		REQUIRED:                    settings.Required,
		SERVICE_KIND:                settings.ServiceKind,
		SERVICE_CONFIG:              settings.ServiceConfig,
		AUTO_APPLY_APPROVED_CHANGES: settings.AutoApplyApprovedChanges,
	}
	return []map[string]interface{}{transformed}
}

func approvalPatchFromSettings(oldApprovalSettings, newApprovalSettings interface{}) ([]ldapi.PatchOperation, error) {
	settings, err := approvalSettingsFromResourceData(newApprovalSettings)
	if err != nil {
		return []ldapi.PatchOperation{}, err
	}
	new := newApprovalSettings.([]interface{})
	old := oldApprovalSettings.([]interface{})
	if len(new) == 0 && len(old) == 0 {
		return []ldapi.PatchOperation{}, nil
	}
	if len(new) == 0 && len(old) > 0 {
		return []ldapi.PatchOperation{
			patchRemove("/approvalSettings/required"),
			patchRemove("/approvalSettings/requiredApprovalTags"),
		}, nil
	}
	patch := []ldapi.PatchOperation{
		patchReplace("/approvalSettings/required", settings.Required),
		patchReplace("/approvalSettings/canReviewOwnRequest", settings.CanReviewOwnRequest),
		patchReplace("/approvalSettings/minNumApprovals", settings.MinNumApprovals),
		patchReplace("/approvalSettings/canApplyDeclinedChanges", settings.CanApplyDeclinedChanges),
		patchReplace("/approvalSettings/requiredApprovalTags", settings.RequiredApprovalTags),
		patchReplace("/approvalSettings/serviceKind", settings.ServiceKind),
		patchReplace("/approvalSettings/serviceConfig", settings.ServiceConfig),
	}
	if settings.AutoApplyApprovedChanges != nil {
		patch = append(patch, patchReplace("/approvalSettings/autoApplyApprovedChanges", *settings.AutoApplyApprovedChanges))
	}
	return patch, nil
}
