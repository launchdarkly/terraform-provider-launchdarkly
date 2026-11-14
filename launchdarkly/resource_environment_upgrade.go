package launchdarkly

// Frozen pre-object-syntax environment schema + model used as
// PriorSchema for the v0->v1 state upgrader. The v0 shape stored
// `approval_settings` (v2.x SDKv2 block) and `segment_approval_settings`
// (3.0.0-beta.3/-beta.4 nested attribute) as single-element lists; the
// current schema models both as single objects. The upgrader decodes
// prior state into EnvironmentResourceModelV0 and projects each list to
// an object. Genuine v2.x state has no segment_approval_settings
// attribute at all, which decodes as null and stays null.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type EnvironmentResourceModelV0 struct {
	ID                      types.String `tfsdk:"id"`
	ProjectKey              types.String `tfsdk:"project_key"`
	Key                     types.String `tfsdk:"key"`
	Name                    types.String `tfsdk:"name"`
	Color                   types.String `tfsdk:"color"`
	APIKey                  types.String `tfsdk:"api_key"`
	MobileKey               types.String `tfsdk:"mobile_key"`
	ClientSideID            types.String `tfsdk:"client_side_id"`
	DefaultTTL              types.Int64  `tfsdk:"default_ttl"`
	SecureMode              types.Bool   `tfsdk:"secure_mode"`
	DefaultTrackEvents      types.Bool   `tfsdk:"default_track_events"`
	RequireComments         types.Bool   `tfsdk:"require_comments"`
	ConfirmChanges          types.Bool   `tfsdk:"confirm_changes"`
	Critical                types.Bool   `tfsdk:"critical"`
	Tags                    types.Set    `tfsdk:"tags"`
	ApprovalSettings        types.List   `tfsdk:"approval_settings"`
	SegmentApprovalSettings types.List   `tfsdk:"segment_approval_settings"`
}

// environmentSchemaAttributesV0 pins approval_settings and
// segment_approval_settings to the original single-element list shape so
// prior state decodes. All other attributes are unchanged from the
// current schema.
func environmentSchemaAttributesV0() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id":           schema.StringAttribute{Computed: true},
		PROJECT_KEY:    schema.StringAttribute{Required: true},
		KEY:            schema.StringAttribute{Required: true},
		NAME:           schema.StringAttribute{Required: true},
		COLOR:          schema.StringAttribute{Required: true},
		API_KEY:        schema.StringAttribute{Computed: true, Sensitive: true},
		MOBILE_KEY:     schema.StringAttribute{Computed: true, Sensitive: true},
		CLIENT_SIDE_ID: schema.StringAttribute{Computed: true, Sensitive: true},
		DEFAULT_TTL:    schema.Int64Attribute{Optional: true, Computed: true},
		SECURE_MODE:    schema.BoolAttribute{Optional: true, Computed: true},
		DEFAULT_TRACK_EVENTS: schema.BoolAttribute{
			Optional: true, Computed: true,
		},
		REQUIRE_COMMENTS: schema.BoolAttribute{Optional: true, Computed: true},
		CONFIRM_CHANGES:  schema.BoolAttribute{Optional: true, Computed: true},
		CRITICAL:         schema.BoolAttribute{Optional: true, Computed: true},
		TAGS: schema.SetAttribute{
			Optional: true, ElementType: types.StringType,
		},
		APPROVAL_SETTINGS:         approvalSettingsAttributeV0(),
		SEGMENT_APPROVAL_SETTINGS: approvalSettingsAttributeV0(),
	}
}

func (r *EnvironmentResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	priorSchema := schema.Schema{Attributes: environmentSchemaAttributesV0()}
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &priorSchema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior EnvironmentResourceModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}
				approvalsObj, d := approvalSettingsObjectFromV0List(ctx, prior.ApprovalSettings)
				resp.Diagnostics.Append(d...)
				segmentApprovalsObj, d := approvalSettingsObjectFromV0List(ctx, prior.SegmentApprovalSettings)
				resp.Diagnostics.Append(d...)
				if resp.Diagnostics.HasError() {
					return
				}
				data := EnvironmentResourceModel{
					ID:                      prior.ID,
					ProjectKey:              prior.ProjectKey,
					Key:                     prior.Key,
					Name:                    prior.Name,
					Color:                   prior.Color,
					APIKey:                  prior.APIKey,
					MobileKey:               prior.MobileKey,
					ClientSideID:            prior.ClientSideID,
					DefaultTTL:              prior.DefaultTTL,
					SecureMode:              prior.SecureMode,
					DefaultTrackEvents:      prior.DefaultTrackEvents,
					RequireComments:         prior.RequireComments,
					ConfirmChanges:          prior.ConfirmChanges,
					Critical:                prior.Critical,
					Tags:                    prior.Tags,
					ApprovalSettings:        approvalsObj,
					SegmentApprovalSettings: segmentApprovalsObj,
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			},
		},
	}
}
