package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var (
	_ resource.Resource                   = &FeatureFlagEnvironmentResource{}
	_ resource.ResourceWithImportState    = &FeatureFlagEnvironmentResource{}
	_ resource.ResourceWithUpgradeState   = &FeatureFlagEnvironmentResource{}
	_ resource.ResourceWithValidateConfig = &FeatureFlagEnvironmentResource{}
	_ resource.ResourceWithModifyPlan     = &FeatureFlagEnvironmentResource{}
)

type FeatureFlagEnvironmentResource struct {
	client *Client
}

type FeatureFlagEnvironmentResourceModel struct {
	ID                types.String `tfsdk:"id"`
	FlagID            types.String `tfsdk:"flag_id"`
	EnvKey            types.String `tfsdk:"env_key"`
	On                types.Bool   `tfsdk:"on"`
	Targets           types.Set    `tfsdk:"targets"`
	ContextTargets    types.Set    `tfsdk:"context_targets"`
	Rules             types.List   `tfsdk:"rules"`
	Prerequisites     types.List   `tfsdk:"prerequisites"`
	Fallthrough       types.Object `tfsdk:"fallthrough"`
	TrackEvents       types.Bool   `tfsdk:"track_events"`
	OffVariation      types.Int64  `tfsdk:"off_variation"`
	OffVariationName  types.String `tfsdk:"off_variation_name"`
	OffVariationValue types.String `tfsdk:"off_variation_value"`
}

// ffeResourceFallthroughAttrTypes / ffeResourceRuleAttrTypes are the
// resource-side object shapes for fallthrough/rules, distinct from the
// data source's ffeFallthroughAttrTypes/ffeRuleAttrTypes: the resource
// additionally carries the write-only variation_name/variation_value
// alternatives to variation (REL-14238). The data source stays
// index-only since it's a read-only projection.
var (
	ffeResourceFallthroughAttrTypes = map[string]attr.Type{
		VARIATION:       types.Int64Type,
		VARIATION_NAME:  types.StringType,
		VARIATION_VALUE: types.StringType,
		ROLLOUT_WEIGHTS: types.ListType{ElemType: types.Int64Type},
		BUCKET_BY:       types.StringType,
		CONTEXT_KIND:    types.StringType,
	}
	ffeResourceRuleAttrTypes = map[string]attr.Type{
		DESCRIPTION:     types.StringType,
		CLAUSES:         types.ListType{ElemType: types.ObjectType{AttrTypes: frameworkClauseAttrTypes}},
		VARIATION:       types.Int64Type,
		VARIATION_NAME:  types.StringType,
		VARIATION_VALUE: types.StringType,
		ROLLOUT_WEIGHTS: types.ListType{ElemType: types.Int64Type},
		BUCKET_BY:       types.StringType,
		CONTEXT_KIND:    types.StringType,
	}
)

func NewFeatureFlagEnvironmentResource() resource.Resource {
	return &FeatureFlagEnvironmentResource{}
}

func (r *FeatureFlagEnvironmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_feature_flag_environment"
}

func (r *FeatureFlagEnvironmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Provides a LaunchDarkly environment-specific feature flag resource.\n\nThis resource allows you to create and manage environment-specific feature flags attributes within your LaunchDarkly organization.\n\n-> **Note:** If you intend to attach a feature flag to any experiments, we do _not_ recommend configuring environment-specific flag settings using Terraform. Subsequent applies may overwrite the changes made by experiments and break your experiment. An alternate workaround is to use the [lifecycle.ignore_changes](https://developer.hashicorp.com/terraform/language/meta-arguments/lifecycle#ignore_changes) Terraform meta-argument on the `fallthrough` field to prevent potential overwrites.",
		Attributes:  featureFlagEnvironmentSchemaAttributes(),
	}
}

func featureFlagEnvironmentSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		FLAG_ID: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The feature flag's unique `id` in the format `project_key/flag_key`.", true),
			Validators:    []validator.String{flagIDValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		ENV_KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The environment key.", true),
			Validators:    []validator.String{keyValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		ON: schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
			Description: "Whether targeting is enabled. Defaults to `false` if not set.",
		},
		TRACK_EVENTS: schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
			Description: "Whether to send event data back to LaunchDarkly. Defaults to `false` if not set.",
		},
		OFF_VARIATION: schema.Int64Attribute{
			Optional:      true,
			Computed:      true,
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			Validators:    []validator.Int64{int64validator.AtLeast(0)},
			Description:   "The index of the variation to serve when targeting is off. Omitting this attribute (and `off_variation_name`/`off_variation_value`) leaves the off variation unset (the UI's \"Not set\" state), which is distinct from setting it to `0`. When it is unset and targeting is off, LaunchDarkly serves no variation: SDKs return the application-provided default value and the evaluation carries a null variation index, which affects Data Export and Experimentation. At most one of `off_variation`, `off_variation_name`, or `off_variation_value` may be set. Marked `(known after apply)` when resolved from `off_variation_name`/`off_variation_value`, since the flag's variations live on a separate `launchdarkly_feature_flag` resource this resource can't see at plan time.",
		},
		OFF_VARIATION_NAME: schema.StringAttribute{
			Optional:    true,
			Description: "The `name` of the flag variation to serve when targeting is off. Alternative to `off_variation`. Resolved against the flag's variations when applied — errors if none, or more than one, match. At most one of `off_variation`, `off_variation_name`, or `off_variation_value` may be set.",
		},
		OFF_VARIATION_VALUE: schema.StringAttribute{
			Optional:    true,
			Description: "The `value` of the flag variation to serve when targeting is off, in the same format as the flag's `variations[].value`. Alternative to `off_variation`. Resolved against the flag's variations when applied — errors if none, or more than one, match. At most one of `off_variation`, `off_variation_name`, or `off_variation_value` may be set.",
		},
		TARGETS: schema.SetNestedAttribute{
			Optional:    true,
			Description: "Individual user targets for each variation.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					VALUES: schema.ListAttribute{
						Required:    true,
						ElementType: types.StringType,
						Description: "List of `user` strings to target.",
					},
					VARIATION: schema.Int64Attribute{
						Required:    true,
						Validators:  []validator.Int64{int64validator.AtLeast(0)},
						Description: "The index of the variation to serve if a user target value is matched.",
					},
				},
			},
		},
		CONTEXT_TARGETS: schema.SetNestedAttribute{
			Optional:    true,
			Description: "Individual targets for non-user context kinds for each variation.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					VALUES: schema.ListAttribute{
						Required:    true,
						ElementType: types.StringType,
						Description: "List of `user` strings to target.",
					},
					VARIATION: schema.Int64Attribute{
						Required:    true,
						Validators:  []validator.Int64{int64validator.AtLeast(0)},
						Description: "The index of the variation to serve if a user target value is matched.",
					},
					CONTEXT_KIND: schema.StringAttribute{
						Required:    true,
						Description: "The context kind on which the flag should target in this environment. User (`user`) targets should be specified as `targets`.",
					},
				},
			},
		},
		PREREQUISITES: schema.ListNestedAttribute{
			Optional:    true,
			Description: "Prerequisite feature flag rules.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					FLAG_KEY: schema.StringAttribute{
						Required:    true,
						Description: "The prerequisite feature flag's `key`.",
						Validators:  []validator.String{keyValidator()},
					},
					VARIATION: schema.Int64Attribute{
						Required:    true,
						Validators:  []validator.Int64{int64validator.AtLeast(0)},
						Description: "The index of the prerequisite feature flag's variation to target.",
					},
				},
			},
		},
		RULES: schema.ListNestedAttribute{
			Optional:    true,
			Description: "List of logical targeting rules.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					DESCRIPTION: schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Description: "A human-readable description of the targeting rule.",
					},
					VARIATION: schema.Int64Attribute{
						Optional:      true,
						Computed:      true,
						PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
						Validators:    []validator.Int64{int64validator.AtLeast(0)},
						Description:   "The integer variation index to serve if the rule clauses evaluate to `true`. You must specify one of `variation`, `variation_name`, `variation_value`, or `rollout_weights`.",
					},
					VARIATION_NAME: schema.StringAttribute{
						Optional:    true,
						Description: "The `name` of the flag variation to serve if the rule clauses evaluate to `true`. Alternative to `variation`. Resolved against the flag's variations when applied — errors if none, or more than one, match.",
					},
					VARIATION_VALUE: schema.StringAttribute{
						Optional:    true,
						Description: "The `value` of the flag variation to serve if the rule clauses evaluate to `true`, in the same format as the flag's `variations[].value`. Alternative to `variation`. Resolved against the flag's variations when applied — errors if none, or more than one, match.",
					},
					BUCKET_BY: schema.StringAttribute{
						Optional:    true,
						Description: "Group percentage rollout by a custom attribute. This argument is only valid if `rollout_weights` is also specified.",
					},
					CONTEXT_KIND: schema.StringAttribute{
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("user"),
						Description: "The context kind associated with the specified rollout. This argument is only valid if `rollout_weights` is also specified. Defaults to `user` if omitted.",
					},
					ROLLOUT_WEIGHTS: schema.ListAttribute{
						Optional:    true,
						ElementType: types.Int64Type,
						Validators: []validator.List{
							listvalidator.ValueInt64sAre(int64validator.Between(0, 100000)),
						},
						Description: "List of integer percentage rollout weights (in thousandths of a percent) to apply to each variation if the rule clauses evaluates to `true`. The sum of the `rollout_weights` must equal 100000 and the number of rollout weights specified in the array must match the number of flag variations. You must specify either `variation` or `rollout_weights`.",
					},
					CLAUSES: frameworkClausesResourceAttribute(),
				},
			},
		},
		FALLTHROUGH: schema.SingleNestedAttribute{
			Required:    true,
			Description: "The default variation to serve if no `prerequisites`, `target`, or `rules` apply.",
			Attributes: map[string]schema.Attribute{
				VARIATION: schema.Int64Attribute{
					Optional:    true,
					Computed:    true,
					Default:     int64default.StaticInt64(0),
					Validators:  []validator.Int64{int64validator.AtLeast(0)},
					Description: "The default integer variation index to serve if no `prerequisites`, `target`, or `rules` apply. You must specify one of `variation`, `variation_name`, `variation_value`, or `rollout_weights`. Defaults to `0` when none of `variation`, `variation_name`, `variation_value`, or `rollout_weights` is set.",
				},
				VARIATION_NAME: schema.StringAttribute{
					Optional:    true,
					Description: "The `name` of the flag variation to serve if no `prerequisites`, `target`, or `rules` apply. Alternative to `variation`. Resolved against the flag's variations when applied — errors if none, or more than one, match.",
				},
				VARIATION_VALUE: schema.StringAttribute{
					Optional:    true,
					Description: "The `value` of the flag variation to serve if no `prerequisites`, `target`, or `rules` apply, in the same format as the flag's `variations[].value`. Alternative to `variation`. Resolved against the flag's variations when applied — errors if none, or more than one, match.",
				},
				BUCKET_BY: schema.StringAttribute{
					Optional:    true,
					Description: "Group percentage rollout by a custom attribute. This argument is only valid if rollout_weights is also specified.",
				},
				CONTEXT_KIND: schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Default:     stringdefault.StaticString("user"),
					Description: "The context kind associated with the specified rollout. This argument is only valid if rollout_weights is also specified. If omitted, defaults to `user`.",
				},
				ROLLOUT_WEIGHTS: schema.ListAttribute{
					Optional:    true,
					ElementType: types.Int64Type,
					Validators: []validator.List{
						listvalidator.ValueInt64sAre(int64validator.Between(0, 100000)),
					},
					Description: "List of integer percentage rollout weights (in thousandths of a percent) to apply to each variation if the rule clauses evaluates to `true`. The sum of the `rollout_weights` must equal 100000 and the number of rollout weights specified in the array must match the number of flag variations. You must specify either `variation` or `rollout_weights`.",
				},
			},
		},
	}
}

func (r *FeatureFlagEnvironmentResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	priorSchema := schema.Schema{Attributes: featureFlagEnvironmentSchemaAttributesV0()}
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &priorSchema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior FeatureFlagEnvironmentResourceModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}
				// v0 (SDKv2) stored fallthrough as a block (single-element
				// list); v3 models it as a single object.
				fallthroughObj, d := ffeFallthroughObjectFromV0List(ctx, prior.Fallthrough)
				resp.Diagnostics.Append(d...)
				if resp.Diagnostics.HasError() {
					return
				}
				data := FeatureFlagEnvironmentResourceModel{
					ID:             prior.ID,
					FlagID:         prior.FlagID,
					EnvKey:         prior.EnvKey,
					On:             prior.On,
					Targets:        nullIfEmptySet(ctx, prior.Targets),
					ContextTargets: nullIfEmptySet(ctx, prior.ContextTargets),
					Rules:          nullIfEmptyList(ctx, prior.Rules),
					Prerequisites:  nullIfEmptyList(ctx, prior.Prerequisites),
					Fallthrough:    fallthroughObj,
					TrackEvents:    prior.TrackEvents,
					OffVariation:   prior.OffVariation,
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			},
		},
	}
}

func (r *FeatureFlagEnvironmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

// ffeRuleSelectorModel is the full field set of the resource's rule object
// (ffeResourceRuleAttrTypes) — Object.As/ElementsAs require an exact 1:1
// match between struct tags and object attribute types, so this must
// stay in sync with ffeResourceRuleAttrTypes even where a caller only
// needs a subset of fields.
type ffeRuleSelectorModel struct {
	Description    types.String `tfsdk:"description"`
	Clauses        types.List   `tfsdk:"clauses"`
	Variation      types.Int64  `tfsdk:"variation"`
	VariationName  types.String `tfsdk:"variation_name"`
	VariationValue types.String `tfsdk:"variation_value"`
	RolloutWeights types.List   `tfsdk:"rollout_weights"`
	BucketBy       types.String `tfsdk:"bucket_by"`
	ContextKind    types.String `tfsdk:"context_kind"`
}

// ffeFallthroughSelectorModel is the full field set of the resource's
// fallthrough object (ffeResourceFallthroughAttrTypes); see
// ffeRuleSelectorModel for why every field must be present.
type ffeFallthroughSelectorModel struct {
	Variation      types.Int64  `tfsdk:"variation"`
	VariationName  types.String `tfsdk:"variation_name"`
	VariationValue types.String `tfsdk:"variation_value"`
	RolloutWeights types.List   `tfsdk:"rollout_weights"`
	BucketBy       types.String `tfsdk:"bucket_by"`
	ContextKind    types.String `tfsdk:"context_kind"`
}

// ValidateConfig catches off_variation/variation _name/_value conflicts at
// `terraform plan`, before any API call. Unlike launchdarkly_feature_flag,
// this resource can't see the sibling flag's variations, so only the "at
// most one set" check runs here — full resolution happens at apply time
// in Create/Update, once the flag has been fetched.
func (r *FeatureFlagEnvironmentResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config FeatureFlagEnvironmentResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	checkExclusivity := func(attrPath path.Path, sel variationSelector, label string) {
		if err := validateVariationSelectorExclusivity(sel, label); err != nil {
			resp.Diagnostics.AddAttributeError(attrPath, "invalid "+label, err.Error())
		}
	}

	checkExclusivity(
		path.Root(OFF_VARIATION),
		variationSelectorFromInt64AndStrings(config.OffVariation, config.OffVariationName, config.OffVariationValue),
		OFF_VARIATION,
	)

	if !config.Fallthrough.IsNull() && !config.Fallthrough.IsUnknown() {
		var m ffeFallthroughSelectorModel
		resp.Diagnostics.Append(config.Fallthrough.As(ctx, &m, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		checkExclusivity(
			path.Root(FALLTHROUGH).AtName(VARIATION),
			variationSelectorFromInt64AndStrings(m.Variation, m.VariationName, m.VariationValue),
			"fallthrough.variation",
		)
	}

	if !config.Rules.IsNull() && !config.Rules.IsUnknown() {
		var rules []ffeRuleSelectorModel
		resp.Diagnostics.Append(config.Rules.ElementsAs(ctx, &rules, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for i, rule := range rules {
			checkExclusivity(
				path.Root(RULES).AtListIndex(i).AtName(VARIATION),
				variationSelectorFromInt64AndStrings(rule.Variation, rule.VariationName, rule.VariationValue),
				fmt.Sprintf("rules[%d].variation", i),
			)
		}
	}
}

// ModifyPlan marks off_variation/fallthrough.variation/rules[].variation
// Unknown at plan time when their _name/_value alternative is configured
// instead — the framework requires Computed:true attributes to explicitly
// signal "known after apply" rather than silently locking in a stale
// index. Actual resolution against the flag's variations happens in
// Create/Update, since this resource can't see the sibling
// launchdarkly_feature_flag resource's variations at plan time.
//
// The inverse case (off_variation/rules[].variation genuinely unset, no
// _name/_value either) is forced back to explicit Null: Computed:true
// with no config value otherwise defaults to Unknown, which would
// regress the "Not set" contract off_variation already relies on
// (#482/#483).
func (r *FeatureFlagEnvironmentResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return // destroy plan
	}
	var config FeatureFlagEnvironmentResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	switch {
	case !config.OffVariation.IsNull():
		// user set the index directly; plan already mirrors config.
	case !config.OffVariationName.IsNull() || !config.OffVariationValue.IsNull():
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root(OFF_VARIATION), types.Int64Unknown())...)
	default:
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root(OFF_VARIATION), types.Int64Null())...)
	}

	if !config.Fallthrough.IsNull() && !config.Fallthrough.IsUnknown() {
		var m ffeFallthroughSelectorModel
		resp.Diagnostics.Append(config.Fallthrough.As(ctx, &m, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		// Only the "resolve from name/value" case needs an override here:
		// the "nothing set" case is already handled correctly by the
		// schema's existing Default(0) plan modifier.
		if m.Variation.IsNull() && (!m.VariationName.IsNull() || !m.VariationValue.IsNull()) {
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root(FALLTHROUGH).AtName(VARIATION), types.Int64Unknown())...)
		}
	}

	if !config.Rules.IsNull() && !config.Rules.IsUnknown() {
		var rules []ffeRuleSelectorModel
		resp.Diagnostics.Append(config.Rules.ElementsAs(ctx, &rules, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for i, rule := range rules {
			rulePath := path.Root(RULES).AtListIndex(i).AtName(VARIATION)
			switch {
			case !rule.Variation.IsNull():
			case !rule.VariationName.IsNull() || !rule.VariationValue.IsNull():
				resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, rulePath, types.Int64Unknown())...)
			default:
				resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, rulePath, types.Int64Null())...)
			}
		}
	}
}

func (r *FeatureFlagEnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FeatureFlagEnvironmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	flagID := plan.FlagID.ValueString()
	projectKey, flagKey, err := flagIdToKeys(flagID)
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "")
		return
	}
	envKey := plan.EnvKey.ValueString()
	if envKey == "" {
		resp.Diagnostics.AddError(
			fmt.Sprintf("%s is required and must be the LaunchDarkly environment key (not the display name). If the embedded schema omits it, set resource id to project_key/env_key/flag_key before create.", ENV_KEY),
			"",
		)
		return
	}

	if exists, err := projectExists(projectKey, r.client); !exists {
		if err != nil {
			resp.Diagnostics.AddError(err.Error(), "")
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("cannot find project with key %q", projectKey), "")
		return
	}
	if exists, err := environmentExists(projectKey, envKey, r.client); !exists {
		if err != nil {
			resp.Diagnostics.AddError(err.Error(), "")
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("environment %q not found in project %q — env_key must be the LaunchDarkly environment **key**, not its display name. Create the environment first.", envKey, projectKey),
			"",
		)
		return
	}

	// Fetch the flag unconditionally: needed both to determine whether a
	// live offVariation is present (a remove of an absent path returns 400
	// invalid_patch, which would otherwise block create for an environment
	// already in "Not set") and to resolve any off_variation_name/value,
	// fallthrough.variation_name/value, or rules[].variation_name/value
	// against the flag's real variations (REL-14238) — this resource can't
	// see the sibling launchdarkly_feature_flag resource's variations any
	// other way.
	flag, _, gerr := getFeatureFlagEnvironment(r.client, projectKey, flagKey, envKey)
	if gerr != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to read flag %q of project %q before create: %s", flagKey, projectKey, handleLdapiErr(gerr).Error()), "")
		return
	}
	offVariationLiveSet := false
	if flag.Environments != nil {
		if env, ok := (*flag.Environments)[envKey]; ok && env.OffVariation != nil {
			offVariationLiveSet = true
		}
	}
	resp.Diagnostics.Append(resolveFFEVariationIndices(ctx, &plan, resolvableVariationsFromAPI(flag.Variations))...)
	if resp.Diagnostics.HasError() {
		return
	}

	patches, d := buildFFEPatches(ctx, envKey, plan, FeatureFlagEnvironmentResourceModel{}, true, offVariationLiveSet)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(patches) > 0 {
		comment := "Terraform"
		patch := ldapi.PatchWithComment{Comment: &comment, Patch: patches}
		log.Printf("[DEBUG] %+v\n", patch)
		err = r.client.withConcurrency(r.client.ctx, func() error {
			_, _, e := r.client.ld.FeatureFlagsApi.PatchFeatureFlag(r.client.ctx, projectKey, flagKey).PatchWithComment(patch).Execute()
			return e
		})
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("failed to update flag %q in project %q: %s", flagKey, projectKey, handleLdapiErr(err).Error()), "")
			return
		}
	}

	r.readIntoModel(ctx, projectKey, flagKey, envKey, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.StringValue(projectKey + "/" + envKey + "/" + flagKey)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FeatureFlagEnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FeatureFlagEnvironmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectKey, flagKey, err := flagIdToKeys(data.FlagID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "")
		return
	}
	r.readIntoModel(ctx, projectKey, flagKey, data.EnvKey.ValueString(), &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FeatureFlagEnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state FeatureFlagEnvironmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectKey, flagKey, err := flagIdToKeys(plan.FlagID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "")
		return
	}
	envKey := plan.EnvKey.ValueString()
	if envKey == "" {
		resp.Diagnostics.AddError(fmt.Sprintf("%s is empty and resource id %q is not project_key/env_key/flag_key", ENV_KEY, plan.ID.ValueString()), "")
		return
	}

	if exists, err := projectExists(projectKey, r.client); !exists {
		if err != nil {
			resp.Diagnostics.AddError(err.Error(), "")
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("cannot find project with key %q", projectKey), "")
		return
	}
	if exists, err := environmentExists(projectKey, envKey, r.client); !exists {
		if err != nil {
			resp.Diagnostics.AddError(err.Error(), "")
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("environment %q not found in project %q — env_key must be the LaunchDarkly environment **key**. Create the environment first or correct env_key.", envKey, projectKey),
			"",
		)
		return
	}

	// Fetch the flag to resolve any off_variation_name/value,
	// fallthrough.variation_name/value, or rules[].variation_name/value
	// against the flag's real variations (REL-14238) — this resource can't
	// see the sibling launchdarkly_feature_flag resource's variations any
	// other way.
	flag, _, gerr := getFeatureFlagEnvironment(r.client, projectKey, flagKey, envKey)
	if gerr != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to read flag %q of project %q before update: %s", flagKey, projectKey, handleLdapiErr(gerr).Error()), "")
		return
	}
	resp.Diagnostics.Append(resolveFFEVariationIndices(ctx, &plan, resolvableVariationsFromAPI(flag.Variations))...)
	if resp.Diagnostics.HasError() {
		return
	}

	// state is post-refresh, so its off_variation presence mirrors the live
	// environment — a safe basis for deciding whether a remove is valid.
	patches, d := buildFFEPatches(ctx, envKey, plan, state, false, !state.OffVariation.IsNull())
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(patches) > 0 {
		comment := "Terraform"
		patch := ldapi.PatchWithComment{Comment: &comment, Patch: patches}
		log.Printf("[DEBUG] %+v\n", patch)
		err = r.client.withConcurrency(r.client.ctx, func() error {
			_, _, e := r.client.ld.FeatureFlagsApi.PatchFeatureFlag(r.client.ctx, projectKey, flagKey).PatchWithComment(patch).Execute()
			return e
		})
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("failed to update flag %q in project %q, environment %q: %s", flagKey, projectKey, envKey, handleLdapiErr(err).Error()), "")
			return
		}
	}
	r.readIntoModel(ctx, projectKey, flagKey, envKey, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.StringValue(projectKey + "/" + envKey + "/" + flagKey)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FeatureFlagEnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FeatureFlagEnvironmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectKey, flagKey, err := flagIdToKeys(data.FlagID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "")
		return
	}
	envKey := data.EnvKey.ValueString()

	if exists, err := projectExists(projectKey, r.client); !exists {
		if err != nil {
			resp.Diagnostics.AddError(err.Error(), "")
			return
		}
		resp.Diagnostics.AddError(fmt.Sprintf("cannot find project with key %q", projectKey), "")
		return
	}
	if exists, err := environmentExists(projectKey, envKey, r.client); !exists {
		if err != nil {
			resp.Diagnostics.AddError(err.Error(), "")
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("environment %q not found in project %q — env_key must be the LaunchDarkly environment **key**.", envKey, projectKey),
			"",
		)
		return
	}

	var flag *ldapi.FeatureFlag
	err = r.client.withConcurrency(r.client.ctx, func() error {
		flag, _, err = r.client.ld.FeatureFlagsApi.GetFeatureFlag(r.client.ctx, projectKey, flagKey).Execute()
		return err
	})
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to update flag %q in project %q, environment %q: %s", flagKey, projectKey, envKey, handleLdapiErr(err).Error()), "")
		return
	}

	offVariation := int32(len(flag.Variations) - 1)
	if flag.Defaults != nil {
		offVariation = flag.Defaults.OffVariation
	}

	comment := "Terraform"
	zeroVar := int32(0)
	patch := ldapi.PatchWithComment{
		Comment: &comment,
		Patch: []ldapi.PatchOperation{
			patchReplace(ffePatchPath(envKey, "on"), false),
			patchReplace(ffePatchPath(envKey, "rules"), []ldapi.Rule{}),
			patchReplace(ffePatchPath(envKey, "trackEvents"), false),
			patchReplace(ffePatchPath(envKey, "prerequisites"), []ldapi.Prerequisite{}),
			patchReplace(ffePatchPath(envKey, "offVariation"), offVariation),
			patchReplace(ffePatchPath(envKey, "targets"), []ldapi.Target{}),
			patchReplace(ffePatchPath(envKey, "contextTargets"), []ldapi.Target{}),
			patchReplace(ffePatchPath(envKey, "fallthough"), ffeFallthroughPayload{Variation: &zeroVar}),
		},
	}
	log.Printf("[DEBUG] %+v\n", patch)

	err = r.client.withConcurrency(r.client.ctx, func() error {
		_, _, err = r.client.ld.FeatureFlagsApi.PatchFeatureFlag(r.client.ctx, projectKey, flagKey).PatchWithComment(patch).Execute()
		return err
	})
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to update flag %q in project %q, environment %q: %s", flagKey, projectKey, envKey, handleLdapiErr(err).Error()), "")
	}
}

func (r *FeatureFlagEnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if strings.Count(req.ID, "/") != 2 {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("expected project_key/env_key/flag_key, got %q", req.ID))
		return
	}
	parts := strings.SplitN(req.ID, "/", 3)
	flagID := parts[0] + "/" + parts[2]
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(FLAG_ID), flagID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(ENV_KEY), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *FeatureFlagEnvironmentResource) readIntoModel(ctx context.Context, projectKey, flagKey, envKey string, data *FeatureFlagEnvironmentResourceModel, diags *diag.Diagnostics) {
	envExists, err := environmentExists(projectKey, envKey, r.client)
	if err != nil {
		diags.AddError(err.Error(), "")
		return
	}
	if !envExists {
		data.ID = types.StringNull()
		return
	}

	flag, res, err := getFeatureFlagEnvironment(r.client, projectKey, flagKey, envKey)
	if isStatusNotFound(res) {
		data.ID = types.StringNull()
		return
	}
	if err != nil {
		diags.AddError(fmt.Sprintf("failed to get flag %q of project %q: %s", flagKey, projectKey, handleLdapiErr(err).Error()), "")
		return
	}
	if flag.Environments == nil {
		data.ID = types.StringNull()
		return
	}
	environment, ok := (*flag.Environments)[envKey]
	if !ok {
		data.ID = types.StringNull()
		return
	}

	data.ID = types.StringValue(projectKey + "/" + envKey + "/" + flagKey)
	data.FlagID = types.StringValue(projectKey + "/" + flag.Key)
	data.On = types.BoolValue(environment.On)
	data.TrackEvents = types.BoolValue(environment.TrackEvents)
	if environment.OffVariation != nil {
		data.OffVariation = types.Int64Value(int64(*environment.OffVariation))
	} else {
		// No offVariation on the environment ("Not set") — model as null so
		// it round-trips instead of collapsing to a literal index 0.
		data.OffVariation = types.Int64Null()
	}

	noopDiags := noopDiagSink{}
	data.Targets = ffeTargetsValue(ctx, environment.Targets, false, noopDiags)
	data.ContextTargets = ffeTargetsValue(ctx, environment.ContextTargets, true, noopDiags)
	// variation_name/variation_value are write-only (Optional, not
	// Computed): pass through whatever the caller already had rather than
	// deriving fresh values from the API, or the plan-apply consistency
	// check would trip (REL-14238).
	priorRules, priorFallthrough := data.Rules, data.Fallthrough
	data.Rules = ffeResourceRulesValue(ctx, environment.Rules, priorRules, diags)
	data.Fallthrough = ffeResourceFallthroughValue(ctx, environment.Fallthrough, priorFallthrough, diags)

	prereqObjectType := types.ObjectType{AttrTypes: ffePrerequisiteAttrTypes}
	prereqElements := make([]attr.Value, 0, len(environment.Prerequisites))
	for _, p := range environment.Prerequisites {
		obj, d := types.ObjectValue(ffePrerequisiteAttrTypes, map[string]attr.Value{
			FLAG_KEY:  types.StringValue(p.Key),
			VARIATION: types.Int64Value(int64(p.Variation)),
		})
		diags.Append(d...)
		prereqElements = append(prereqElements, obj)
	}
	if len(prereqElements) == 0 {
		data.Prerequisites = types.ListNull(prereqObjectType)
	} else {
		prereqList, d := types.ListValue(prereqObjectType, prereqElements)
		diags.Append(d...)
		data.Prerequisites = prereqList
	}
}

// noopDiagSink absorbs diags from ffe* helpers that take a sink
// interface. Used in Read paths where the conversion errors would also
// surface on the structured types.ListValue / SetValue paths anyway.
type noopDiagSink struct{}

func (noopDiagSink) AddError(string, string) {}

// ffeRulePriorVariationRefsByIndex extracts variation_name/variation_value
// from a prior rules list, keyed by list position (rules have no other
// stable identity to match on). Returns nil slices when priorRules is
// null/unknown/unreadable — callers treat any out-of-range index as null.
func ffeRulePriorVariationRefsByIndex(ctx context.Context, priorRules types.List) (names, values []types.String) {
	if priorRules.IsNull() || priorRules.IsUnknown() {
		return nil, nil
	}
	var models []ffeRuleSelectorModel
	if priorRules.ElementsAs(ctx, &models, false).HasError() {
		return nil, nil
	}
	names = make([]types.String, len(models))
	values = make([]types.String, len(models))
	for i, m := range models {
		names[i] = m.VariationName
		values[i] = m.VariationValue
	}
	return names, values
}

// ffeResourceRulesValue emits null for Optional-only attributes the
// user did not configure: variation is null when a rollout is present,
// bucket_by / context_kind are null when not a rollout, description is
// null when nil. The data-source-side ffeRulesValue emits zero values
// instead — fine for Computed-only data source attrs but would trip
// the plan-apply consistency check on the resource's Optional-only
// attrs.
//
// variation_name/variation_value are write-only (Optional, not
// Computed): pass through priorRules by list position rather than
// deriving fresh values from the API (REL-14238).
func ffeResourceRulesValue(ctx context.Context, rules []ldapi.Rule, priorRules types.List, diags *diag.Diagnostics) types.List {
	objectType := types.ObjectType{AttrTypes: ffeResourceRuleAttrTypes}
	priorNames, priorValues := ffeRulePriorVariationRefsByIndex(ctx, priorRules)
	elements := make([]attr.Value, 0, len(rules))
	for i, r := range rules {
		clauses, d := frameworkClausesValue(ctx, r.Clauses)
		diags.Append(d...)

		// variation / rollout_weights are mutually exclusive; emit null
		// for whichever the user did not configure so plan and state
		// match. context_kind has Default+Computed at the schema level,
		// so we always emit a value (defaults to "user").
		variation := types.Int64Null()
		bucketBy := types.StringNull()
		contextKind := types.StringValue("user")
		weights := types.ListNull(types.Int64Type)
		if r.Rollout != nil {
			weightValues := make([]attr.Value, 0, len(r.Rollout.Variations))
			for _, w := range r.Rollout.Variations {
				weightValues = append(weightValues, types.Int64Value(int64(w.Weight)))
			}
			w, d := types.ListValue(types.Int64Type, weightValues)
			diags.Append(d...)
			weights = w
			if r.Rollout.BucketBy != nil {
				bucketBy = types.StringValue(*r.Rollout.BucketBy)
			}
			if r.Rollout.ContextKind != nil {
				contextKind = types.StringValue(*r.Rollout.ContextKind)
			}
		}
		if r.Variation != nil {
			variation = types.Int64Value(int64(*r.Variation))
		}
		// Schema declares Optional+Computed+Default("") so plan and
		// post-apply state stay aligned even when the user omits
		// description.
		description := types.StringValue("")
		if r.Description != nil {
			description = types.StringValue(*r.Description)
		}
		variationName, variationValue := types.StringNull(), types.StringNull()
		if i < len(priorNames) {
			variationName, variationValue = priorNames[i], priorValues[i]
		}
		obj, d := types.ObjectValue(ffeResourceRuleAttrTypes, map[string]attr.Value{
			DESCRIPTION:     description,
			CLAUSES:         clauses,
			VARIATION:       variation,
			VARIATION_NAME:  variationName,
			VARIATION_VALUE: variationValue,
			ROLLOUT_WEIGHTS: weights,
			BUCKET_BY:       bucketBy,
			CONTEXT_KIND:    contextKind,
		})
		diags.Append(d...)
		elements = append(elements, obj)
	}
	if len(elements) == 0 {
		return types.ListNull(objectType)
	}
	list, d := types.ListValue(objectType, elements)
	diags.Append(d...)
	return list
}

// ffeResourceFallthroughValue emits the resource-side fallthrough object
// with defaults applied: variation defaults to 0 and context_kind
// defaults to "user" so plan-vs-state stays consistent when the user
// omits these attrs (framework Default+Computed schema flags fill the
// plan; Read must emit matching values).
//
// variation_name/variation_value are write-only (Optional, not
// Computed): pass through priorFallthrough rather than deriving fresh
// values from the API (REL-14238).
func ffeResourceFallthroughValue(ctx context.Context, fallthroughRep *ldapi.VariationOrRolloutRep, priorFallthrough types.Object, diags *diag.Diagnostics) types.Object {
	if fallthroughRep == nil {
		return types.ObjectNull(ffeResourceFallthroughAttrTypes)
	}
	variation := types.Int64Value(0)
	bucketBy := types.StringNull()
	contextKind := types.StringValue("user")
	weights := types.ListNull(types.Int64Type)
	if fallthroughRep.Rollout != nil {
		weightValues := make([]attr.Value, 0, len(fallthroughRep.Rollout.Variations))
		for _, w := range fallthroughRep.Rollout.Variations {
			weightValues = append(weightValues, types.Int64Value(int64(w.Weight)))
		}
		w, d := types.ListValue(types.Int64Type, weightValues)
		diags.Append(d...)
		weights = w
		if fallthroughRep.Rollout.BucketBy != nil {
			bucketBy = types.StringValue(*fallthroughRep.Rollout.BucketBy)
		}
		if fallthroughRep.Rollout.ContextKind != nil {
			contextKind = types.StringValue(*fallthroughRep.Rollout.ContextKind)
		}
	}
	if fallthroughRep.Variation != nil {
		variation = types.Int64Value(int64(*fallthroughRep.Variation))
	}
	variationName, variationValue := types.StringNull(), types.StringNull()
	if !priorFallthrough.IsNull() && !priorFallthrough.IsUnknown() {
		var m ffeFallthroughSelectorModel
		if d := priorFallthrough.As(ctx, &m, basetypes.ObjectAsOptions{}); !d.HasError() {
			variationName, variationValue = m.VariationName, m.VariationValue
		}
	}
	obj, d := types.ObjectValue(ffeResourceFallthroughAttrTypes, map[string]attr.Value{
		VARIATION:       variation,
		VARIATION_NAME:  variationName,
		VARIATION_VALUE: variationValue,
		ROLLOUT_WEIGHTS: weights,
		BUCKET_BY:       bucketBy,
		CONTEXT_KIND:    contextKind,
	})
	diags.Append(d...)
	return obj
}

// ffePatchPath returns the JSON-Pointer path for an environment-scoped
// attribute of a feature flag (e.g. /environments/<envKey>/on).
func ffePatchPath(envKey, op string) string {
	return "/environments/" + envKey + "/" + op
}

// ffeFallthroughPayload is the JSON shape LD expects on a fallthrough
// patch — variation OR rollout, never both.
type ffeFallthroughPayload struct {
	Variation *int32         `json:"variation,omitempty"`
	Rollout   *ldapi.Rollout `json:"rollout,omitempty"`
}

// resolveFFEVariationIndices resolves off_variation, fallthrough.variation,
// and each rules[].variation from their _name/_value alternatives
// (REL-14238) against the flag's real variations, mutating plan in place so
// buildFFEPatches (and everything downstream of it) sees only concrete
// indices — no other code needs to know resolution happened. Only sites
// where the index is null/unknown AND a name/value is configured get
// touched; sites where the index is already concrete (whether user-set or
// defaulted) are left alone. Errors are reported per-site via
// diag.AddAttributeError so a bad reference in one rule doesn't mask
// others.
func resolveFFEVariationIndices(ctx context.Context, plan *FeatureFlagEnvironmentResourceModel, resolvable []resolvableVariation) diag.Diagnostics {
	var diags diag.Diagnostics

	offSel := variationSelectorFromInt64AndStrings(plan.OffVariation, plan.OffVariationName, plan.OffVariationValue)
	if offSel.Name != nil || offSel.Value != nil {
		idx, err := resolveVariationIndex(offSel, resolvable, OFF_VARIATION)
		if err != nil {
			diags.AddAttributeError(path.Root(OFF_VARIATION), "invalid "+OFF_VARIATION, err.Error())
		} else {
			plan.OffVariation = types.Int64Value(int64(idx))
		}
	}
	// else: offSel.Index set (nothing to resolve) or entirely empty
	// (genuine "Not set" — leave plan.OffVariation as null).

	if !plan.Fallthrough.IsNull() && !plan.Fallthrough.IsUnknown() {
		var m ffeFallthroughSelectorModel
		diags.Append(plan.Fallthrough.As(ctx, &m, basetypes.ObjectAsOptions{})...)
		if !diags.HasError() {
			sel := variationSelectorFromInt64AndStrings(m.Variation, m.VariationName, m.VariationValue)
			if sel.Name != nil || sel.Value != nil {
				idx, err := resolveVariationIndex(sel, resolvable, "fallthrough.variation")
				if err != nil {
					diags.AddAttributeError(path.Root(FALLTHROUGH).AtName(VARIATION), "invalid fallthrough.variation", err.Error())
				} else {
					obj, od := types.ObjectValue(ffeResourceFallthroughAttrTypes, map[string]attr.Value{
						VARIATION:       types.Int64Value(int64(idx)),
						VARIATION_NAME:  m.VariationName,
						VARIATION_VALUE: m.VariationValue,
						ROLLOUT_WEIGHTS: m.RolloutWeights,
						BUCKET_BY:       m.BucketBy,
						CONTEXT_KIND:    m.ContextKind,
					})
					diags.Append(od...)
					plan.Fallthrough = obj
				}
			}
		}
	}

	if !plan.Rules.IsNull() && !plan.Rules.IsUnknown() {
		var rules []ffeRuleSelectorModel
		diags.Append(plan.Rules.ElementsAs(ctx, &rules, false)...)
		if !diags.HasError() {
			changed := false
			for i := range rules {
				sel := variationSelectorFromInt64AndStrings(rules[i].Variation, rules[i].VariationName, rules[i].VariationValue)
				if sel.Name == nil && sel.Value == nil {
					continue
				}
				idx, err := resolveVariationIndex(sel, resolvable, fmt.Sprintf("rules[%d].variation", i))
				if err != nil {
					diags.AddAttributeError(path.Root(RULES).AtListIndex(i).AtName(VARIATION), fmt.Sprintf("invalid rules[%d].variation", i), err.Error())
					continue
				}
				rules[i].Variation = types.Int64Value(int64(idx))
				changed = true
			}
			if changed && !diags.HasError() {
				elements := make([]attr.Value, 0, len(rules))
				for _, r := range rules {
					obj, od := types.ObjectValue(ffeResourceRuleAttrTypes, map[string]attr.Value{
						DESCRIPTION:     r.Description,
						CLAUSES:         r.Clauses,
						VARIATION:       r.Variation,
						VARIATION_NAME:  r.VariationName,
						VARIATION_VALUE: r.VariationValue,
						ROLLOUT_WEIGHTS: r.RolloutWeights,
						BUCKET_BY:       r.BucketBy,
						CONTEXT_KIND:    r.ContextKind,
					})
					diags.Append(od...)
					elements = append(elements, obj)
				}
				list, ld := types.ListValue(types.ObjectType{AttrTypes: ffeResourceRuleAttrTypes}, elements)
				diags.Append(ld...)
				plan.Rules = list
			}
		}
	}

	return diags
}

// buildFFEPatches assembles the JSON-Patch document applied at
// Create/Update. Each attribute is patched only when it differs from
// state (or unconditionally on create).
// offVariationLiveSet reports whether the live environment currently has an
// offVariation. The caller supplies it because a JSON Patch remove of an
// absent path returns 400 invalid_patch from LaunchDarkly: Update derives it
// from the post-refresh state, Create reads it from the API.
func buildFFEPatches(ctx context.Context, envKey string, plan, state FeatureFlagEnvironmentResourceModel, isCreate, offVariationLiveSet bool) ([]ldapi.PatchOperation, diag.Diagnostics) {
	var diags diag.Diagnostics
	patches := make([]ldapi.PatchOperation, 0)

	if isCreate || !plan.On.Equal(state.On) {
		patches = append(patches, patchReplace(ffePatchPath(envKey, "on"), plan.On.ValueBool()))
	}
	// off_variation is optional: a null value models LD's "Not set" state
	// (no offVariation field on the environment). LaunchDarkly initialises
	// every environment's offVariation to the flag's default when the flag
	// is created, so honouring a null generally means emitting a JSON Patch
	// remove — merely omitting the patch would leave the default in place and
	// trip the "inconsistent result after apply" check. We only remove when
	// the live environment actually has an offVariation, because a remove of
	// an absent path returns 400 invalid_patch.
	if plan.OffVariation.IsNull() {
		if offVariationLiveSet {
			patches = append(patches, patchRemove(ffePatchPath(envKey, "offVariation")))
		}
	} else if isCreate || !plan.OffVariation.Equal(state.OffVariation) {
		patches = append(patches, patchReplace(ffePatchPath(envKey, "offVariation"), int32(plan.OffVariation.ValueInt64())))
	}
	if isCreate || !plan.TrackEvents.Equal(state.TrackEvents) {
		patches = append(patches, patchReplace(ffePatchPath(envKey, "trackEvents"), plan.TrackEvents.ValueBool()))
	}

	rules, d := ffeRulesFromList(ctx, plan.Rules)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	if isCreate || !plan.Rules.Equal(state.Rules) {
		patches = append(patches, patchReplace(ffePatchPath(envKey, "rules"), rules))
	}

	prereqs, d := ffePrerequisitesFromList(ctx, plan.Prerequisites)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	if isCreate || !plan.Prerequisites.Equal(state.Prerequisites) {
		patches = append(patches, patchReplace(ffePatchPath(envKey, "prerequisites"), prereqs))
	}

	targets, d := ffeTargetsFromSet(ctx, plan.Targets, false)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	if isCreate || !plan.Targets.Equal(state.Targets) {
		patches = append(patches, patchReplace(ffePatchPath(envKey, "targets"), targets))
	}
	ctxTargets, d := ffeTargetsFromSet(ctx, plan.ContextTargets, true)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	if isCreate || !plan.ContextTargets.Equal(state.ContextTargets) {
		patches = append(patches, patchReplace(ffePatchPath(envKey, "contextTargets"), ctxTargets))
	}

	fall, d := ffeFallthroughFromObject(ctx, plan.Fallthrough)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	if isCreate || !plan.Fallthrough.Equal(state.Fallthrough) {
		patches = append(patches, patchReplace(ffePatchPath(envKey, "fallthrough"), fall))
	}
	return patches, diags
}

// ffeRulesFromList converts the framework List<rule> into the API's
// rule payload shape (variation XOR rollout).
type ffeRulePayload struct {
	Description *string        `json:"description,omitempty"`
	Variation   *int32         `json:"variation,omitempty"`
	Rollout     *ldapi.Rollout `json:"rollout,omitempty"`
	Clauses     []ldapi.Clause `json:"clauses,omitempty"`
}

func ffeRulesFromList(ctx context.Context, list types.List) ([]ffeRulePayload, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return []ffeRulePayload{}, diags
	}
	// variation_name/variation_value aren't read here: resolveFFEVariationIndices
	// resolves them into Variation before buildFFEPatches ever calls this
	// function. They must still appear in the struct — Object.As/ElementsAs
	// require an exact 1:1 field match with ffeResourceRuleAttrTypes.
	type ruleModel struct {
		Description    types.String `tfsdk:"description"`
		Variation      types.Int64  `tfsdk:"variation"`
		VariationName  types.String `tfsdk:"variation_name"`
		VariationValue types.String `tfsdk:"variation_value"`
		BucketBy       types.String `tfsdk:"bucket_by"`
		ContextKind    types.String `tfsdk:"context_kind"`
		RolloutWeights types.List   `tfsdk:"rollout_weights"`
		Clauses        types.List   `tfsdk:"clauses"`
	}
	var models []ruleModel
	d := list.ElementsAs(ctx, &models, false)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]ffeRulePayload, 0, len(models))
	for _, m := range models {
		clauses, d := frameworkClausesFromList(ctx, m.Clauses)
		diags.Append(d...)
		var weights []int64
		if !m.RolloutWeights.IsNull() && !m.RolloutWeights.IsUnknown() {
			d = m.RolloutWeights.ElementsAs(ctx, &weights, false)
			diags.Append(d...)
		}
		bucketBy := m.BucketBy.ValueString()
		ck := m.ContextKind.ValueString()
		hasRollout := len(weights) > 0
		if !hasRollout && bucketBy != "" {
			diags.AddError("rules: cannot use bucket_by argument with variation, only with rollout_weights", "")
			return nil, diags
		}
		if !hasRollout && ck != "" && ck != "user" {
			diags.AddError("rules: cannot use context_kind argument with variation, only with rollout_weights", "")
			return nil, diags
		}
		p := ffeRulePayload{Clauses: clauses}
		descStr := m.Description.ValueString()
		p.Description = &descStr
		if hasRollout {
			rollout := &ldapi.Rollout{
				Variations: make([]ldapi.WeightedVariation, 0, len(weights)),
			}
			for i, w := range weights {
				rollout.Variations = append(rollout.Variations, ldapi.WeightedVariation{
					Variation: int32(i),
					Weight:    int32(w),
				})
			}
			if bucketBy != "" {
				rollout.BucketBy = &bucketBy
			}
			if ck != "" {
				rollout.ContextKind = &ck
			}
			p.Rollout = rollout
		} else {
			v := int32(m.Variation.ValueInt64())
			p.Variation = &v
		}
		out = append(out, p)
	}
	return out, diags
}

func ffePrerequisitesFromList(ctx context.Context, list types.List) ([]ldapi.Prerequisite, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return []ldapi.Prerequisite{}, diags
	}
	type prereqModel struct {
		FlagKey   types.String `tfsdk:"flag_key"`
		Variation types.Int64  `tfsdk:"variation"`
	}
	var models []prereqModel
	d := list.ElementsAs(ctx, &models, false)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]ldapi.Prerequisite, 0, len(models))
	for _, m := range models {
		out = append(out, ldapi.Prerequisite{
			Key:       m.FlagKey.ValueString(),
			Variation: int32(m.Variation.ValueInt64()),
		})
	}
	return out, diags
}

func ffeTargetsFromSet(ctx context.Context, set types.Set, isContextTarget bool) ([]ldapi.Target, diag.Diagnostics) {
	var diags diag.Diagnostics
	if set.IsNull() || set.IsUnknown() {
		return []ldapi.Target{}, diags
	}
	if isContextTarget {
		type targetModel struct {
			Values      types.List   `tfsdk:"values"`
			Variation   types.Int64  `tfsdk:"variation"`
			ContextKind types.String `tfsdk:"context_kind"`
		}
		var models []targetModel
		d := set.ElementsAs(ctx, &models, false)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		out := make([]ldapi.Target, 0, len(models))
		for _, m := range models {
			values, d := stringSliceFromList(ctx, m.Values)
			diags.Append(d...)
			ck := m.ContextKind.ValueString()
			out = append(out, ldapi.Target{
				Variation:   int32(m.Variation.ValueInt64()),
				Values:      values,
				ContextKind: &ck,
			})
		}
		return out, diags
	}
	type targetModel struct {
		Values    types.List  `tfsdk:"values"`
		Variation types.Int64 `tfsdk:"variation"`
	}
	var models []targetModel
	d := set.ElementsAs(ctx, &models, false)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]ldapi.Target, 0, len(models))
	for _, m := range models {
		values, d := stringSliceFromList(ctx, m.Values)
		diags.Append(d...)
		ck := "user"
		out = append(out, ldapi.Target{
			Variation:   int32(m.Variation.ValueInt64()),
			Values:      values,
			ContextKind: &ck,
		})
	}
	return out, diags
}

func ffeFallthroughFromObject(ctx context.Context, obj types.Object) (ffeFallthroughPayload, diag.Diagnostics) {
	var diags diag.Diagnostics
	if obj.IsNull() || obj.IsUnknown() {
		diags.AddError("feature flag fallthrough cannot be empty. Please specify at least one of variation or rollout_weights", "")
		return ffeFallthroughPayload{}, diags
	}
	// variation_name/variation_value aren't read here: resolveFFEVariationIndices
	// resolves them into Variation before buildFFEPatches ever calls this
	// function. They must still appear in the struct — Object.As requires
	// an exact 1:1 field match with ffeResourceFallthroughAttrTypes.
	type fallthroughModel struct {
		Variation      types.Int64  `tfsdk:"variation"`
		VariationName  types.String `tfsdk:"variation_name"`
		VariationValue types.String `tfsdk:"variation_value"`
		BucketBy       types.String `tfsdk:"bucket_by"`
		ContextKind    types.String `tfsdk:"context_kind"`
		RolloutWeights types.List   `tfsdk:"rollout_weights"`
	}
	var m fallthroughModel
	d := obj.As(ctx, &m, basetypes.ObjectAsOptions{})
	diags.Append(d...)
	if diags.HasError() {
		return ffeFallthroughPayload{}, diags
	}
	var weights []int64
	if !m.RolloutWeights.IsNull() && !m.RolloutWeights.IsUnknown() {
		d := m.RolloutWeights.ElementsAs(ctx, &weights, false)
		diags.Append(d...)
	}
	bucketBy := m.BucketBy.ValueString()
	ck := m.ContextKind.ValueString()
	if len(weights) == 0 {
		if bucketBy != "" {
			diags.AddError("flag fallthrough: cannot use bucket_by argument with variation, only with rollout_weights", "")
			return ffeFallthroughPayload{}, diags
		}
		v := int32(m.Variation.ValueInt64())
		return ffeFallthroughPayload{Variation: &v}, diags
	}
	rollout := &ldapi.Rollout{
		Variations: make([]ldapi.WeightedVariation, 0, len(weights)),
	}
	for i, w := range weights {
		rollout.Variations = append(rollout.Variations, ldapi.WeightedVariation{
			Variation: int32(i),
			Weight:    int32(w),
		})
	}
	if bucketBy != "" {
		rollout.BucketBy = &bucketBy
	}
	if ck != "" {
		rollout.ContextKind = &ck
	}
	return ffeFallthroughPayload{Rollout: rollout}, diags
}

// flagIDValidator validates strings of the form `project_key/flag_key`.
func flagIDValidator() validator.String {
	return flagIDValidatorType{}
}

type flagIDValidatorType struct{}

func (flagIDValidatorType) Description(_ context.Context) string {
	return "must be in the format `project_key/flag_key`"
}
func (v flagIDValidatorType) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (flagIDValidatorType) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	v := req.ConfigValue.ValueString()
	if strings.Count(v, "/") != 1 {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid flag_id format", fmt.Sprintf("%q must be in the format 'project_key/flag_key'. Got: %s", req.Path, v))
		return
	}
	for _, part := range strings.SplitN(v, "/", 2) {
		if !keyPattern.MatchString(part) {
			resp.Diagnostics.AddAttributeError(req.Path, "Invalid flag_id key", fmt.Sprintf("%q has an invalid key segment %q", req.Path, part))
			return
		}
	}
}

// suppress unused import noise — http used for isStatusNotFound only,
// which receives an *http.Response from r.client APIs.
var _ = http.StatusOK
