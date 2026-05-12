package launchdarkly

// Phase 3.5 scaffold for launchdarkly_ai_config.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &AIConfigResource{}

type AIConfigResource struct{ client *Client }

type AIConfigResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	ProjectKey          types.String `tfsdk:"project_key"`
	Key                 types.String `tfsdk:"key"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	Mode                types.String `tfsdk:"mode"`
	Tags                types.Set    `tfsdk:"tags"`
	MaintainerID        types.String `tfsdk:"maintainer_id"`
	MaintainerTeamKey   types.String `tfsdk:"maintainer_team_key"`
	EvaluationMetricKey types.String `tfsdk:"evaluation_metric_key"`
	IsInverted          types.Bool   `tfsdk:"is_inverted"`
	Version             types.Int64  `tfsdk:"version"`
	CreationDate        types.Int64  `tfsdk:"creation_date"`
}

func NewAIConfigResource() resource.Resource { return &AIConfigResource{} }

func (r *AIConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_config"
}

func (r *AIConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly AI Config resource.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			PROJECT_KEY:   schema.StringAttribute{Required: true, Validators: []validator.String{keyValidator()}, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			KEY:           schema.StringAttribute{Required: true, Validators: []validator.String{keyValidator()}, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			NAME:          schema.StringAttribute{Required: true},
			DESCRIPTION:   schema.StringAttribute{Optional: true, Computed: true},
			MODE:          schema.StringAttribute{Optional: true, Computed: true, Validators: []validator.String{oneOfValidator{allowed: []string{"completion", "agent", "judge"}}}, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			TAGS:          schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType},
			MAINTAINER_ID: schema.StringAttribute{Optional: true, Computed: true},
			MAINTAINER_TEAM_KEY: schema.StringAttribute{Optional: true, Computed: true},
			EVALUATION_METRIC_KEY: schema.StringAttribute{Optional: true, Computed: true},
			IS_INVERTED:           schema.BoolAttribute{Optional: true, Computed: true},
			VERSION:               schema.Int64Attribute{Computed: true},
			CREATION_DATE:         schema.Int64Attribute{Computed: true},
		},
	}
}

func (r *AIConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *AIConfigResource) Create(_ context.Context, _ resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError("launchdarkly_ai_config scaffold", "Phase 3.5 framework body pending.")
}
func (r *AIConfigResource) Read(_ context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.AddError("launchdarkly_ai_config scaffold", "see Create.")
}
func (r *AIConfigResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("launchdarkly_ai_config scaffold", "see Create.")
}
func (r *AIConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("launchdarkly_ai_config scaffold", "see Create.")
}
