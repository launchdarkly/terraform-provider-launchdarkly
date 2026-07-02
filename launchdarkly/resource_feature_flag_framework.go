package launchdarkly

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var (
	_ resource.Resource                     = &FeatureFlagResource{}
	_ resource.ResourceWithImportState      = &FeatureFlagResource{}
	_ resource.ResourceWithModifyPlan       = &FeatureFlagResource{}
	_ resource.ResourceWithConfigValidators = &FeatureFlagResource{}
	_ resource.ResourceWithValidateConfig   = &FeatureFlagResource{}
	_ resource.ResourceWithUpgradeState     = &FeatureFlagResource{}
)

type FeatureFlagResource struct {
	client *Client
}

type FeatureFlagResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	ProjectKey             types.String `tfsdk:"project_key"`
	Key                    types.String `tfsdk:"key"`
	Name                   types.String `tfsdk:"name"`
	Description            types.String `tfsdk:"description"`
	MaintainerID           types.String `tfsdk:"maintainer_id"`
	MaintainerTeamKey      types.String `tfsdk:"maintainer_team_key"`
	Tags                   types.Set    `tfsdk:"tags"`
	VariationType          types.String `tfsdk:"variation_type"`
	Variations             types.List   `tfsdk:"variations"`
	Temporary              types.Bool   `tfsdk:"temporary"`
	ClientSideAvailability types.Object `tfsdk:"client_side_availability"`
	CustomProperties       types.Map    `tfsdk:"custom_properties"`
	Defaults               types.Object `tfsdk:"defaults"`
	Archived               types.Bool   `tfsdk:"archived"`
	Deprecated             types.Bool   `tfsdk:"deprecated"`
	ViewKeys               types.Set    `tfsdk:"view_keys"`
}

var (
	featureFlagCSAAttrTypes = map[string]attr.Type{
		USING_ENVIRONMENT_ID: types.BoolType,
		USING_MOBILE_KEY:     types.BoolType,
	}
)

func NewFeatureFlagResource() resource.Resource {
	return &FeatureFlagResource{}
}

func (r *FeatureFlagResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_feature_flag"
}

func (r *FeatureFlagResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 1,
		Description: `Provides a LaunchDarkly feature flag resource.

This resource allows you to create and manage feature flags within your LaunchDarkly organization.

-> **Note:** This resource is for global-level feature flag configuration. Unexpected behavior may result if your environment-level configurations are not also managed from Terraform.`,
		Attributes: featureFlagSchemaAttributes(),
	}
}

func featureFlagSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		PROJECT_KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The feature flag's project key.", true),
			Validators:    []validator.String{keyValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The unique feature flag key that references the flag in your application code.", true),
			Validators:    []validator.String{keyValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		NAME: schema.StringAttribute{
			Required:    true,
			Description: "The human-readable name of the feature flag.",
		},
		DESCRIPTION: schema.StringAttribute{
			Optional:    true,
			Description: "The feature flag's description.",
		},
		MAINTAINER_ID: schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "The feature flag maintainer's 24 character alphanumeric team member ID. `maintainer_team_key` cannot be set if `maintainer_id` is set. If neither is set, it will automatically be or stay set to the member ID associated with the API key used by your LaunchDarkly Terraform provider or the most recently-set maintainer.",
			Validators:  []validator.String{idValidator()},
			// Intentionally no UseStateForUnknown: maintainer_id and
			// maintainer_team_key are mutually exclusive, so when the
			// user switches from one to the other the unused field
			// must transition to "" (LD's "no value" form) rather than
			// retain the previous state value.
		},
		MAINTAINER_TEAM_KEY: schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "The key of the associated team that maintains this feature flag. `maintainer_id` cannot be set if `maintainer_team_key` is set",
			Validators:  []validator.String{keyAndLengthValidator(1, 256)},
			// See maintainer_id above — same mutual-exclusion reason.
		},
		VARIATION_TYPE: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The feature flag's variation type: `boolean`, `string`, `number` or `json`.", true),
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			Validators: []validator.String{
				oneOfValidator{allowed: []string{BOOL_VARIATION, STRING_VARIATION, NUMBER_VARIATION, JSON_VARIATION}},
			},
		},
		TEMPORARY: schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
			Description: "Specifies whether the flag is a temporary flag.",
		},
		TAGS: schema.SetAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Validators:  []validator.Set{setvalidator.ValueStringsAre(tagValidator())},
			Description: "Tags associated with your resource.",
		},
		ARCHIVED: schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
			Description: "Specifies whether the flag is archived or not. Note that you cannot create a new flag that is archived, but can update a flag to be archived.",
		},
		DEPRECATED: schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
			Description: "Specifies whether the flag is deprecated or not. Note that you cannot create a new flag that is deprecated, but can update a flag to be deprecated.",
		},
		VIEW_KEYS: schema.SetAttribute{
			Optional:      true,
			Computed:      true,
			ElementType:   types.StringType,
			Description:   "A set of view keys to link this flag to. This is an alternative to using the `launchdarkly_view_links` resource for managing view associations. When set, this flag will be linked to the specified views. The field is also computed, meaning Terraform will read back the current view associations from LaunchDarkly to detect drift. To explicitly remove all view associations, set `view_keys = []`. Simply removing the field from your configuration will leave existing associations unchanged. **Important**: Avoid using both `view_keys` and `launchdarkly_view_links` to manage the same flag. Mixed ownership can cause conflicts; when detected, Terraform logs a warning and reconciles to the configured `view_keys`. Choose one approach per resource.",
			PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
		},
		VARIATIONS: schema.ListNestedAttribute{
			Required:    true,
			Description: "An array of possible variations for the flag.",
			Validators:  []validator.List{listvalidator.SizeAtLeast(1)},
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					NAME: schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The name of the variation.",
					},
					DESCRIPTION: schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "The variation's description.",
					},
					VALUE: schema.StringAttribute{
						Required: true,
						PlanModifiers: []planmodifier.String{
							jsonNormalizePlanModifier{},
						},
						Description: fmt.Sprintf("The variation value. The value's type must correspond to the `variation_type` argument. For example: `variation_type = %q` accepts only `true` or `false`. The `number` variation type accepts both floats and ints, but please note that any trailing zeroes on floats will be trimmed (i.e. `1.1` and `1.100` will both be converted to `1.1`).\n\nIf you wish to define an empty string variation, you must still define the value field like so:\n\n```terraform\nvariations = [{\n  value = %q\n}]\n```\n\n-> **Note:** Terraform manages `variations` as an ordered array and identifies them by index. Changing the order of `variations` may destroy and recreate variations. Deleted variations that still have targets attached outside of Terraform may have their targets reassigned to a different variation.", "boolean", ""),
					},
				},
			},
		},
		CLIENT_SIDE_AVAILABILITY: schema.SingleNestedAttribute{
			Optional:    true,
			Description: "Whether this flag should be made available to the client-side JavaScript SDK using the client-side Id, mobile key, or both. This value gets its default from your project configuration if not set. Once set, if removed, it will retain its last set value.",
			Attributes: map[string]schema.Attribute{
				USING_ENVIRONMENT_ID: schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Description: "Whether this flag is available to SDKs using the client-side ID.",
				},
				USING_MOBILE_KEY: schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Description: "Whether this flag is available to SDKs using a mobile key.",
				},
			},
		},
		CUSTOM_PROPERTIES: schema.MapNestedAttribute{
			Optional:    true,
			Description: "The feature flag's [custom properties](https://docs.launchdarkly.com/home/connecting/custom-properties), keyed by the custom property key. Adding or removing one custom property does not affect the others.",
			Validators: []validator.Map{
				mapvalidator.SizeAtMost(CUSTOM_PROPERTY_ITEM_LIMIT),
				mapvalidator.KeysAre(stringLenBetween(1, CUSTOM_PROPERTY_CHAR_LIMIT)),
			},
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					KEY: schema.StringAttribute{
						Optional:      true,
						Computed:      true,
						Description:   "The unique custom property key. Must equal the map key; it defaults to the map key when omitted.",
						Validators:    []validator.String{stringLenBetween(1, CUSTOM_PROPERTY_CHAR_LIMIT)},
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					NAME: schema.StringAttribute{
						Required:    true,
						Description: "The name of the custom property.",
						Validators:  []validator.String{stringLenBetween(1, CUSTOM_PROPERTY_CHAR_LIMIT)},
					},
					VALUE: schema.ListAttribute{
						Required:    true,
						ElementType: types.StringType,
						Validators: []validator.List{
							listvalidator.SizeAtMost(CUSTOM_PROPERTY_ITEM_LIMIT),
							listvalidator.ValueStringsAre(stringLenBetween(1, CUSTOM_PROPERTY_CHAR_LIMIT)),
						},
						Description: "The list of custom property value strings.",
					},
				},
			},
		},
		DEFAULTS: schema.SingleNestedAttribute{
			Optional:    true,
			Description: "The indices of the variations to be used as the default on and off variations in all new environments. Flag configurations in existing environments will not be changed nor updated if removed.",
			Attributes: map[string]schema.Attribute{
				ON_VARIATION: schema.Int64Attribute{
					Required:    true,
					Validators:  []validator.Int64{int64validator.AtLeast(0)},
					Description: "The index of the variation the flag will default to in all new environments when on.",
				},
				OFF_VARIATION: schema.Int64Attribute{
					Required:    true,
					Validators:  []validator.Int64{int64validator.AtLeast(0)},
					Description: "The index of the variation the flag will default to in all new environments when off.",
				},
			},
		},
	}
}

func featureFlagDefaultsMatchesAPIShape(ctx context.Context, defaults types.Object, variations types.List) bool {
	if defaults.IsNull() || defaults.IsUnknown() {
		return false
	}
	if variations.IsNull() || variations.IsUnknown() {
		return false
	}
	variationCount := len(variations.Elements())
	if variationCount == 0 {
		return false
	}
	type defaultsItem struct {
		OnVariation  int64 `tfsdk:"on_variation"`
		OffVariation int64 `tfsdk:"off_variation"`
	}
	var item defaultsItem
	d := defaults.As(ctx, &item, basetypes.ObjectAsOptions{})
	if d.HasError() {
		return false
	}
	return item.OnVariation == 0 && item.OffVariation == int64(variationCount-1)
}

func featureFlagCSAMatchesAPIShape(ctx context.Context, csa types.Object) bool {
	if csa.IsNull() || csa.IsUnknown() {
		return false
	}
	type csaItem struct {
		UsingEnvironmentID types.Bool `tfsdk:"using_environment_id"`
		UsingMobileKey     types.Bool `tfsdk:"using_mobile_key"`
	}
	var item csaItem
	d := csa.As(ctx, &item, basetypes.ObjectAsOptions{})
	if d.HasError() {
		return false
	}
	envOK := !item.UsingEnvironmentID.IsNull() && item.UsingEnvironmentID.ValueBool()
	mobileOK := !item.UsingMobileKey.IsNull() && item.UsingMobileKey.ValueBool()
	return envOK && mobileOK
}

func (r *FeatureFlagResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	priorSchema := schema.Schema{Attributes: featureFlagSchemaAttributesV0()}
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &priorSchema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior FeatureFlagResourceModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}
				// v0 (SDKv2) stored client_side_availability and defaults
				// as single-element lists (blocks). v3 models them as
				// single objects — project the prior lists accordingly.
				priorCSA, d := csaObjectFromV0List(ctx, prior.ClientSideAvailability, featureFlagCSAAttrTypes)
				resp.Diagnostics.Append(d...)
				priorDefaults, d := defaultsObjectFromV0List(ctx, prior.Defaults)
				resp.Diagnostics.Append(d...)
				// v0 stored custom_properties as a set whose elements carried
				// the property key inline; v3 keys a map by property key.
				priorCustomProps, d := customPropertiesMapFromV0Set(ctx, prior.CustomProperties)
				resp.Diagnostics.Append(d...)
				if resp.Diagnostics.HasError() {
					return
				}
				data := FeatureFlagResourceModel{
					ID:                     prior.ID,
					ProjectKey:             prior.ProjectKey,
					Key:                    prior.Key,
					Name:                   prior.Name,
					Description:            nullIfEmptyString(prior.Description),
					MaintainerID:           prior.MaintainerID,
					MaintainerTeamKey:      prior.MaintainerTeamKey,
					Tags:                   nullIfEmptySet(ctx, prior.Tags),
					VariationType:          prior.VariationType,
					Variations:             prior.Variations,
					Temporary:              prior.Temporary,
					ClientSideAvailability: priorCSA,
					CustomProperties:       priorCustomProps,
					Defaults:               priorDefaults,
					Archived:               prior.Archived,
					Deprecated:             prior.Deprecated,
					ViewKeys:               nullIfEmptySet(ctx, prior.ViewKeys),
				}
				if featureFlagDefaultsMatchesAPIShape(ctx, data.Defaults, data.Variations) {
					data.Defaults = types.ObjectNull(featureFlagDefaultsAttrTypes)
				}
				// IIS->CSA migration: when prior state set include_in_snippet
				// and left client_side_availability empty, materialize the
				// CSA object so the resource still controls SDK availability.
				// using_mobile_key was never expressible via IIS — match the
				// Create-path projection (false). When both were populated,
				// the v2 Conflicting validator should have prevented it, so
				// drop IIS and keep CSA.
				csaEmpty := data.ClientSideAvailability.IsNull() || data.ClientSideAvailability.IsUnknown()
				iisSet := !prior.IncludeInSnippet.IsNull() && !prior.IncludeInSnippet.IsUnknown()
				if csaEmpty && iisSet {
					obj, d := types.ObjectValue(featureFlagCSAAttrTypes, map[string]attr.Value{
						USING_ENVIRONMENT_ID: types.BoolValue(prior.IncludeInSnippet.ValueBool()),
						USING_MOBILE_KEY:     types.BoolValue(false),
					})
					resp.Diagnostics.Append(d...)
					data.ClientSideAvailability = obj
				} else if featureFlagCSAMatchesAPIShape(ctx, data.ClientSideAvailability) {
					data.ClientSideAvailability = types.ObjectNull(featureFlagCSAAttrTypes)
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			},
		},
	}
}

func (r *FeatureFlagResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.Conflicting(
			path.MatchRoot(MAINTAINER_ID),
			path.MatchRoot(MAINTAINER_TEAM_KEY),
		),
	}
}

func (r *FeatureFlagResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

// Check for various 400/409 issues at plan time
// * Whether project requires view association on flag creation
// * Whether flag has dependent flags on flag deletion
func (r *FeatureFlagResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return
	}
	// Destroy plan: plan is null, state is not. Pre-flight a dependent-flag
	// check so users see the conflict at plan time instead of apply time
	// (issue #372).
	if req.Plan.Raw.IsNull() && !req.State.Raw.IsNull() {
		var state FeatureFlagResourceModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		projectKey, flagKey := state.ProjectKey.ValueString(), state.Key.ValueString()
		if projectKey == "" || flagKey == "" {
			return
		}
		deps, err := getDependentFlags(ctx, r.client, projectKey, flagKey)
		if err != nil {
			// Non-Enterprise tokens 403 here. Degrade to a warning so the
			// destroy still proceeds; the existing apply-time 409 path
			// remains as defence-in-depth.
			resp.Diagnostics.AddWarning(
				fmt.Sprintf("could not check dependent flags for %q in project %q during plan", flagKey, projectKey),
				err.Error()+"\n\nApply may still fail with a 409 conflict if this flag is referenced as a prerequisite by other flags.",
			)
			return
		}
		if deps != nil && len(deps.Items) > 0 {
			// Advisory only. At plan time nothing has been destroyed yet, so
			// a plan that removes both this flag AND the prerequisite links
			// (e.g. a whole-stack destroy, or one that updates the dependent
			// flags' launchdarkly_feature_flag_environment) is legitimate and
			// will succeed. The authoritative gate is the apply-time DELETE,
			// which returns a 409 if the references still exist. Warn so the
			// user sees the dependency early without blocking valid destroys.
			resp.Diagnostics.AddWarning(
				fmt.Sprintf("flag %q in project %q is referenced as a prerequisite by other flags", flagKey, projectKey),
				formatDependentFlagsHint(deps.Items)+"\n\nIf those references still exist when this destroy is applied, the apply will fail with a 409 conflict. If the same plan also removes them, the destroy will succeed.",
			)
		}
		return
	}
	if req.Plan.Raw.IsNull() {
		return
	}
	if !req.State.Raw.IsNull() {
		return
	}
	var plan FeatureFlagResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectKey := plan.ProjectKey.ValueString()
	if projectKey == "" {
		return
	}
	settings, err := getProjectViewSettings(ctx, r.client, projectKey)
	if err != nil {
		resp.Diagnostics.AddWarning(
			fmt.Sprintf("could not fetch project view settings for %q during plan", projectKey),
			err.Error(),
		)
		return
	}
	if !settings.RequireViewAssociationForNewFlags {
		return
	}
	if plan.ViewKeys.IsNull() || plan.ViewKeys.IsUnknown() || len(plan.ViewKeys.Elements()) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root(VIEW_KEYS),
			fmt.Sprintf("project %q requires new flags to be associated with at least one view. Please set the 'view_keys' attribute", projectKey),
			"",
		)
	}
}

func (r *FeatureFlagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FeatureFlagResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectKey := plan.ProjectKey.ValueString()
	key := plan.Key.ValueString()

	if exists, err := projectExists(projectKey, r.client); !exists {
		if err != nil {
			resp.Diagnostics.AddError(err.Error(), "")
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("cannot find project with key %q", projectKey), "")
		return
	}

	// Pre-flight check: if a flag with this key already exists on the
	// server in an archived state, refuse to create. This avoids a
	// confusing 409 from POST /flags and gives the user an actionable
	// next step. Typically happens when archive_flags_on_destroy=true
	// archived the flag in a previous run.
	var existing *ldapi.FeatureFlag
	var existingRes *http.Response
	if err := r.client.withConcurrency(ctx, func() error {
		f, res, e := r.client.ld.FeatureFlagsApi.GetFeatureFlag(r.client.ctx, projectKey, key).Execute()
		existing, existingRes = f, res
		return e
	}); err != nil && !isStatusNotFound(existingRes) {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to check for existing flag %q in project %q: %s", key, projectKey, handleLdapiErr(err).Error()), "")
		return
	}
	if existing != nil && existing.Archived {
		resp.Diagnostics.AddError(
			fmt.Sprintf("flag %q already exists in project %q in an archived state", key, projectKey),
			fmt.Sprintf("LaunchDarkly retains archived flags and their keys. To bring this flag back under Terraform management, import it first:\n\n  terraform import <resource_address> %s/%s\n\nThen unarchive it by setting `archived = false` (or remove the attribute) and re-apply. This typically happens when `archive_flags_on_destroy = true` archived the flag in a previous run.", projectKey, key),
		)
		return
	}

	desc := plan.Description.ValueString()
	tags, d := stringSliceFromSet(ctx, plan.Tags)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	temporary := plan.Temporary.ValueBool()
	variationType := plan.VariationType.ValueString()
	variations, d := variationsFromList(ctx, plan.Variations, variationType)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if variationType == BOOL_VARIATION && len(variations) == 0 {
		t, f := true, false
		variations = []ldapi.Variation{{Value: &t}, {Value: &f}}
	}
	defaults, d := defaultsFromObject(ctx, plan.Defaults)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	csaPlanned := !plan.ClientSideAvailability.IsNull() && !plan.ClientSideAvailability.IsUnknown()

	var finalCSA *ldapi.ClientSideAvailabilityPost
	if csaPlanned {
		csa, d := csaPostFromObject(ctx, plan.ClientSideAvailability)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		finalCSA = csa
	} else {
		defaultCSA, err := getProjectDefaultCSA(r.client, projectKey)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("failed to get project level client side availability defaults. %s", err.Error()), "")
			return
		}
		finalCSA = &ldapi.ClientSideAvailabilityPost{
			UsingEnvironmentId: *defaultCSA.UsingEnvironmentId,
			UsingMobileKey:     *defaultCSA.UsingMobileKey,
		}
	}

	viewKeys, d := stringSliceFromSet(ctx, plan.ViewKeys)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	var err error
	if len(viewKeys) > 0 {
		body := FeatureFlagBodyWithViewKeys{
			Name:                   plan.Name.ValueString(),
			Key:                    key,
			Description:            desc,
			Variations:             variations,
			Temporary:              temporary,
			Tags:                   tags,
			Defaults:               defaults,
			ClientSideAvailability: finalCSA,
			ViewKeys:               viewKeys,
		}
		err = r.client.withConcurrency(ctx, func() error {
			return createFeatureFlagWithViewKeys(ctx, r.client, projectKey, body)
		})
	} else {
		body := ldapi.FeatureFlagBody{
			Name:                   plan.Name.ValueString(),
			Key:                    key,
			Description:            &desc,
			Variations:             variations,
			Temporary:              &temporary,
			Tags:                   tags,
			Defaults:               defaults,
			ClientSideAvailability: finalCSA,
		}
		err = r.client.withConcurrency(ctx, func() error {
			_, _, e := r.client.ld.FeatureFlagsApi.PostFeatureFlag(r.client.ctx, projectKey).FeatureFlagBody(body).Execute()
			return e
		})
	}
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to create flag %q in project %q: %s", key, projectKey, handleLdapiErr(err).Error()), "")
		return
	}

	if d := r.applyFlagUpdate(ctx, plan, FeatureFlagResourceModel{}, true); d.HasError() {
		// Roll back the flag on update failure.
		_ = r.client.withConcurrency(ctx, func() error {
			_, e := r.client.ld.FeatureFlagsApi.DeleteFeatureFlag(r.client.ctx, projectKey, key).Execute()
			return e
		})
		resp.Diagnostics.Append(d...)
		return
	}
	r.readIntoModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.StringValue(projectKey + "/" + key)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FeatureFlagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FeatureFlagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readIntoModel(ctx, &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FeatureFlagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state FeatureFlagResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if d := r.applyFlagUpdate(ctx, plan, state, false); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}
	r.readIntoModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.StringValue(plan.ProjectKey.ValueString() + "/" + plan.Key.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FeatureFlagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FeatureFlagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectKey := data.ProjectKey.ValueString()
	key := data.Key.ValueString()

	if r.client.archiveFlagsOnDestroy {
		patch := []ldapi.PatchOperation{patchReplace("/archived", true)}
		err := r.client.withConcurrency(ctx, func() error {
			_, _, e := r.client.ld.FeatureFlagsApi.PatchFeatureFlag(r.client.ctx, projectKey, key).PatchWithComment(ldapi.PatchWithComment{Patch: patch}).Execute()
			return e
		})
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("failed to archive flag %q in project %q: %s", key, projectKey, handleLdapiErr(err).Error()), "")
		}
		return
	}

	err := r.client.withConcurrency(ctx, func() error {
		_, e := r.client.ld.FeatureFlagsApi.DeleteFeatureFlag(r.client.ctx, projectKey, key).Execute()
		return e
	})
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to delete flag %q from project %q: %s", key, projectKey, handleLdapiErr(err).Error()), "")
	}
}

func (r *FeatureFlagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectKey, flagKey, err := flagIdToKeys(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), projectKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(KEY), flagKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *FeatureFlagResource) applyFlagUpdate(ctx context.Context, plan, state FeatureFlagResourceModel, isCreate bool) diag.Diagnostics {
	var diags diag.Diagnostics
	projectKey := plan.ProjectKey.ValueString()
	key := plan.Key.ValueString()
	desc := plan.Description.ValueString()

	tags, d := stringSliceFromSet(ctx, plan.Tags)
	diags.Append(d...)
	customProps, d := customPropertiesFromMap(ctx, plan.CustomProperties)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	comment := "Terraform"
	patch := ldapi.PatchWithComment{
		Comment: &comment,
		Patch: []ldapi.PatchOperation{
			patchReplace("/name", plan.Name.ValueString()),
			patchReplace("/description", desc),
			patchReplace("/tags", tags),
			patchReplace("/temporary", plan.Temporary.ValueBool()),
			patchReplace("/customProperties", customProps),
			patchReplace("/archived", plan.Archived.ValueBool()),
			patchReplace("/deprecated", plan.Deprecated.ValueBool()),
		},
	}

	csaChanged := isCreate || !plan.ClientSideAvailability.Equal(state.ClientSideAvailability)
	csaPlanned := !plan.ClientSideAvailability.IsNull() && !plan.ClientSideAvailability.IsUnknown()

	if csaPlanned && csaChanged && !isCreate {
		csa, d := csaPostFromObject(ctx, plan.ClientSideAvailability)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
		patch.Patch = append(patch.Patch, patchReplace("/clientSideAvailability", csa))
	}

	if !isCreate {
		variationType := plan.VariationType.ValueString()
		variationPatches, d := variationPatchesFromLists(ctx, state.Variations, plan.Variations, variationType)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
		patch.Patch = append(patch.Patch, variationPatches...)
	}

	defaults, d := defaultsFromObject(ctx, plan.Defaults)
	diags.Append(d...)
	if defaults != nil {
		patch.Patch = append(patch.Patch, patchReplace("/defaults", defaults))
	}

	maintainerChanged := isCreate || !plan.MaintainerID.Equal(state.MaintainerID) || !plan.MaintainerTeamKey.Equal(state.MaintainerTeamKey)
	if maintainerChanged {
		flag, _, fErr := r.client.ld.FeatureFlagsApi.GetFeatureFlag(r.client.ctx, projectKey, key).Execute()
		if fErr == nil {
			mID := plan.MaintainerID.ValueString()
			mTeam := plan.MaintainerTeamKey.ValueString()
			switch {
			case mID != "":
				patch.Patch = append(patch.Patch, patchReplace("/maintainerId", mID))
				if flag != nil && flag.MaintainerTeamKey != nil {
					patch.Patch = append(patch.Patch, patchRemove("/maintainerTeamKey"))
				}
			case mTeam != "":
				patch.Patch = append(patch.Patch, patchReplace("/maintainerTeamKey", mTeam))
				if flag != nil && flag.MaintainerId != nil {
					patch.Patch = append(patch.Patch, patchRemove("/maintainerId"))
				}
			}
		}
	}

	err := r.client.withConcurrency(ctx, func() error {
		_, _, e := r.client.ld.FeatureFlagsApi.PatchFeatureFlag(r.client.ctx, projectKey, key).PatchWithComment(patch).Execute()
		return e
	})
	if err != nil {
		diags.AddError(fmt.Sprintf("failed to update flag %q in project %q: %s", key, projectKey, handleLdapiErr(err).Error()), "")
		return diags
	}

	// View association reconciliation.
	viewKeysChanged := isCreate || !plan.ViewKeys.Equal(state.ViewKeys)
	if !viewKeysChanged {
		return diags
	}
	desiredViews, d := stringSliceFromSet(ctx, plan.ViewKeys)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	betaClient, err := newBetaClient(r.client.apiKey, r.client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		diags.AddError(fmt.Sprintf("failed to create beta client for view linking: %v", err), "")
		return diags
	}

	if plan.ViewKeys.IsNull() || plan.ViewKeys.IsUnknown() {
		if isCreate {
			return diags
		}
		oldKeys, _ := stringSliceFromSet(ctx, state.ViewKeys)
		for _, vk := range oldKeys {
			if err := unlinkResourcesFromView(betaClient, projectKey, vk, FLAGS, []string{key}); err != nil {
				diags.AddError(fmt.Sprintf("failed to unlink flag %q from view %q: %v", key, vk, err), "")
				return diags
			}
		}
		return diags
	}

	for _, vk := range desiredViews {
		exists, vErr := viewExists(projectKey, vk, betaClient)
		if vErr != nil {
			diags.AddError(fmt.Sprintf("failed to check if view %q exists: %v", vk, vErr), "")
			return diags
		}
		if !exists {
			diags.AddError(fmt.Sprintf("cannot link flag to view %q in project %q: view does not exist", vk, projectKey), "")
			return diags
		}
	}
	currentViews, vErr := getViewsContainingFlag(betaClient, projectKey, key)
	if vErr != nil {
		log.Printf("[WARN] failed to get current views for flag %q: %v", key, vErr)
		currentViews = []string{}
	}
	toAdd := difference(desiredViews, currentViews)
	toRemove := difference(currentViews, desiredViews)
	if !isCreate {
		for _, vk := range toRemove {
			if err := unlinkResourcesFromView(betaClient, projectKey, vk, FLAGS, []string{key}); err != nil {
				diags.AddError(fmt.Sprintf("failed to unlink flag %q from view %q: %v", key, vk, err), "")
				return diags
			}
		}
	}
	for _, vk := range toAdd {
		if err := linkResourcesToView(betaClient, projectKey, vk, FLAGS, []string{key}); err != nil {
			diags.AddError(fmt.Sprintf("failed to link flag %q to view %q: %v", key, vk, err), "")
			return diags
		}
	}
	return diags
}

func (r *FeatureFlagResource) readIntoModel(ctx context.Context, data *FeatureFlagResourceModel, diags *diag.Diagnostics) {
	projectKey := data.ProjectKey.ValueString()
	key := data.Key.ValueString()

	var flag *ldapi.FeatureFlag
	var res *http.Response
	var err error
	err = r.client.withConcurrency(r.client.ctx, func() error {
		flag, res, err = r.client.ld.FeatureFlagsApi.GetFeatureFlag(r.client.ctx, projectKey, key).Execute()
		return err
	})
	if isStatusNotFound(res) {
		data.ID = types.StringNull()
		return
	}
	if err != nil {
		diags.AddError(fmt.Sprintf("failed to get flag %q of project %q: %s", key, projectKey, handleLdapiErr(err).Error()), "")
		return
	}

	data.ID = types.StringValue(projectKey + "/" + key)
	data.Key = types.StringValue(flag.Key)
	data.Name = types.StringValue(flag.Name)
	data.Description = stringValueOrNullFromPointer(flag.Description)
	data.Temporary = types.BoolValue(flag.Temporary)
	data.Archived = types.BoolValue(flag.Archived)
	data.Deprecated = types.BoolValue(flag.GetDeprecated())

	tagsSet, d := setFromStringSliceOrNull(ctx, flag.Tags)
	diags.Append(d...)
	data.Tags = tagsSet

	csaObj, d := featureFlagCSAObjectFromAPI(ctx, flag.ClientSideAvailability, data.ClientSideAvailability)
	diags.Append(d...)
	data.ClientSideAvailability = csaObj

	// Maintainer fields — Optional+Computed. Only write these to state
	// when the user declares either maintainer_id or maintainer_team_key:
	// if either attribute is managed, emit both API values (using "" for
	// nil pointers so TestCheckResourceAttr(..., "") asserts pass); if
	// neither is managed, emit null on both so TestCheckNoResourceAttr
	// asserts pass.
	maintainerManaged := priorMaintainerSet(data.MaintainerID) || priorMaintainerSet(data.MaintainerTeamKey)
	if maintainerManaged {
		data.MaintainerID = stringValueOrEmpty(flag.MaintainerId)
		data.MaintainerTeamKey = stringValueOrEmpty(flag.MaintainerTeamKey)
	} else {
		data.MaintainerID = types.StringNull()
		data.MaintainerTeamKey = types.StringNull()
	}

	// Variations
	variationType, vErr := variationsToVariationType(flag.Variations)
	if vErr != nil {
		diags.AddError(fmt.Sprintf("failed to determine variation type on flag with key %q: %v", flag.Key, vErr), "")
		return
	}
	data.VariationType = types.StringValue(variationType)
	variationsList, d := variationsListFromAPI(ctx, flag.Variations, variationType, data.Variations)
	diags.Append(d...)
	data.Variations = variationsList

	// Custom properties — keyed by property key, sorted values.
	cpMap, d := customPropertiesMapFromAPI(ctx, flag.CustomProperties)
	diags.Append(d...)
	data.CustomProperties = cpMap

	// Defaults
	defaultsObj, d := defaultsObjectFromAPI(ctx, flag.Defaults, len(flag.Variations), data.Defaults)
	diags.Append(d...)
	data.Defaults = defaultsObj

	// View associations — best-effort.
	betaClient, bcErr := newBetaClient(r.client.apiKey, r.client.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if bcErr != nil {
		log.Printf("[WARN] failed to create beta client for views lookup: %v", bcErr)
		data.ViewKeys = types.SetValueMust(types.StringType, []attr.Value{})
		return
	}
	viewKeys, vErr := getViewsContainingFlag(betaClient, projectKey, key)
	if vErr != nil {
		log.Printf("[WARN] failed to get views for flag %q in project %q: %v", key, projectKey, vErr)
		viewKeys = []string{}
	}
	viewKeysSet, d := setFromStringSlice(ctx, viewKeys)
	diags.Append(d...)
	data.ViewKeys = viewKeysSet
}

// featureFlagCSAObjectFromAPI flattens LD's ClientSideAvailability into
// the single-object shape used by the framework schema. Mirrors the
// prior state's attribute presence: emit null when the user did not
// declare the attribute, populated when they did.
func featureFlagCSAObjectFromAPI(_ context.Context, csa *ldapi.ClientSideAvailability, prior types.Object) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	priorEmpty := prior.IsNull() || prior.IsUnknown()
	if priorEmpty || csa == nil {
		return types.ObjectNull(featureFlagCSAAttrTypes), diags
	}
	usingEnv := false
	if csa.UsingEnvironmentId != nil {
		usingEnv = *csa.UsingEnvironmentId
	}
	usingMobile := false
	if csa.UsingMobileKey != nil {
		usingMobile = *csa.UsingMobileKey
	}
	obj, d := types.ObjectValue(featureFlagCSAAttrTypes, map[string]attr.Value{
		USING_ENVIRONMENT_ID: types.BoolValue(usingEnv),
		USING_MOBILE_KEY:     types.BoolValue(usingMobile),
	})
	diags.Append(d...)
	return obj, diags
}

// variationsFromList converts a framework List<variation> into
// ldapi.Variation slices, parsing the typed `value` string per
// variation_type.
func variationsFromList(ctx context.Context, list types.List, variationType string) ([]ldapi.Variation, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return []ldapi.Variation{}, diags
	}
	type variationModel struct {
		Name        types.String `tfsdk:"name"`
		Description types.String `tfsdk:"description"`
		Value       types.String `tfsdk:"value"`
	}
	var models []variationModel
	d := list.ElementsAs(ctx, &models, false)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	if variationType != BOOL_VARIATION && len(models) < 2 {
		diags.AddError("invalid variations", "multivariate flags must have at least two variations defined")
		return nil, diags
	}
	out := make([]ldapi.Variation, 0, len(models))
	for _, m := range models {
		v, err := variationFromTypedValue(m.Name.ValueString(), m.Description.ValueString(), m.Value.ValueString(), variationType)
		if err != nil {
			diags.AddError("invalid variation", err.Error())
			return nil, diags
		}
		out = append(out, v)
	}
	return out, diags
}

func variationFromTypedValue(name, description, value, variationType string) (ldapi.Variation, error) {
	v := ldapi.Variation{}
	switch variationType {
	case BOOL_VARIATION:
		b := value == "true"
		v.Value = &b
	case STRING_VARIATION:
		var s interface{} = value
		v.Value = &s
	case NUMBER_VARIATION:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return v, fmt.Errorf("%q is an invalid number variation value. %v", value, err)
		}
		v.Value = &f
	case JSON_VARIATION:
		var raw interface{}
		if err := json.Unmarshal([]byte(value), &raw); err != nil {
			return v, fmt.Errorf("%q is an invalid json variation value. %v", value, err)
		}
		v.Value = &raw
	default:
		return v, fmt.Errorf("invalid variation type: %q", variationType)
	}
	if name != "" {
		v.Name = &name
	}
	if description != "" {
		v.Description = &description
	}
	return v, nil
}

// variationPatchesFromLists computes patch operations for the diff
// between two variations lists.
func variationPatchesFromLists(ctx context.Context, oldList, newList types.List, variationType string) ([]ldapi.PatchOperation, diag.Diagnostics) {
	var diags diag.Diagnostics
	var patches []ldapi.PatchOperation
	if oldList.IsNull() || oldList.IsUnknown() {
		return patches, diags
	}
	oldVariations, d := variationsFromList(ctx, oldList, variationType)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	newVariations, d := variationsFromList(ctx, newList, variationType)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	for idx := len(newVariations); idx < len(oldVariations); idx++ {
		patches = append(patches, patchRemove(fmt.Sprintf("/variations/%d", idx)))
	}
	for idx, v := range newVariations {
		if idx < len(oldVariations) {
			patches = append(patches, patchReplace(fmt.Sprintf("/variations/%d/value", idx), v.Value))
			// name and description are Optional+Computed. Only patch them when the config sets a value.
			// Replacing with a nil value clears a name/description set outside Terraform — the LD API
			// treats a null replace as a clear. Omitting the patch leaves the server value as-is, which
			// is the expected "omit = preserve" behavior for a computed attribute and keeps the migration
			// lossless for boolean flags whose variations were named in the UI. To clear a name, set it
			// explicitly to "" is not supported (the provider treats "" as unset); clear it in the UI.
			if v.Name != nil {
				patches = append(patches, patchReplace(fmt.Sprintf("/variations/%d/name", idx), v.Name))
			}
			if v.Description != nil {
				patches = append(patches, patchReplace(fmt.Sprintf("/variations/%d/description", idx), v.Description))
			}
		} else {
			patches = append(patches, patchAdd(fmt.Sprintf("/variations/%d", idx), v))
		}
	}
	return patches, diags
}

// variationsListFromAPI flattens LD-API variations into a framework
// List<variation>. Returns null when the API has no variations
// (e.g. mid-create or for not-yet-saved flags).
//
// For JSON-typed variations, the LD API normalises the stored value
// (e.g. collapses whitespace) which would otherwise diverge from the
// HCL-supplied string. When the prior value at the same index is
// semantically equal JSON, re-emit the prior string so plan/state stay
// aligned. Variation name/description fall back to "" so the
// schema-level Default+Computed semantics carry through Read.
func variationsListFromAPI(ctx context.Context, variations []ldapi.Variation, variationType string, prior types.List) (types.List, diag.Diagnostics) {
	objType := types.ObjectType{AttrTypes: featureFlagVariationAttrTypes}
	var diags diag.Diagnostics
	if len(variations) == 0 {
		return types.ListNull(objType), diags
	}
	priorByIdx := variationPriorByIndex(ctx, prior, &diags)
	elements := make([]attr.Value, 0, len(variations))
	for i, v := range variations {
		valueStr, err := variationValueToString(&v.Value, variationType)
		if err != nil {
			diags.AddError("failed to serialise variation value", err.Error())
			return types.ListNull(objType), diags
		}
		if variationType == JSON_VARIATION && i < len(priorByIdx) {
			priorVal := priorByIdx[i].Value
			if !priorVal.IsNull() && !priorVal.IsUnknown() && jsonSemanticallyEqual(priorVal.ValueString(), valueStr) {
				valueStr = priorVal.ValueString()
			}
		}
		nameVal := types.StringValue("")
		if v.Name != nil {
			nameVal = types.StringValue(*v.Name)
		}
		descVal := types.StringValue("")
		if v.Description != nil {
			descVal = types.StringValue(*v.Description)
		}
		obj, d := types.ObjectValue(featureFlagVariationAttrTypes, map[string]attr.Value{
			NAME:        nameVal,
			DESCRIPTION: descVal,
			VALUE:       types.StringValue(valueStr),
		})
		diags.Append(d...)
		elements = append(elements, obj)
	}
	list, d := types.ListValue(objType, elements)
	diags.Append(d...)
	return list, diags
}

// variationPriorView is the slim subset of a variation we read off the
// prior plan/state when reconciling JSON-equivalence on Read.
type variationPriorView struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Value       types.String `tfsdk:"value"`
}

func variationPriorByIndex(ctx context.Context, prior types.List, diags *diag.Diagnostics) []variationPriorView {
	if prior.IsNull() || prior.IsUnknown() {
		return nil
	}
	var rows []variationPriorView
	d := prior.ElementsAs(ctx, &rows, false)
	diags.Append(d...)
	if diags.HasError() {
		return nil
	}
	return rows
}

// jsonSemanticallyEqual reports whether two strings parse to equal JSON
// documents. Mirrors framework_json_helpers.go's jsonNormalizePlanModifier
// but operates inline during Read so we can preserve the user-formatted
// string when the API normalises whitespace.
func jsonSemanticallyEqual(a, b string) bool {
	if a == b {
		return true
	}
	if a == "" || b == "" {
		return false
	}
	var aj, bj interface{}
	if err := json.Unmarshal([]byte(a), &aj); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &bj); err != nil {
		return false
	}
	return reflect.DeepEqual(aj, bj)
}

// customPropertyModel matches featureFlagCustomPropertyAttrTypes.
type customPropertyModel struct {
	Key   types.String `tfsdk:"key"`
	Name  types.String `tfsdk:"name"`
	Value types.List   `tfsdk:"value"`
}

// ValidateConfig enforces that each custom property's `key` (when set)
// equals its map key. The map key is the authoritative identity; a
// per-attribute validator can't see its own map key, so the cross-check
// lives here.
func (r *FeatureFlagResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config FeatureFlagResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if config.CustomProperties.IsNull() || config.CustomProperties.IsUnknown() {
		return
	}
	models := map[string]customPropertyModel{}
	resp.Diagnostics.Append(config.CustomProperties.ElementsAs(ctx, &models, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	for mapKey, cp := range models {
		if cp.Key.IsNull() || cp.Key.IsUnknown() {
			continue
		}
		if cp.Key.ValueString() != mapKey {
			resp.Diagnostics.AddAttributeError(
				path.Root(CUSTOM_PROPERTIES).AtMapKey(mapKey).AtName(KEY),
				"custom property key must match its map key",
				fmt.Sprintf("custom property %q sets key = %q; the nested `key` must equal the map key (or be omitted). The map key is the custom property's identity.", mapKey, cp.Key.ValueString()),
			)
		}
	}
}

// customPropertiesFromMap converts the framework Map<custom_property>
// (keyed by the custom property key) into the API's
// map[string]ldapi.CustomProperty. Values are sorted before sending to
// the API for stable diffs. The map key is authoritative; the inner
// `key` attribute is validated to match in ValidateConfig.
func customPropertiesFromMap(ctx context.Context, m types.Map) (map[string]ldapi.CustomProperty, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := map[string]ldapi.CustomProperty{}
	if m.IsNull() || m.IsUnknown() {
		return out, diags
	}
	models := map[string]customPropertyModel{}
	d := m.ElementsAs(ctx, &models, false)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	for k, cp := range models {
		values, d := stringSliceFromList(ctx, cp.Value)
		diags.Append(d...)
		sort.Strings(values)
		out[k] = ldapi.CustomProperty{
			Name:  cp.Name.ValueString(),
			Value: values,
		}
	}
	return out, diags
}

// customPropertiesMapFromAPI converts the LD-API map[string]CustomProperty
// into the framework Map<custom_property> keyed by property key, sorting
// values for stable diffs. The inner `key` attribute always equals the
// map key.
func customPropertiesMapFromAPI(ctx context.Context, props map[string]ldapi.CustomProperty) (types.Map, diag.Diagnostics) {
	objType := types.ObjectType{AttrTypes: featureFlagCustomPropertyAttrTypes}
	var diags diag.Diagnostics
	if len(props) == 0 {
		return types.MapNull(objType), diags
	}
	elements := make(map[string]attr.Value, len(props))
	for k, cp := range props {
		sortedValues := make([]string, len(cp.Value))
		copy(sortedValues, cp.Value)
		sort.Strings(sortedValues)
		valuesList, d := listFromStringSlice(ctx, sortedValues)
		diags.Append(d...)
		obj, d := types.ObjectValue(featureFlagCustomPropertyAttrTypes, map[string]attr.Value{
			KEY:   types.StringValue(k),
			NAME:  types.StringValue(cp.Name),
			VALUE: valuesList,
		})
		diags.Append(d...)
		elements[k] = obj
	}
	result, d := types.MapValue(objType, elements)
	diags.Append(d...)
	return result, diags
}

// customPropertiesMapFromV0Set projects the v0 (SDKv2 / pre-map)
// Set<custom_property> — whose elements carried the property key inline —
// into the current map keyed by property key. Returns a null map for
// null/empty input.
func customPropertiesMapFromV0Set(ctx context.Context, set types.Set) (types.Map, diag.Diagnostics) {
	objType := types.ObjectType{AttrTypes: featureFlagCustomPropertyAttrTypes}
	var diags diag.Diagnostics
	if set.IsNull() || set.IsUnknown() || len(set.Elements()) == 0 {
		return types.MapNull(objType), diags
	}
	var models []customPropertyModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return types.MapNull(objType), diags
	}
	elements := make(map[string]attr.Value, len(models))
	for _, cp := range models {
		obj, d := types.ObjectValue(featureFlagCustomPropertyAttrTypes, map[string]attr.Value{
			KEY:   cp.Key,
			NAME:  cp.Name,
			VALUE: cp.Value,
		})
		diags.Append(d...)
		elements[cp.Key.ValueString()] = obj
	}
	result, d := types.MapValue(objType, elements)
	diags.Append(d...)
	return result, diags
}

// defaultsFromObject converts the framework defaults object into
// *ldapi.Defaults. Returns nil when the object is null/unknown.
func defaultsFromObject(ctx context.Context, obj types.Object) (*ldapi.Defaults, diag.Diagnostics) {
	var diags diag.Diagnostics
	if obj.IsNull() || obj.IsUnknown() {
		return nil, diags
	}
	type defaultsModel struct {
		OnVariation  types.Int64 `tfsdk:"on_variation"`
		OffVariation types.Int64 `tfsdk:"off_variation"`
	}
	var m defaultsModel
	d := obj.As(ctx, &m, basetypes.ObjectAsOptions{})
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	return &ldapi.Defaults{
		OnVariation:  int32(m.OnVariation.ValueInt64()),
		OffVariation: int32(m.OffVariation.ValueInt64()),
	}, diags
}

// defaultsObjectFromAPI flattens LD-API Defaults into the single-object
// shape. Mirrors prior-state attribute presence: emit null when the
// user did not declare `defaults`, populated when they did.
func defaultsObjectFromAPI(_ context.Context, defaults *ldapi.Defaults, variationCount int, prior types.Object) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	priorEmpty := prior.IsNull() || prior.IsUnknown()
	if priorEmpty {
		return types.ObjectNull(featureFlagDefaultsAttrTypes), diags
	}
	var on, off int64
	if defaults != nil {
		on = int64(defaults.OnVariation)
		off = int64(defaults.OffVariation)
	} else {
		on = 0
		off = int64(variationCount - 1)
		if off < 0 {
			off = 0
		}
	}
	obj, d := types.ObjectValue(featureFlagDefaultsAttrTypes, map[string]attr.Value{
		ON_VARIATION:  types.Int64Value(on),
		OFF_VARIATION: types.Int64Value(off),
	})
	diags.Append(d...)
	return obj, diags
}

// priorMaintainerSet returns true when the prior plan/state had a
// concrete value for a maintainer attribute (i.e. the user managed it).
// Unknown / Null / empty string mean the user never declared it; we
// suppress those from state so unmanaged maintainer_id / team_key stay
// absent.
func priorMaintainerSet(v types.String) bool {
	return !v.IsNull() && !v.IsUnknown() && v.ValueString() != ""
}

// stringValueOrEmpty returns the API value as types.String, emitting
// "" rather than null for nil pointers. This satisfies
// TestCheckResourceAttr(..., "") in tests where the API returns nil
// for an attribute the user is otherwise managing (e.g. maintainer_id
// when only maintainer_team_key was set).
func stringValueOrEmpty(p *string) types.String {
	if p == nil {
		return types.StringValue("")
	}
	return types.StringValue(*p)
}

// Suppress unused-import warning for strings when no strings.* used.
var _ = strings.Count
