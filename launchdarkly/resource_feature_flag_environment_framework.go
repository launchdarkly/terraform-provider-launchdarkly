package launchdarkly

// Phase 4.4 scaffold for launchdarkly_feature_flag_environment.
// Largest schema in the provider: targets, context_targets, rules,
// prerequisites, fallthrough. CRUD body pending; will reuse
// frameworkClausesDataSourceBlock + variants ported in Phase 1.3.7.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &FeatureFlagEnvironmentResource{}

type FeatureFlagEnvironmentResource struct{ client *Client }

type FeatureFlagEnvironmentResourceModel struct {
	ID             types.String `tfsdk:"id"`
	FlagID         types.String `tfsdk:"flag_id"`
	EnvKey         types.String `tfsdk:"env_key"`
	On             types.Bool   `tfsdk:"on"`
	Targets        types.Set    `tfsdk:"targets"`
	ContextTargets types.Set    `tfsdk:"context_targets"`
	Rules          types.List   `tfsdk:"rules"`
	Prerequisites  types.List   `tfsdk:"prerequisites"`
	Fallthrough    types.List   `tfsdk:"fallthrough"`
	TrackEvents    types.Bool   `tfsdk:"track_events"`
	OffVariation   types.Int64  `tfsdk:"off_variation"`
}

func NewFeatureFlagEnvironmentResource() resource.Resource {
	return &FeatureFlagEnvironmentResource{}
}

func (r *FeatureFlagEnvironmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_feature_flag_environment"
}

func (r *FeatureFlagEnvironmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly environment-specific feature flag configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			FLAG_ID: schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			ENV_KEY: schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			ON: schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
			TRACK_EVENTS: schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
			OFF_VARIATION: schema.Int64Attribute{Required: true},
		},
		Blocks: map[string]schema.Block{
			TARGETS: schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						VALUES:    schema.ListAttribute{Required: true, ElementType: types.StringType},
						VARIATION: schema.Int64Attribute{Required: true},
					},
				},
			},
			CONTEXT_TARGETS: schema.SetNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						VALUES:       schema.ListAttribute{Required: true, ElementType: types.StringType},
						VARIATION:    schema.Int64Attribute{Required: true},
						CONTEXT_KIND: schema.StringAttribute{Required: true},
					},
				},
			},
			PREREQUISITES: schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						FLAG_KEY:  schema.StringAttribute{Required: true, Validators: []validator.String{keyValidator()}},
						VARIATION: schema.Int64Attribute{Required: true},
					},
				},
			},
			RULES: schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						DESCRIPTION:     schema.StringAttribute{Optional: true},
						VARIATION:       schema.Int64Attribute{Optional: true},
						BUCKET_BY:       schema.StringAttribute{Optional: true},
						CONTEXT_KIND:    schema.StringAttribute{Optional: true},
						ROLLOUT_WEIGHTS: schema.ListAttribute{Optional: true, ElementType: types.Int64Type},
					},
					Blocks: map[string]schema.Block{
						CLAUSES: schema.ListNestedBlock{
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									ATTRIBUTE:    schema.StringAttribute{Required: true},
									OP:           schema.StringAttribute{Required: true, Validators: []validator.String{opValidator()}},
									VALUES:       schema.ListAttribute{Required: true, ElementType: types.StringType},
									VALUE_TYPE:   schema.StringAttribute{Optional: true, Computed: true},
									NEGATE:       schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
									CONTEXT_KIND: schema.StringAttribute{Optional: true, Computed: true},
								},
							},
						},
					},
				},
			},
			FALLTHROUGH: schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						VARIATION:       schema.Int64Attribute{Optional: true},
						BUCKET_BY:       schema.StringAttribute{Optional: true},
						CONTEXT_KIND:    schema.StringAttribute{Optional: true},
						ROLLOUT_WEIGHTS: schema.ListAttribute{Optional: true, ElementType: types.Int64Type},
					},
				},
			},
		},
	}
}

func (r *FeatureFlagEnvironmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *FeatureFlagEnvironmentResource) Create(_ context.Context, _ resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError("launchdarkly_feature_flag_environment scaffold", "Phase 4.4 framework body pending.")
}
func (r *FeatureFlagEnvironmentResource) Read(_ context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.AddError("launchdarkly_feature_flag_environment scaffold", "see Create.")
}
func (r *FeatureFlagEnvironmentResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("launchdarkly_feature_flag_environment scaffold", "see Create.")
}
func (r *FeatureFlagEnvironmentResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("launchdarkly_feature_flag_environment scaffold", "see Create.")
}
