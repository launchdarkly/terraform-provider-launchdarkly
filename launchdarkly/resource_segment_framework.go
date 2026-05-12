package launchdarkly

// Phase 4.2 scaffold for launchdarkly_segment. CRUD pending; framework
// schema reuses clauses_framework helpers from Phase 1.3.5.

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

var _ resource.Resource = &SegmentResource{}

type SegmentResource struct{ client *Client }

type SegmentResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	ProjectKey           types.String `tfsdk:"project_key"`
	EnvKey               types.String `tfsdk:"env_key"`
	Key                  types.String `tfsdk:"key"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	Tags                 types.Set    `tfsdk:"tags"`
	CreationDate         types.Int64  `tfsdk:"creation_date"`
	Included             types.List   `tfsdk:"included"`
	Excluded             types.List   `tfsdk:"excluded"`
	IncludedContexts     types.List   `tfsdk:"included_contexts"`
	ExcludedContexts     types.List   `tfsdk:"excluded_contexts"`
	Rules                types.List   `tfsdk:"rules"`
	Unbounded            types.Bool   `tfsdk:"unbounded"`
	UnboundedContextKind types.String `tfsdk:"unbounded_context_kind"`
	ViewKeys             types.Set    `tfsdk:"view_keys"`
}

func NewSegmentResource() resource.Resource { return &SegmentResource{} }

func (r *SegmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_segment"
}

func (r *SegmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly segment resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			PROJECT_KEY: schema.StringAttribute{
				Required:      true,
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			ENV_KEY: schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			KEY: schema.StringAttribute{
				Required:      true,
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			NAME:          schema.StringAttribute{Required: true},
			DESCRIPTION:   schema.StringAttribute{Optional: true, Computed: true},
			CREATION_DATE: schema.Int64Attribute{Computed: true},
			TAGS:          schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType},
			INCLUDED:      schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType},
			EXCLUDED:      schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType},
			UNBOUNDED: schema.BoolAttribute{
				Optional: true, Computed: true,
				Default:       booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{},
			},
			UNBOUNDED_CONTEXT_KIND: schema.StringAttribute{Optional: true, Computed: true},
			VIEW_KEYS:              schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType},
		},
		Blocks: map[string]schema.Block{
			INCLUDED_CONTEXTS: schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						VALUES:       schema.ListAttribute{Required: true, ElementType: types.StringType},
						CONTEXT_KIND: schema.StringAttribute{Required: true},
					},
				},
			},
			EXCLUDED_CONTEXTS: schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						VALUES:       schema.ListAttribute{Required: true, ElementType: types.StringType},
						CONTEXT_KIND: schema.StringAttribute{Required: true},
					},
				},
			},
			RULES: schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						WEIGHT:               schema.Int64Attribute{Optional: true},
						BUCKET_BY:            schema.StringAttribute{Optional: true},
						ROLLOUT_CONTEXT_KIND: schema.StringAttribute{Optional: true},
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
		},
	}
}

func (r *SegmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *SegmentResource) Create(_ context.Context, _ resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError("launchdarkly_segment scaffold", "Phase 4.2 framework body pending.")
}
func (r *SegmentResource) Read(_ context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.AddError("launchdarkly_segment scaffold", "see Create.")
}
func (r *SegmentResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("launchdarkly_segment scaffold", "see Create.")
}
func (r *SegmentResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("launchdarkly_segment scaffold", "see Create.")
}
