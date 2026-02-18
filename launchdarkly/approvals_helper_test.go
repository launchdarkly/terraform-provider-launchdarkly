package launchdarkly

import (
	"testing"

	ldapi "github.com/launchdarkly/api-client-go/v17"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApprovalPatchFromSettings_MultipleResourceKinds(t *testing.T) {
	// Test creating patches for both flag and segment approval settings
	oldSettings := []interface{}{}
	newSettings := []interface{}{
		map[string]interface{}{
			RESOURCE_KIND:               "flag",
			REQUIRED:                    true,
			CAN_REVIEW_OWN_REQUEST:      false,
			MIN_NUM_APPROVALS:           2,
			CAN_APPLY_DECLINED_CHANGES:  false,
			SERVICE_KIND:                "launchdarkly",
			SERVICE_CONFIG:              map[string]interface{}{},
			AUTO_APPLY_APPROVED_CHANGES: false,
			REQUIRED_APPROVAL_TAGS:      []interface{}{},
		},
		map[string]interface{}{
			RESOURCE_KIND:               "segment",
			REQUIRED:                    true,
			CAN_REVIEW_OWN_REQUEST:      false,
			MIN_NUM_APPROVALS:           1,
			CAN_APPLY_DECLINED_CHANGES:  true,
			SERVICE_KIND:                "launchdarkly",
			SERVICE_CONFIG:              map[string]interface{}{},
			AUTO_APPLY_APPROVED_CHANGES: false,
			REQUIRED_APPROVAL_TAGS:      []interface{}{},
		},
	}

	patches, err := approvalPatchFromSettings(oldSettings, newSettings)
	require.NoError(t, err)

	// Should have patches for both flag and segment
	// Each resource kind has 8 fields (required, canReviewOwnRequest, minNumApprovals,
	// canApplyDeclinedChanges, requiredApprovalTags, serviceKind, serviceConfig, autoApplyApprovedChanges)
	assert.Equal(t, 16, len(patches), "Expected 16 patches (8 for flag + 8 for segment)")

	// Verify flag patches have /approvalSettings path
	flagPatchCount := 0
	segmentPatchCount := 0
	for _, patch := range patches {
		if len(patch.Path) >= len("/approvalSettings") && patch.Path[:len("/approvalSettings")] == "/approvalSettings" {
			flagPatchCount++
		}
		if len(patch.Path) >= len("/resourceApprovalSettings/segment") && patch.Path[:len("/resourceApprovalSettings/segment")] == "/resourceApprovalSettings/segment" {
			segmentPatchCount++
		}
	}

	assert.Equal(t, 8, flagPatchCount, "Expected 8 patches for flag approval settings")
	assert.Equal(t, 8, segmentPatchCount, "Expected 8 patches for segment approval settings")
}

func TestApprovalPatchFromSettings_RemoveResourceKind(t *testing.T) {
	// Test removing segment approval settings while keeping flag settings
	oldSettings := []interface{}{
		map[string]interface{}{
			RESOURCE_KIND:               "flag",
			REQUIRED:                    true,
			CAN_REVIEW_OWN_REQUEST:      false,
			MIN_NUM_APPROVALS:           1,
			CAN_APPLY_DECLINED_CHANGES:  true,
			SERVICE_KIND:                "launchdarkly",
			SERVICE_CONFIG:              map[string]interface{}{},
			AUTO_APPLY_APPROVED_CHANGES: false,
			REQUIRED_APPROVAL_TAGS:      []interface{}{},
		},
		map[string]interface{}{
			RESOURCE_KIND:               "segment",
			REQUIRED:                    true,
			CAN_REVIEW_OWN_REQUEST:      false,
			MIN_NUM_APPROVALS:           1,
			CAN_APPLY_DECLINED_CHANGES:  true,
			SERVICE_KIND:                "launchdarkly",
			SERVICE_CONFIG:              map[string]interface{}{},
			AUTO_APPLY_APPROVED_CHANGES: false,
			REQUIRED_APPROVAL_TAGS:      []interface{}{},
		},
	}
	newSettings := []interface{}{
		map[string]interface{}{
			RESOURCE_KIND:               "flag",
			REQUIRED:                    true,
			CAN_REVIEW_OWN_REQUEST:      false,
			MIN_NUM_APPROVALS:           1,
			CAN_APPLY_DECLINED_CHANGES:  true,
			SERVICE_KIND:                "launchdarkly",
			SERVICE_CONFIG:              map[string]interface{}{},
			AUTO_APPLY_APPROVED_CHANGES: false,
			REQUIRED_APPROVAL_TAGS:      []interface{}{},
		},
	}

	patches, err := approvalPatchFromSettings(oldSettings, newSettings)
	require.NoError(t, err)

	// Should have removal patches for segment and update patches for flag
	// 2 remove patches for segment + 8 update patches for flag = 10 total
	assert.GreaterOrEqual(t, len(patches), 10, "Expected at least 10 patches")

	// Verify we have remove operations for segment
	hasSegmentRemove := false
	for _, patch := range patches {
		if patch.Op == "remove" && len(patch.Path) >= len("/resourceApprovalSettings/segment") {
			if patch.Path[:len("/resourceApprovalSettings/segment")] == "/resourceApprovalSettings/segment" {
				hasSegmentRemove = true
				break
			}
		}
	}

	assert.True(t, hasSegmentRemove, "Expected remove operation for segment approval settings")
}

func TestEnvironmentApprovalSettingsToResourceData(t *testing.T) {
	// Test converting API response to Terraform resource data
	flagSettings := ldapi.ApprovalSettings{
		Required:                true,
		CanReviewOwnRequest:     false,
		MinNumApprovals:         2,
		CanApplyDeclinedChanges: false,
		ServiceKind:             "launchdarkly",
		ServiceConfig:           map[string]interface{}{},
		RequiredApprovalTags:    []string{},
	}
	autoApply := false
	flagSettings.AutoApplyApprovedChanges = &autoApply

	segmentSettings := ldapi.ApprovalSettings{
		Required:                true,
		CanReviewOwnRequest:     false,
		MinNumApprovals:         1,
		CanApplyDeclinedChanges: true,
		ServiceKind:             "launchdarkly",
		ServiceConfig:           map[string]interface{}{},
		RequiredApprovalTags:    []string{},
	}
	segmentSettings.AutoApplyApprovedChanges = &autoApply

	resourceApprovalSettings := map[string]ldapi.ApprovalSettings{
		"segment": segmentSettings,
	}

	env := ldapi.Environment{
		ApprovalSettings:         &flagSettings,
		ResourceApprovalSettings: &resourceApprovalSettings,
	}

	result := environmentApprovalSettingsToResourceData(env)

	// Should have 2 approval settings blocks
	assert.Equal(t, 2, len(result), "Expected 2 approval settings blocks")

	// Verify flag settings
	var flagBlock map[string]interface{}
	var segmentBlock map[string]interface{}
	for _, block := range result {
		if block[RESOURCE_KIND] == "flag" {
			flagBlock = block
		} else if block[RESOURCE_KIND] == "segment" {
			segmentBlock = block
		}
	}

	assert.NotNil(t, flagBlock, "Expected flag approval settings block")
	assert.Equal(t, true, flagBlock[REQUIRED])
	assert.Equal(t, int32(2), flagBlock[MIN_NUM_APPROVALS])

	assert.NotNil(t, segmentBlock, "Expected segment approval settings block")
	assert.Equal(t, true, segmentBlock[REQUIRED])
	assert.Equal(t, int32(1), segmentBlock[MIN_NUM_APPROVALS])
	assert.Equal(t, true, segmentBlock[CAN_APPLY_DECLINED_CHANGES])
}

func TestApprovalPatchFromSettings_BackwardsCompatibility(t *testing.T) {
	// Test that omitting resource_kind defaults to "flag"
	oldSettings := []interface{}{}
	newSettings := []interface{}{
		map[string]interface{}{
			// No RESOURCE_KIND specified - should default to "flag"
			REQUIRED:                    true,
			CAN_REVIEW_OWN_REQUEST:      false,
			MIN_NUM_APPROVALS:           1,
			CAN_APPLY_DECLINED_CHANGES:  true,
			SERVICE_KIND:                "launchdarkly",
			SERVICE_CONFIG:              map[string]interface{}{},
			AUTO_APPLY_APPROVED_CHANGES: false,
			REQUIRED_APPROVAL_TAGS:      []interface{}{},
		},
	}

	patches, err := approvalPatchFromSettings(oldSettings, newSettings)
	require.NoError(t, err)

	// Verify all patches use /approvalSettings path (flag path)
	for _, patch := range patches {
		assert.Contains(t, patch.Path, "/approvalSettings", "Expected /approvalSettings path for backwards compatibility")
		assert.NotContains(t, patch.Path, "/resourceApprovalSettings", "Should not use /resourceApprovalSettings path when resource_kind is omitted")
	}
}
