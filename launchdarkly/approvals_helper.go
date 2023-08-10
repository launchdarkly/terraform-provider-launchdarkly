package launchdarkly

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go/v12"
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
	settings := ldapi.ApprovalSettings{
		CanReviewOwnRequest:     approvalSettingsMap[CAN_REVIEW_OWN_REQUEST].(bool),
		MinNumApprovals:         int32(approvalSettingsMap[MIN_NUM_APPROVALS].(int)),
		CanApplyDeclinedChanges: approvalSettingsMap[CAN_APPLY_DECLINED_CHANGES].(bool),
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
	return settings, nil
}

func approvalSettingsToResourceData(settings ldapi.ApprovalSettings) interface{} {
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
