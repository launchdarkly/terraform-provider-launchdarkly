package launchdarkly

// Phase 3.2 scaffold for launchdarkly_audit_log_subscription. CRUD is
// a TODO marker; resource is NOT registered on the framework provider
// yet. SDKv2 path continues serving the resource until promotion.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &AuditLogSubscriptionResource{}

type AuditLogSubscriptionResource struct{ client *Client }

type AuditLogSubscriptionResourceModel struct {
	ID             types.String `tfsdk:"id"`
	IntegrationKey types.String `tfsdk:"integration_key"`
	Name           types.String `tfsdk:"name"`
	Config         types.Map    `tfsdk:"config"`
	Statements     types.List   `tfsdk:"statements"`
	On             types.Bool   `tfsdk:"on"`
	Tags           types.Set    `tfsdk:"tags"`
}

func NewAuditLogSubscriptionResource() resource.Resource {
	return &AuditLogSubscriptionResource{}
}

func (r *AuditLogSubscriptionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_audit_log_subscription"
}

func (r *AuditLogSubscriptionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a LaunchDarkly audit log subscription integration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			INTEGRATION_KEY: schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			NAME:   schema.StringAttribute{Required: true},
			CONFIG: schema.MapAttribute{Required: true, ElementType: types.StringType},
			ON:     schema.BoolAttribute{Required: true},
			TAGS:   schema.SetAttribute{Optional: true, Computed: true, ElementType: types.StringType},
		},
		Blocks: map[string]schema.Block{
			STATEMENTS: frameworkPolicyStatementsResourceBlock(true, "Resources to subscribe to.", ""),
		},
	}
}

func (r *AuditLogSubscriptionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *AuditLogSubscriptionResource) Create(_ context.Context, _ resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError("launchdarkly_audit_log_subscription scaffold", "Phase 3.2 framework body pending.")
}
func (r *AuditLogSubscriptionResource) Read(_ context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.AddError("launchdarkly_audit_log_subscription scaffold", "see Create.")
}
func (r *AuditLogSubscriptionResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("launchdarkly_audit_log_subscription scaffold", "see Create.")
}
func (r *AuditLogSubscriptionResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError("launchdarkly_audit_log_subscription scaffold", "see Create.")
}
