package launchdarkly

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go"
)

func approvalSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				REQUIRED: {
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "Whether any changes to flags in this environment will require approval. You may only set required or requiredApprovalTags, not both.",
					Default:     false,
				},
				CAN_REVIEW_OWN_REQUEST: {
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "Whether requesters can approve or decline their own request. They may always comment.",
					Default:     false,
				},
				MIN_NUM_APPROVALS: {
					Type:         schema.TypeInt,
					Optional:     true,
					Description:  "The number of approvals required before an approval request can be applied.",
					ValidateFunc: validation.IntBetween(1, 5),
					Default:      1,
				},
				CAN_APPLY_DECLINED_CHANGES: {
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "Whether changes can be applied as long as minNumApprovals is met, regardless of whether any reviewers have declined a request. Defaults to true",
					Default:     true,
				},
				REQUIRED_APPROVAL_TAGS: {
					Type:        schema.TypeList,
					Optional:    true,
					Description: "An array of tags used to specify which flags with those tags require approval. You may only set requiredApprovalTags or required, not both.",
					Elem: &schema.Schema{
						Type:         schema.TypeString,
						ValidateFunc: validateTags(),
					},
				},
			},
		},
	}
}

func approvalSettingsFromResourceData(val interface{}) (ldapi.EnvironmentApprovalSettings, error) {
	raw := val.([]interface{})
	if len(raw) == 0 {
		return ldapi.EnvironmentApprovalSettings{}, nil
	}
	approvalSettingsMap := raw[0].(map[string]interface{})
	settings := ldapi.EnvironmentApprovalSettings{
		CanReviewOwnRequest:     approvalSettingsMap[CAN_REVIEW_OWN_REQUEST].(bool),
		MinNumApprovals:         int64(approvalSettingsMap[MIN_NUM_APPROVALS].(int)),
		CanApplyDeclinedChanges: approvalSettingsMap[CAN_APPLY_DECLINED_CHANGES].(bool),
	}
	// Required and RequiredApprovalTags should never be defined simultaneously
	// unfortunately since they default to their null values and are nested we cannot tell if the
	// user has put a value in their config, so we'll check this way
	required := approvalSettingsMap[REQUIRED].(bool)
	tags := approvalSettingsMap[REQUIRED_APPROVAL_TAGS].([]interface{})
	if len(tags) > 0 {
		if required {
			return ldapi.EnvironmentApprovalSettings{}, fmt.Errorf("invalid approval_settings config: required and required_approval_tags cannot be set simultaneously")
		}
		stringTags := make([]string, len(tags))
		for i := range tags {
			stringTags[i] = tags[i].(string)
		}
		settings.RequiredApprovalTags = stringTags
	} else {
		settings.Required = required
	}
	return settings, nil
}

func approvalSettingsToResourceData(settings ldapi.EnvironmentApprovalSettings) interface{} {
	transformed := map[string]interface{}{
		CAN_REVIEW_OWN_REQUEST:     settings.CanReviewOwnRequest,
		MIN_NUM_APPROVALS:          settings.MinNumApprovals,
		CAN_APPLY_DECLINED_CHANGES: settings.CanApplyDeclinedChanges,
		REQUIRED_APPROVAL_TAGS:     settings.RequiredApprovalTags,
		REQUIRED:                   settings.Required,
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
	}
	return patch, nil
}
