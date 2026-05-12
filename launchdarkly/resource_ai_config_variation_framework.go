package launchdarkly

// Phase 3.6 scaffold for launchdarkly_ai_config_variation. Variations
// are versioned (memory:ai-config-variations.md); PATCH creates new
// version. Full CRUD pending.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &AIConfigVariationResource{}

type AIConfigVariationResource struct{ client *Client }

type AIConfigVariationResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectKey     types.String `tfsdk:"project_key"`
	AIConfigKey    types.String `tfsdk:"config_key"`
	Key            types.String `tfsdk:"key"`
	Name           types.String `tfsdk:"name"`
	Messages       types.List   `tfsdk:"messages"`
	Model          types.String `tfsdk:"model"`
	ModelConfigKey types.String `tfsdk:"model_config_key"`
	Description    types.String `tfsdk:"description"`
	Instructions   types.String `tfsdk:"instructions"`
	ToolKeys       types.Set    `tfsdk:"tool_keys"`
	State          types.String `tfsdk:"state"`
	VariationID    types.String `tfsdk:"variation_id"`
	Version        types.Int64  `tfsdk:"version"`
	CreationDate   types.Int64  `tfsdk:"creation_date"`
}

func NewAIConfigVariationResource() resource.Resource { return &AIConfigVariationResource{} }

func (r *AIConfigVariationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_config_variation"
}

func (r *AIConfigVariationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly AI Config variation resource (versioned).",
		Attributes: map[string]schema.Attribute{
			"id":            schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			PROJECT_KEY:     schema.StringAttribute{Required: true, Validators: []validator.String{keyValidator()}, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			AI_CONFIG_KEY:   schema.StringAttribute{Required: true, Validators: []validator.String{keyValidator()}, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			KEY:             schema.StringAttribute{Required: true, Validators: []validator.String{keyValidator()}, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			NAME:            schema.StringAttribute{Required: true},
			MODEL:           schema.StringAttribute{Optional: true, Computed: true},
			MODEL_CONFIG_KEY: schema.StringAttribute{Optional: true, Computed: true},
			DESCRIPTION:     schema.StringAttribute{Optional: true, Computed: true},
			INSTRUCTIONS:    schema.StringAttribute{Optional: true, Computed: true},
			TOOL_KEYS:       schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType},
			STATE:           schema.StringAttribute{Optional: true, Computed: true, Validators: []validator.String{oneOfValidator{allowed: []string{"archived", "published"}}}},
			VARIATION_ID:    schema.StringAttribute{Computed: true},
			VERSION:         schema.Int64Attribute{Computed: true},
			CREATION_DATE:   schema.Int64Attribute{Computed: true},
		},
		Blocks: map[string]schema.Block{
			MESSAGES: schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						ROLE: schema.StringAttribute{Required: true, Validators: []validator.String{oneOfValidator{allowed: []string{"system", "user", "assistant", "developer"}}}},
						CONTENT: schema.StringAttribute{Required: true},
					},
				},
			},
		},
	}
}

func (r *AIConfigVariationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *AIConfigVariationResource) Create(_ context.Context, _ resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError("launchdarkly_ai_config_variation scaffold", "Phase 3.6 framework body pending.")
}
func (r *AIConfigVariationResource) Read(_ context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.AddError("launchdarkly_ai_config_variation scaffold", "see Create.")
}
func (r *AIConfigVariationResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("launchdarkly_ai_config_variation scaffold", "see Create.")
}
func (r *AIConfigVariationResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("launchdarkly_ai_config_variation scaffold", "see Create.")
}
