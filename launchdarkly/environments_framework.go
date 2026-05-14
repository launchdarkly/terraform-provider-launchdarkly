package launchdarkly

// environments_framework.go houses the framework-flavoured nested
// environments block schema (used by launchdarkly_project) and the
// conversion helpers between framework state values and the SDKv2
// environment helpers in environments_helper.go.
//
// The standalone launchdarkly_environment resource (Phase 3) lives in
// resource_environment_framework.go and uses its own ApprovalSettings
// block defined inline; here we re-declare the equivalent shape so the
// nested project block matches user HCL byte-for-byte.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// environmentBlockModel matches the nested-environments block element
// shape used in launchdarkly_project. Mirrors environment_helper.go's
// SDKv2 environmentSchema (forProject: true), where KEY is NOT ForceNew.
type environmentBlockModel struct {
	Key                types.String `tfsdk:"key"`
	Name               types.String `tfsdk:"name"`
	Color              types.String `tfsdk:"color"`
	Critical           types.Bool   `tfsdk:"critical"`
	APIKey             types.String `tfsdk:"api_key"`
	MobileKey          types.String `tfsdk:"mobile_key"`
	ClientSideID       types.String `tfsdk:"client_side_id"`
	DefaultTTL         types.Int64  `tfsdk:"default_ttl"`
	SecureMode         types.Bool   `tfsdk:"secure_mode"`
	DefaultTrackEvents types.Bool   `tfsdk:"default_track_events"`
	RequireComments    types.Bool   `tfsdk:"require_comments"`
	ConfirmChanges     types.Bool   `tfsdk:"confirm_changes"`
	Tags               types.Set    `tfsdk:"tags"`
	ApprovalSettings   types.List   `tfsdk:"approval_settings"`
}

var environmentBlockAttrTypes = map[string]attr.Type{
	KEY:                  types.StringType,
	NAME:                 types.StringType,
	COLOR:                types.StringType,
	CRITICAL:             types.BoolType,
	API_KEY:              types.StringType,
	MOBILE_KEY:           types.StringType,
	CLIENT_SIDE_ID:       types.StringType,
	DEFAULT_TTL:          types.Int64Type,
	SECURE_MODE:          types.BoolType,
	DEFAULT_TRACK_EVENTS: types.BoolType,
	REQUIRE_COMMENTS:     types.BoolType,
	CONFIRM_CHANGES:      types.BoolType,
	TAGS:                 types.SetType{ElemType: types.StringType},
	APPROVAL_SETTINGS:    types.ListType{ElemType: types.ObjectType{AttrTypes: frameworkApprovalSettingsObjectAttrTypes}},
}

// projectEnvironmentsAttribute returns the nested-environments attribute
// for the project resource schema. It is Required (Min:1 enforced by
// the list-size validator).
func projectEnvironmentsAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Required:    true,
		Description: "List of nested `environments` attributes describing LaunchDarkly environments that belong to the project. When managing LaunchDarkly projects in Terraform, you should always manage your environments as nested project resources.\n\n-> **Note:** Mixing the use of nested `environments` and [`launchdarkly_environment`](/docs/providers/launchdarkly/r/environment.html) resources is not recommended. `launchdarkly_environment` resources should only be used when the encapsulating project is not managed in Terraform.",
		Validators:  []validator.List{listvalidator.SizeAtLeast(1)},
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				KEY: schema.StringAttribute{
					Required:    true,
					Description: addForceNewDescription("The project-unique key for the environment.", true),
					Validators:  []validator.String{keyValidator()},
				},
				NAME: schema.StringAttribute{
					Required:    true,
					Description: "The name of the environment.",
				},
				COLOR: schema.StringAttribute{
					Required:    true,
					Description: "The color swatch as an RGB hex value with no leading `#`. For example: `000000`",
				},
				CRITICAL: schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "Denotes whether the environment is critical.",
				},
				API_KEY:        schema.StringAttribute{Computed: true, Sensitive: true, Description: "The environment's SDK key."},
				MOBILE_KEY:     schema.StringAttribute{Computed: true, Sensitive: true, Description: "The environment's mobile key."},
				CLIENT_SIDE_ID: schema.StringAttribute{Computed: true, Sensitive: true, Description: "The environment's client-side ID."},
				DEFAULT_TTL: schema.Int64Attribute{
					Optional:    true,
					Computed:    true,
					Default:     int64default.StaticInt64(0),
					Validators:  []validator.Int64{int64validator.Between(0, 60)},
					Description: "The TTL for the environment. This must be between 0 and 60 minutes. The TTL setting only applies to environments using the PHP SDK. This field will default to `0` when not set. To learn more, read [TTL settings](https://docs.launchdarkly.com/home/organize/environments#ttl-settings).",
				},
				SECURE_MODE: schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "Set to `true` to ensure a user of the client-side SDK cannot impersonate another user. This field will default to `false` when not set.",
				},
				DEFAULT_TRACK_EVENTS: schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "Set to `true` to enable data export for every flag created in this environment after you configure this argument. This field will default to `false` when not set. To learn more, read [Data Export](https://docs.launchdarkly.com/home/data-export).",
				},
				REQUIRE_COMMENTS: schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "Set to `true` if this environment requires comments for flag and segment changes. This field will default to `false` when not set.",
				},
				CONFIRM_CHANGES: schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "Set to `true` if this environment requires confirmation for flag and segment changes. This field will default to `false` when not set.",
				},
				TAGS: schema.SetAttribute{
					Optional:    true,
					ElementType: types.StringType,
					Validators:  []validator.Set{setvalidator.ValueStringsAre(tagValidator())},
					Description: "Tags associated with your resource.",
				},
				APPROVAL_SETTINGS: frameworkApprovalSettingsResourceAttribute(),
			},
		},
	}
}

// environmentModelsFromList unpacks a framework ListValue of nested
// environment blocks into a slice of typed models.
func environmentModelsFromList(ctx context.Context, list types.List) ([]environmentBlockModel, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var models []environmentBlockModel
	diags := list.ElementsAs(ctx, &models, false)
	return models, diags
}

// environmentPostsFromPlan converts the plan's environments block into a
// slice of ldapi.EnvironmentPost for the initial PostProject call.
func environmentPostsFromPlan(ctx context.Context, list types.List) ([]ldapi.EnvironmentPost, diag.Diagnostics) {
	models, diags := environmentModelsFromList(ctx, list)
	if diags.HasError() || models == nil {
		return nil, diags
	}
	posts := make([]ldapi.EnvironmentPost, 0, len(models))
	for _, m := range models {
		p, d := environmentPostFromModel(ctx, m)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		posts = append(posts, p)
	}
	return posts, diags
}

func environmentPostFromModel(_ context.Context, m environmentBlockModel) (ldapi.EnvironmentPost, diag.Diagnostics) {
	post := ldapi.EnvironmentPost{
		Name:  m.Name.ValueString(),
		Key:   m.Key.ValueString(),
		Color: m.Color.ValueString(),
	}
	if !m.DefaultTTL.IsNull() && !m.DefaultTTL.IsUnknown() {
		ttl := int32(m.DefaultTTL.ValueInt64())
		post.DefaultTtl = &ttl
	}
	return post, nil
}

// environmentPatchFromModels builds the PATCH document mirroring the
// SDKv2 getEnvironmentUpdatePatches behaviour.
func environmentPatchFromModels(ctx context.Context, old environmentBlockModel, hadOld bool, env environmentBlockModel) ([]ldapi.PatchOperation, diag.Diagnostics) {
	var diags diag.Diagnostics
	patches := []ldapi.PatchOperation{
		patchReplace("/name", env.Name.ValueString()),
		patchReplace("/color", env.Color.ValueString()),
		patchReplace("/defaultTtl", env.DefaultTTL.ValueInt64()),
		patchReplace("/secureMode", env.SecureMode.ValueBool()),
		patchReplace("/defaultTrackEvents", env.DefaultTrackEvents.ValueBool()),
		patchReplace("/requireComments", env.RequireComments.ValueBool()),
		patchReplace("/confirmChanges", env.ConfirmChanges.ValueBool()),
		patchReplace("/critical", env.Critical.ValueBool()),
	}
	tags, d := stringSliceFromSet(ctx, env.Tags)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	patches = append(patches, patchReplace("/tags", &tags))

	approvalPatches, d := approvalPatchesFromModels(ctx, env.ApprovalSettings, planOrNullList(hadOld, old.ApprovalSettings))
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	patches = append(patches, approvalPatches...)
	return patches, diags
}

func planOrNullList(hadOld bool, l types.List) types.List {
	if !hadOld {
		return types.ListNull(types.ObjectType{AttrTypes: frameworkApprovalSettingsObjectAttrTypes})
	}
	return l
}

// environmentsListFromAPI flattens the LD environments slice back into a
// framework ListValue, preserving the order of envs already in `prior`
// (the most recent state) then appending any unmanaged environments.
func environmentsListFromAPI(ctx context.Context, envs []ldapi.Environment, prior types.List) (basetypes.ListValue, diag.Diagnostics) {
	objType := types.ObjectType{AttrTypes: environmentBlockAttrTypes}
	envByKey := make(map[string]ldapi.Environment, len(envs))
	for _, e := range envs {
		envByKey[e.Key] = e
	}
	priorModels, diags := environmentModelsFromList(ctx, prior)
	added := map[string]bool{}
	ordered := make([]attr.Value, 0, len(envs))
	for _, p := range priorModels {
		envKey := p.Key.ValueString()
		envAPI, ok := envByKey[envKey]
		if !ok {
			continue
		}
		added[envKey] = true
		obj, d := environmentObjectFromAPI(ctx, envAPI, &p)
		diags.Append(d...)
		ordered = append(ordered, obj)
	}
	for _, e := range envs {
		if added[e.Key] {
			continue
		}
		obj, d := environmentObjectFromAPI(ctx, e, nil)
		diags.Append(d...)
		ordered = append(ordered, obj)
	}
	list, d := types.ListValue(objType, ordered)
	diags.Append(d...)
	return list, diags
}

func environmentObjectFromAPI(ctx context.Context, e ldapi.Environment, prior *environmentBlockModel) (basetypes.ObjectValue, diag.Diagnostics) {
	var diags diag.Diagnostics
	var (
		tags      types.Set
		approvals basetypes.ListValue
	)
	if prior == nil {
		// No prior state for this env (Import context or new env added
		// outside config): emit using SDKv2-style isZero detection so the
		// shape matches what the user's last Apply produced.
		tagsSet, d := setFromStringSliceOrNull(ctx, e.Tags)
		diags.Append(d...)
		tags = tagsSet
		objectType := types.ObjectType{AttrTypes: frameworkApprovalSettingsObjectAttrTypes}
		if e.ApprovalSettings == nil || isZeroApprovalSettings(e.ApprovalSettings) {
			approvals = types.ListValueMust(objectType, []attr.Value{})
		} else {
			list, d := frameworkApprovalSettingsDataSourceValue(ctx, e.ApprovalSettings)
			diags.Append(d...)
			approvals = list
		}
	} else {
		tagsSet, d := setFromStringSlicePreservingPlan(ctx, e.Tags, prior.Tags)
		diags.Append(d...)
		tags = tagsSet
		list, d := frameworkApprovalSettingsValue(ctx, e.ApprovalSettings, prior.ApprovalSettings)
		diags.Append(d...)
		approvals = list
	}
	obj, d := types.ObjectValue(environmentBlockAttrTypes, map[string]attr.Value{
		KEY:                  types.StringValue(e.Key),
		NAME:                 types.StringValue(e.Name),
		COLOR:                types.StringValue(e.Color),
		CRITICAL:             types.BoolValue(e.Critical),
		API_KEY:              types.StringValue(e.ApiKey),
		MOBILE_KEY:           types.StringValue(e.MobileKey),
		CLIENT_SIDE_ID:       types.StringValue(e.Id),
		DEFAULT_TTL:          types.Int64Value(int64(e.DefaultTtl)),
		SECURE_MODE:          types.BoolValue(e.SecureMode),
		DEFAULT_TRACK_EVENTS: types.BoolValue(e.DefaultTrackEvents),
		REQUIRE_COMMENTS:     types.BoolValue(e.RequireComments),
		CONFIRM_CHANGES:      types.BoolValue(e.ConfirmChanges),
		TAGS:                 tags,
		APPROVAL_SETTINGS:    approvals,
	})
	diags.Append(d...)
	return obj, diags
}
