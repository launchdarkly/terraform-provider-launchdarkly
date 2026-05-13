package launchdarkly

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
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
	_ resource.Resource                = &WebhookResource{}
	_ resource.ResourceWithImportState = &WebhookResource{}
)

type WebhookResource struct {
	client *Client
}

type WebhookResourceModel struct {
	ID         types.String `tfsdk:"id"`
	URL        types.String `tfsdk:"url"`
	Secret     types.String `tfsdk:"secret"`
	On         types.Bool   `tfsdk:"on"`
	Name       types.String `tfsdk:"name"`
	Statements types.List   `tfsdk:"statements"`
	Tags       types.Set    `tfsdk:"tags"`
}

func NewWebhookResource() resource.Resource {
	return &WebhookResource{}
}

func (r *WebhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *WebhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly webhook resource.\n\nThis resource allows you to create and manage webhooks within your LaunchDarkly organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			URL: schema.StringAttribute{
				Required:    true,
				Description: "The URL of the remote webhook.",
			},
			SECRET: schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The secret used to sign the webhook.",
			},
			ON: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Specifies whether the webhook is enabled.",
			},
			NAME: schema.StringAttribute{
				Optional:    true,
				Description: "The webhook's human-readable name.",
			},
			TAGS: schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Tags associated with your resource.",
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(tagValidator()),
				},
			},
		},
		Blocks: map[string]schema.Block{
			STATEMENTS: frameworkPolicyStatementsResourceBlock(false, "List of policy statement blocks used to filter webhook events. For more information on webhook policy filters read [Adding a policy filter](https://docs.launchdarkly.com/integrations/webhooks#adding-a-policy-filter).", ""),
		},
	}
}

func (r *WebhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *WebhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WebhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	stmts, diags := frameworkPolicyStatementsFromList(ctx, plan.Statements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	url := plan.URL.ValueString()
	on := plan.On.ValueBool()
	name := plan.Name.ValueString()
	post := ldapi.WebhookPost{
		Url:        url,
		On:         on,
		Name:       &name,
		Statements: stmts,
	}
	secret := plan.Secret.ValueString()
	if !plan.Secret.IsNull() && !plan.Secret.IsUnknown() && secret != "" {
		post.Secret = &secret
		post.Sign = true
	}

	var webhook *ldapi.Webhook
	err := r.client.withConcurrency(r.client.ctx, func() error {
		var e error
		webhook, _, e = r.client.ld.WebhooksApi.PostWebhook(r.client.ctx).WebhookPost(post).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to create webhook", err)
		return
	}
	plan.ID = types.StringValue(webhook.Id)

	// LD does not accept tags on create — patch them in after.
	tags, diags := stringSliceFromSet(ctx, plan.Tags)
	resp.Diagnostics.Append(diags...)
	patch := []ldapi.PatchOperation{
		patchReplace("/tags", &tags),
	}
	if err := r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.WebhooksApi.PatchWebhook(r.client.ctx, webhook.Id).PatchOperation(patch).Execute()
		return e
	}); err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to update webhook tags", err)
	}

	r.readIntoModel(ctx, webhook.Id, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WebhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readIntoModel(ctx, data.ID.ValueString(), &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state WebhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	stmts, diags := frameworkPolicyStatementsFromList(ctx, plan.Statements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	url := plan.URL.ValueString()
	secret := plan.Secret.ValueString()
	name := plan.Name.ValueString()
	on := plan.On.ValueBool()
	tags, diags := stringSliceFromSet(ctx, plan.Tags)
	resp.Diagnostics.Append(diags...)

	patch := []ldapi.PatchOperation{
		patchReplace("/url", &url),
		patchReplace("/secret", &secret),
		patchReplace("/on", &on),
		patchReplace("/name", &name),
		patchReplace("/tags", &tags),
	}
	if !plan.Statements.Equal(state.Statements) {
		if len(stmts) > 0 {
			patch = append(patch, patchReplace("/statements", &stmts))
		} else {
			patch = append(patch, patchRemove("/statements"))
		}
	}

	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.WebhooksApi.PatchWebhook(r.client.ctx, plan.ID.ValueString()).PatchOperation(patch).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to update webhook", err)
		return
	}

	r.readIntoModel(ctx, plan.ID.ValueString(), &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WebhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, e := r.client.ld.WebhooksApi.DeleteWebhook(r.client.ctx, data.ID.ValueString()).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to delete webhook", err)
	}
}

func (r *WebhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *WebhookResource) readIntoModel(
	ctx context.Context,
	id string,
	data *WebhookResourceModel,
	diags interface{ AddError(string, string) },
) {
	var webhook *ldapi.Webhook
	var res *http.Response
	var err error
	err = r.client.withConcurrency(r.client.ctx, func() error {
		webhook, res, err = r.client.ld.WebhooksApi.GetWebhook(r.client.ctx, id).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError("Failed to get webhook", handleLdapiErr(err).Error())
		return
	}
	data.ID = types.StringValue(webhook.Id)
	data.URL = types.StringValue(webhook.Url)
	data.On = types.BoolValue(webhook.On)
	data.Name = stringValueFromPointer(webhook.Name)
	data.Secret = stringValueFromPointer(webhook.Secret)

	tagsSet, d := setFromStringSlice(ctx, webhook.Tags)
	if d.HasError() {
		for _, e := range d.Errors() {
			diags.AddError(e.Summary(), e.Detail())
		}
	}
	data.Tags = tagsSet

	stmts, d := frameworkPolicyStatementsValue(ctx, webhook.Statements)
	if d.HasError() {
		for _, e := range d.Errors() {
			diags.AddError(e.Summary(), e.Detail())
		}
	}
	data.Statements = stmts
}
