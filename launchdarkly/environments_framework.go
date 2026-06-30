package launchdarkly

// environments_framework.go houses the nested environments schema used
// by launchdarkly_project and the conversion helpers between framework
// state values and the LD-API environment shapes.
//
// As of REL-14236 environments is a Map keyed by the environment key
// (was a positional List). The map key carries the environment identity,
// so the nested object no longer has its own `key` attribute. Keying by
// env key makes reorder/add/remove of one environment a no-op for its
// siblings.
//
// The standalone launchdarkly_environment resource lives in
// resource_environment_framework.go and uses the same approval_settings
// shape as the nested-environments attribute here.

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// environmentModel matches the nested-environments element shape used in
// launchdarkly_project. The environment key lives in the enclosing map's
// key, NOT in this struct — terraform tracks env identity by map key
// across plans.
type environmentModel struct {
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

var environmentAttrTypes = map[string]attr.Type{
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

// environmentObjectType is the element type of the environments map.
var environmentObjectType = types.ObjectType{AttrTypes: environmentAttrTypes}

// projectEnvironmentsAttribute returns the nested-environments attribute
// for the project resource schema. It is a Map keyed by environment key.
// Optional+Computed: omitting it (or setting `{}`) lets the project be
// created with the environments LaunchDarkly auto-provisions without
// terraform churn, and a declared map manages exactly its keys.
func projectEnvironmentsAttribute() schema.MapNestedAttribute {
	return schema.MapNestedAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Map of environments that belong to the project, keyed by environment `key`. When managing LaunchDarkly projects in Terraform, you should always manage your environments as nested project resources. Environments not present in the map are left unmanaged (terraform will not modify or delete them), so you can manage a subset and leave the rest to the LaunchDarkly UI. Set this to `{}` to create a project while managing none of its environments. Omitting the attribute entirely also manages no environments (the same effect as `{}`) but emits a plan-time warning — prefer `{}` to be explicit.\n\n-> **Note:** Mixing the use of nested `environments` and [`launchdarkly_environment`](/docs/providers/launchdarkly/r/environment.html) resources is not recommended. `launchdarkly_environment` resources should only be used when the encapsulating project is not managed in Terraform.",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
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
				API_KEY: schema.StringAttribute{
					Computed:      true,
					Sensitive:     true,
					Description:   "The environment's SDK key.",
					PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				},
				MOBILE_KEY: schema.StringAttribute{
					Computed:      true,
					Sensitive:     true,
					Description:   "The environment's mobile key.",
					PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				},
				CLIENT_SIDE_ID: schema.StringAttribute{
					Computed:      true,
					Sensitive:     true,
					Description:   "The environment's client-side ID.",
					PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				},
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

// environmentModelsFromMap unpacks a framework MapValue of nested
// environment objects into a map of typed models keyed by environment key.
func environmentModelsFromMap(ctx context.Context, m types.Map) (map[string]environmentModel, diag.Diagnostics) {
	if m.IsNull() || m.IsUnknown() {
		return nil, nil
	}
	models := make(map[string]environmentModel, len(m.Elements()))
	diags := m.ElementsAs(ctx, &models, false)
	return models, diags
}

// sortedEnvKeys returns the keys of an environment model map in a stable
// order so POST bodies and patch sequences are deterministic.
func sortedEnvKeys(models map[string]environmentModel) []string {
	keys := make([]string, 0, len(models))
	for k := range models {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// environmentPostsFromPlan converts the plan's environments map into a
// slice of ldapi.EnvironmentPost for the initial PostProject call.
func environmentPostsFromPlan(ctx context.Context, m types.Map) ([]ldapi.EnvironmentPost, diag.Diagnostics) {
	models, diags := environmentModelsFromMap(ctx, m)
	if diags.HasError() || models == nil {
		return nil, diags
	}
	posts := make([]ldapi.EnvironmentPost, 0, len(models))
	for _, key := range sortedEnvKeys(models) {
		p, d := environmentPostFromModel(ctx, key, models[key])
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		posts = append(posts, p)
	}
	return posts, diags
}

func environmentPostFromModel(_ context.Context, key string, m environmentModel) (ldapi.EnvironmentPost, diag.Diagnostics) {
	post := ldapi.EnvironmentPost{
		Name:  m.Name.ValueString(),
		Key:   key,
		Color: m.Color.ValueString(),
	}
	if !m.DefaultTTL.IsNull() && !m.DefaultTTL.IsUnknown() {
		ttl := int32(m.DefaultTTL.ValueInt64())
		post.DefaultTtl = &ttl
	}
	return post, nil
}

// environmentPatchFromModels builds the PATCH document applied when an
// env's nested attributes change on an existing project.
func environmentPatchFromModels(ctx context.Context, old environmentModel, hadOld bool, env environmentModel) ([]ldapi.PatchOperation, diag.Diagnostics) {
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

// environmentsMapFromAPI flattens the LD environments slice into a
// framework MapValue keyed by environment key.
//
// The `prior` value's state drives whether unmanaged environments are
// surfaced — and null vs unknown mean different things:
//   - unknown: a create that omitted `environments` (Optional+Computed,
//     not yet known). Manage NOTHING — return the empty map. Materializing
//     the environments LaunchDarkly auto-provisions would make them look
//     managed, so a later partial config would delete the ones the user
//     never declared.
//   - null: import (ImportState set only key+id). Surface EVERY environment
//     so import captures the whole project.
//   - populated map: managed mode — track ONLY the keys present in `prior`.
//     Environments created outside terraform are left untracked, which is
//     what lets a user manage a subset and leave the rest to the UI.
func environmentsMapFromAPI(ctx context.Context, envs []ldapi.Environment, prior types.Map) (basetypes.MapValue, diag.Diagnostics) {
	var diags diag.Diagnostics
	envByKey := make(map[string]ldapi.Environment, len(envs))
	for _, e := range envs {
		envByKey[e.Key] = e
	}

	elements := map[string]attr.Value{}
	if prior.IsUnknown() {
		m, d := types.MapValue(environmentObjectType, elements)
		diags.Append(d...)
		return m, diags
	}
	if prior.IsNull() {
		for _, e := range envs {
			obj, d := environmentObjectFromAPI(ctx, e, nil)
			diags.Append(d...)
			elements[e.Key] = obj
		}
		m, d := types.MapValue(environmentObjectType, elements)
		diags.Append(d...)
		return m, diags
	}

	priorModels, d := environmentModelsFromMap(ctx, prior)
	diags.Append(d...)
	if diags.HasError() {
		return types.MapNull(environmentObjectType), diags
	}
	for key, pm := range priorModels {
		envAPI, ok := envByKey[key]
		if !ok {
			// Managed env deleted out-of-band: drop it so the next plan
			// shows it being recreated rather than carrying stale state.
			continue
		}
		pmCopy := pm
		obj, d := environmentObjectFromAPI(ctx, envAPI, &pmCopy)
		diags.Append(d...)
		elements[key] = obj
	}
	m, d := types.MapValue(environmentObjectType, elements)
	diags.Append(d...)
	return m, diags
}

func environmentObjectFromAPI(ctx context.Context, e ldapi.Environment, prior *environmentModel) (basetypes.ObjectValue, diag.Diagnostics) {
	var diags diag.Diagnostics
	var (
		tags      types.Set
		approvals basetypes.ListValue
	)
	if prior == nil {
		// No prior state for this env (Import context or new env added
		// outside config): emit using isZero detection so the shape
		// matches what the user's last Apply produced.
		tagsSet, d := setFromStringSliceOrNull(ctx, e.Tags)
		diags.Append(d...)
		tags = tagsSet
		objectType := types.ObjectType{AttrTypes: frameworkApprovalSettingsObjectAttrTypes}
		if e.ApprovalSettings == nil || isZeroApprovalSettings(e.ApprovalSettings) {
			approvals = types.ListNull(objectType)
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
	obj, d := types.ObjectValue(environmentAttrTypes, map[string]attr.Value{
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
