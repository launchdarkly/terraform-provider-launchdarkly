package launchdarkly

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var (
	_ resource.Resource                     = &FeatureFlagResource{}
	_ resource.ResourceWithImportState      = &FeatureFlagResource{}
	_ resource.ResourceWithModifyPlan       = &FeatureFlagResource{}
	_ resource.ResourceWithConfigValidators = &FeatureFlagResource{}
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
	IncludeInSnippet       types.Bool   `tfsdk:"include_in_snippet"`
	ClientSideAvailability types.List   `tfsdk:"client_side_availability"`
	CustomProperties       types.Set    `tfsdk:"custom_properties"`
	Defaults               types.List   `tfsdk:"defaults"`
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
		Description: `Provides a LaunchDarkly feature flag resource.

This resource allows you to create and manage feature flags within your LaunchDarkly organization.

-> **Note:** This resource is for global-level feature flag configuration. Unexpected behavior may result if your environment-level configurations are not also managed from Terraform.`,
		Attributes: map[string]schema.Attribute{
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
			},
			MAINTAINER_TEAM_KEY: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The key of the associated team that maintains this feature flag. `maintainer_id` cannot be set if `maintainer_team_key` is set",
				Validators:  []validator.String{keyAndLengthValidator(1, 256)},
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
			INCLUDE_IN_SNIPPET: schema.BoolAttribute{
				Optional:           true,
				Computed:           true,
				Description:        "Specifies whether this flag should be made available to the client-side JavaScript SDK using the client-side Id. This value gets its default from your project configuration if not set. `include_in_snippet` is now deprecated. Please migrate to `client_side_availability.using_environment_id` to maintain future compatibility.",
				DeprecationMessage: "'include_in_snippet' is now deprecated. Please migrate to 'client_side_availability' to maintain future compatability.",
			},
			TAGS: schema.SetAttribute{
				Optional:    true,
				Computed:    true,
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
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "A set of view keys to link this flag to. This is an alternative to using the `launchdarkly_view_links` resource for managing view associations. When set, this flag will be linked to the specified views. The field is also computed, meaning Terraform will read back the current view associations from LaunchDarkly to detect drift. To explicitly remove all view associations, set `view_keys = []`. Simply removing the field from your configuration will leave existing associations unchanged. **Important**: Avoid using both `view_keys` and `launchdarkly_view_links` to manage the same flag. Mixed ownership can cause conflicts; when detected, Terraform logs a warning and reconciles to the configured `view_keys`. Choose one approach per resource.",
			},
		},
		Blocks: map[string]schema.Block{
			VARIATIONS: schema.ListNestedBlock{
				Description: "An array of possible variations for the flag",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						NAME: schema.StringAttribute{
							Optional:    true,
							Description: "The name of the variation.",
						},
						DESCRIPTION: schema.StringAttribute{
							Optional:    true,
							Description: "The variation's description.",
						},
						VALUE: schema.StringAttribute{
							Required: true,
							PlanModifiers: []planmodifier.String{
								jsonNormalizePlanModifier{},
							},
							Description: fmt.Sprintf("The variation value. The value's type must correspond to the `variation_type` argument. For example: `variation_type = %q` accepts only `true` or `false`. The `number` variation type accepts both floats and ints, but please note that any trailing zeroes on floats will be trimmed (i.e. `1.1` and `1.100` will both be converted to `1.1`).\n\nIf you wish to define an empty string variation, you must still define the value field on the variations block like so:\n\n```terraform\nvariations {\n  value = %q\n}\n```\n\n-> **Note:** Terraform manages `variations` as an ordered array and identifies them by index. This means that if you change the order of your `variations` block, you may end up destroying and recreating those variations. Additionally, if you delete variations that have targets that have been attached outside of Terraform, those targets may be incorrectly reassigned to a different variation.", "boolean", ""),
						},
					},
				},
			},
			CLIENT_SIDE_AVAILABILITY: schema.ListNestedBlock{
				Description: "A block describing whether this flag should be made available to the client-side JavaScript SDK using the client-side Id, mobile key, or both. This value gets its default from your project configuration if not set. Once set, if removed, it will retain its last set value.",
				Validators:  []validator.List{listvalidator.SizeAtMost(1)},
				NestedObject: schema.NestedBlockObject{
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
			},
			CUSTOM_PROPERTIES: schema.SetNestedBlock{
				Description: "List of nested blocks describing the feature flag's [custom properties](https://docs.launchdarkly.com/home/connecting/custom-properties)",
				Validators:  []validator.Set{setvalidator.SizeAtMost(CUSTOM_PROPERTY_ITEM_LIMIT)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						KEY: schema.StringAttribute{
							Required:    true,
							Description: "The unique custom property key.",
							Validators:  []validator.String{stringLenBetween(1, CUSTOM_PROPERTY_CHAR_LIMIT)},
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
			DEFAULTS: schema.ListNestedBlock{
				Description: "A block containing the indices of the variations to be used as the default on and off variations in all new environments. Flag configurations in existing environments will not be changed nor updated if the configuration block is removed.",
				Validators:  []validator.List{listvalidator.SizeAtMost(1)},
				NestedObject: schema.NestedBlockObject{
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
			},
		},
	}
}

func (r *FeatureFlagResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.Conflicting(
			path.MatchRoot(INCLUDE_IN_SNIPPET),
			path.MatchRoot(CLIENT_SIDE_AVAILABILITY),
		),
		resourcevalidator.Conflicting(
			path.MatchRoot(MAINTAINER_ID),
			path.MatchRoot(MAINTAINER_TEAM_KEY),
		),
	}
}

func (r *FeatureFlagResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

// ModifyPlan ports customizeFeatureFlagDiff: create-time view_keys
// validation when the project requires view association.
func (r *FeatureFlagResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	if r.client == nil {
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
	defaults, d := defaultsFromList(ctx, plan.Defaults)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	csaPlanned := !plan.ClientSideAvailability.IsNull() && !plan.ClientSideAvailability.IsUnknown() && len(plan.ClientSideAvailability.Elements()) > 0
	iisPlanned := !plan.IncludeInSnippet.IsNull() && !plan.IncludeInSnippet.IsUnknown()

	var finalCSA *ldapi.ClientSideAvailabilityPost
	switch {
	case csaPlanned:
		csa, d := csaPostFromList(ctx, plan.ClientSideAvailability)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		finalCSA = csa
	case iisPlanned:
		finalCSA = &ldapi.ClientSideAvailabilityPost{
			UsingEnvironmentId: plan.IncludeInSnippet.ValueBool(),
			UsingMobileKey:     false,
		}
	default:
		defaultCSA, _, err := getProjectDefaultCSAandIncludeInSnippet(r.client, projectKey)
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
	err := r.client.withConcurrency(ctx, func() error {
		_, e := r.client.ld.FeatureFlagsApi.DeleteFeatureFlag(r.client.ctx, data.ProjectKey.ValueString(), data.Key.ValueString()).Execute()
		return e
	})
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to delete flag %q from project %q: %s", data.Key.ValueString(), data.ProjectKey.ValueString(), handleLdapiErr(err).Error()), "")
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
	customProps, d := customPropertiesFromSet(ctx, plan.CustomProperties)
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
	iisChanged := isCreate || !plan.IncludeInSnippet.Equal(state.IncludeInSnippet)
	csaPlanned := !plan.ClientSideAvailability.IsNull() && !plan.ClientSideAvailability.IsUnknown() && len(plan.ClientSideAvailability.Elements()) > 0
	iisPlanned := !plan.IncludeInSnippet.IsNull() && !plan.IncludeInSnippet.IsUnknown()

	if csaPlanned && csaChanged && !isCreate {
		csa, d := csaPostFromList(ctx, plan.ClientSideAvailability)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
		patch.Patch = append(patch.Patch, patchReplace("/clientSideAvailability", csa))
	} else if iisPlanned && iisChanged && !isCreate {
		patch.Patch = append(patch.Patch, patchReplace("/clientSideAvailability", &ldapi.ClientSideAvailabilityPost{
			UsingEnvironmentId: plan.IncludeInSnippet.ValueBool(),
			UsingMobileKey:     false,
		}))
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

	defaults, d := defaultsFromList(ctx, plan.Defaults)
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

	// Import context: data.Name is null until this Read populates it.
	// On Import the prior-state-presence pattern emits empty for blocks
	// the user originally declared; force-emit from API instead.
	isImport := data.Name.IsNull()

	data.ID = types.StringValue(projectKey + "/" + key)
	data.Key = types.StringValue(flag.Key)
	data.Name = types.StringValue(flag.Name)
	data.Description = stringValueOrNullFromPointer(flag.Description)
	data.Temporary = types.BoolValue(flag.Temporary)
	data.Archived = types.BoolValue(flag.Archived)
	data.Deprecated = types.BoolValue(flag.GetDeprecated())

	tagsSet, d := setFromStringSlice(ctx, flag.Tags)
	diags.Append(d...)
	data.Tags = tagsSet

	// CSA + IIS — emit both so plan/state remains stable.
	priorCSA := data.ClientSideAvailability
	if isImport {
		// Synthetic populated prior so the helper emits populated.
		priorCSA = types.ListValueMust(
			types.ObjectType{AttrTypes: featureFlagCSAAttrTypes},
			[]attr.Value{types.ObjectValueMust(featureFlagCSAAttrTypes, map[string]attr.Value{
				USING_ENVIRONMENT_ID: types.BoolValue(false),
				USING_MOBILE_KEY:     types.BoolValue(false),
			})},
		)
	}
	csaList, d := featureFlagCSAListFromAPI(ctx, flag.ClientSideAvailability, priorCSA)
	diags.Append(d...)
	data.ClientSideAvailability = csaList
	usingEnvID := false
	if flag.ClientSideAvailability != nil && flag.ClientSideAvailability.UsingEnvironmentId != nil {
		usingEnvID = *flag.ClientSideAvailability.UsingEnvironmentId
	}
	data.IncludeInSnippet = types.BoolValue(usingEnvID)

	// Maintainer fields — Optional+Computed; mirror what LD returned.
	data.MaintainerID = stringValueOrNullFromPointer(flag.MaintainerId)
	data.MaintainerTeamKey = stringValueOrNullFromPointer(flag.MaintainerTeamKey)

	// Variations
	variationType, vErr := variationsToVariationType(flag.Variations)
	if vErr != nil {
		diags.AddError(fmt.Sprintf("failed to determine variation type on flag with key %q: %v", flag.Key, vErr), "")
		return
	}
	data.VariationType = types.StringValue(variationType)
	priorVariations := data.Variations
	if isImport {
		// Non-empty synthetic prior so the helper emits populated.
		priorVariations = types.ListValueMust(
			types.ObjectType{AttrTypes: featureFlagVariationAttrTypes},
			[]attr.Value{types.ObjectValueMust(featureFlagVariationAttrTypes, map[string]attr.Value{
				NAME:        types.StringNull(),
				DESCRIPTION: types.StringNull(),
				VALUE:       types.StringValue(""),
			})},
		)
	}
	variationsList, d := variationsListFromAPI(ctx, flag.Variations, variationType, priorVariations)
	diags.Append(d...)
	data.Variations = variationsList

	// Custom properties — sorted values, custom_properties hash parity.
	cpSet, d := customPropertiesSetFromAPI(ctx, flag.CustomProperties)
	diags.Append(d...)
	data.CustomProperties = cpSet

	// Defaults
	priorDefaults := data.Defaults
	if isImport {
		priorDefaults = types.ListValueMust(
			types.ObjectType{AttrTypes: featureFlagDefaultsAttrTypes},
			[]attr.Value{types.ObjectValueMust(featureFlagDefaultsAttrTypes, map[string]attr.Value{
				ON_VARIATION:  types.Int64Value(0),
				OFF_VARIATION: types.Int64Value(0),
			})},
		)
	}
	defaultsList, d := defaultsListFromAPI(ctx, flag.Defaults, len(flag.Variations), priorDefaults)
	diags.Append(d...)
	data.Defaults = defaultsList

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

// featureFlagCSAListFromAPI flattens LD's ClientSideAvailability into
// the single-element list shape used by the framework schema. Mirrors
// the prior state's block presence: emit empty when the user did not
// manage the block, populated when they did. Framework forbids
// block-level Computed (Phase 3 gotcha #3), so emitting a count=1
// block where config has count=0 fails terraform-core's consistency
// check.
func featureFlagCSAListFromAPI(ctx context.Context, csa *ldapi.ClientSideAvailability, prior types.List) (types.List, diag.Diagnostics) {
	objType := types.ObjectType{AttrTypes: featureFlagCSAAttrTypes}
	var diags diag.Diagnostics
	priorEmpty := prior.IsNull() || prior.IsUnknown() || len(prior.Elements()) == 0
	if priorEmpty || csa == nil {
		list, d := types.ListValue(objType, []attr.Value{})
		diags.Append(d...)
		return list, diags
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
	list, d := types.ListValue(objType, []attr.Value{obj})
	diags.Append(d...)
	return list, diags
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

// variationPatchesFromLists mirrors variationPatchesFromResourceData.
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
			patches = append(patches, patchReplace(fmt.Sprintf("/variations/%d/name", idx), v.Name))
			patches = append(patches, patchReplace(fmt.Sprintf("/variations/%d/description", idx), v.Description))
		} else {
			patches = append(patches, patchAdd(fmt.Sprintf("/variations/%d", idx), v))
		}
	}
	return patches, diags
}

// variationsListFromAPI flattens LD-API variations into a framework
// List<variation>. Value coercion mirrors variationsToResourceData.
// Mirrors prior-state block presence: when the user omits the
// variations block (SDKv2 Optional+Computed at TypeList level allowed
// auto-defaults for boolean flags), state remains empty even though
// the API populates the underlying variations.
func variationsListFromAPI(ctx context.Context, variations []ldapi.Variation, variationType string, prior types.List) (types.List, diag.Diagnostics) {
	objType := types.ObjectType{AttrTypes: featureFlagVariationAttrTypes}
	var diags diag.Diagnostics
	priorEmpty := prior.IsNull() || prior.IsUnknown() || len(prior.Elements()) == 0
	if priorEmpty {
		list, d := types.ListValue(objType, []attr.Value{})
		diags.Append(d...)
		return list, diags
	}
	elements := make([]attr.Value, 0, len(variations))
	for _, v := range variations {
		valueStr, err := variationValueToString(&v.Value, variationType)
		if err != nil {
			diags.AddError("failed to serialise variation value", err.Error())
			return types.ListNull(objType), diags
		}
		obj, d := types.ObjectValue(featureFlagVariationAttrTypes, map[string]attr.Value{
			NAME:        stringValueOrNullFromPointer(v.Name),
			DESCRIPTION: stringValueOrNullFromPointer(v.Description),
			VALUE:       types.StringValue(valueStr),
		})
		diags.Append(d...)
		elements = append(elements, obj)
	}
	list, d := types.ListValue(objType, elements)
	diags.Append(d...)
	return list, diags
}

// customPropertiesFromSet converts the framework Set<custom_property>
// into the API's map[string]ldapi.CustomProperty. Values are sorted
// before sending to the API (mirroring customPropertyFromResourceData).
func customPropertiesFromSet(ctx context.Context, set types.Set) (map[string]ldapi.CustomProperty, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := map[string]ldapi.CustomProperty{}
	if set.IsNull() || set.IsUnknown() {
		return out, diags
	}
	type cpModel struct {
		Key   types.String `tfsdk:"key"`
		Name  types.String `tfsdk:"name"`
		Value types.List   `tfsdk:"value"`
	}
	var models []cpModel
	d := set.ElementsAs(ctx, &models, false)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	for _, m := range models {
		values, d := stringSliceFromList(ctx, m.Value)
		diags.Append(d...)
		sort.Strings(values)
		out[m.Key.ValueString()] = ldapi.CustomProperty{
			Name:  m.Name.ValueString(),
			Value: values,
		}
	}
	return out, diags
}

// customPropertiesSetFromAPI converts the LD-API map[string]CustomProperty
// into the framework Set<custom_property>, sorting the values per
// custom_properties_helper parity.
func customPropertiesSetFromAPI(ctx context.Context, props map[string]ldapi.CustomProperty) (types.Set, diag.Diagnostics) {
	objType := types.ObjectType{AttrTypes: featureFlagCustomPropertyAttrTypes}
	var diags diag.Diagnostics
	elements := make([]attr.Value, 0, len(props))
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
		elements = append(elements, obj)
	}
	set, d := types.SetValue(objType, elements)
	diags.Append(d...)
	return set, diags
}

// defaultsFromList converts the framework List<defaults> into
// *ldapi.Defaults. Returns nil when the list is null/empty, matching
// SDKv2 defaultVariationsFromResourceData behaviour.
func defaultsFromList(ctx context.Context, list types.List) (*ldapi.Defaults, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() || len(list.Elements()) == 0 {
		return nil, diags
	}
	type defaultsModel struct {
		OnVariation  types.Int64 `tfsdk:"on_variation"`
		OffVariation types.Int64 `tfsdk:"off_variation"`
	}
	var models []defaultsModel
	d := list.ElementsAs(ctx, &models, false)
	diags.Append(d...)
	if diags.HasError() || len(models) == 0 {
		return nil, diags
	}
	return &ldapi.Defaults{
		OnVariation:  int32(models[0].OnVariation.ValueInt64()),
		OffVariation: int32(models[0].OffVariation.ValueInt64()),
	}, diags
}

func defaultsListFromAPI(ctx context.Context, defaults *ldapi.Defaults, variationCount int, prior types.List) (types.List, diag.Diagnostics) {
	objType := types.ObjectType{AttrTypes: featureFlagDefaultsAttrTypes}
	var diags diag.Diagnostics
	priorEmpty := prior.IsNull() || prior.IsUnknown() || len(prior.Elements()) == 0
	if priorEmpty {
		list, d := types.ListValue(objType, []attr.Value{})
		diags.Append(d...)
		return list, diags
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
	list, d := types.ListValue(objType, []attr.Value{obj})
	diags.Append(d...)
	return list, diags
}

// Suppress unused-import warning for strings when no strings.* used.
var _ = strings.Count
