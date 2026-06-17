package launchdarkly

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// announcementSeverities is the set of severities accepted by the
// Announcements API. The client docs example uses "warning"; "info" and
// "critical" mirror the in-app banner severities. NOTE (autogen stage 2,
// 2026-06-17): the full enum could not be confirmed against openapi.json in
// this environment (network fetch was unavailable) — a human reviewer must
// verify this list against the live spec / a real apply before merge.
var announcementSeverities = []string{"info", "warning", "critical"}

// announcementListPageSize bounds each page when scanning the list endpoint
// to resolve an announcement by ID (there is no GET-by-ID endpoint).
const announcementListPageSize = 100

var (
	_ resource.Resource                = &AnnouncementResource{}
	_ resource.ResourceWithImportState = &AnnouncementResource{}
)

type AnnouncementResource struct {
	client *Client
}

type AnnouncementResourceModel struct {
	ID            types.String `tfsdk:"id"`
	IsDismissible types.Bool   `tfsdk:"is_dismissible"`
	Title         types.String `tfsdk:"title"`
	Message       types.String `tfsdk:"message"`
	StartTime     types.Int64  `tfsdk:"start_time"`
	EndTime       types.Int64  `tfsdk:"end_time"`
	Severity      types.String `tfsdk:"severity"`
	Status        types.String `tfsdk:"status"`
}

func NewAnnouncementResource() resource.Resource {
	return &AnnouncementResource{}
}

func (r *AnnouncementResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_announcement"
}

func (r *AnnouncementResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly announcement resource.\n\nThis resource allows you to create and manage an in-app announcement banner that appears in the LaunchDarkly user interface for everyone in your organization.",
		Attributes: map[string]schema.Attribute{
			ID: schema.StringAttribute{
				Computed:      true,
				Description:   "The unique announcement ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			TITLE: schema.StringAttribute{
				Required:    true,
				Description: "The title of the announcement.",
			},
			MESSAGE: schema.StringAttribute{
				Required:    true,
				Description: "The body of the announcement. Supports Markdown.",
			},
			SEVERITY: schema.StringAttribute{
				Required:    true,
				Description: "The severity of the announcement. Must be one of " + oxfordCommaJoin(announcementSeverities) + ".",
				Validators: []validator.String{
					stringvalidator.OneOf(announcementSeverities...),
				},
			},
			IS_DISMISSIBLE: schema.BoolAttribute{
				Required:    true,
				Description: "Whether viewers can dismiss the announcement banner.",
			},
			START_TIME: schema.Int64Attribute{
				Required:    true,
				Description: "The time the announcement becomes active, as a Unix timestamp in milliseconds.",
			},
			END_TIME: schema.Int64Attribute{
				Optional:    true,
				Description: "The time the announcement is no longer active, as a Unix timestamp in milliseconds. If omitted, the announcement does not automatically expire.",
			},
			STATUS: schema.StringAttribute{
				Computed:    true,
				Description: "The computed status of the announcement (for example, `active`, `scheduled`, or `inactive`), derived by LaunchDarkly from the current time and the announcement's start and end times.",
			},
		},
	}
}

func (r *AnnouncementResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *AnnouncementResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AnnouncementResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	post := ldapi.NewCreateAnnouncementBody(
		plan.IsDismissible.ValueBool(),
		plan.Title.ValueString(),
		plan.Message.ValueString(),
		plan.StartTime.ValueInt64(),
		plan.Severity.ValueString(),
	)
	if !plan.EndTime.IsNull() && !plan.EndTime.IsUnknown() {
		post.SetEndTime(plan.EndTime.ValueInt64())
	}

	var announcement *ldapi.AnnouncementResponse
	err := r.client.withConcurrency(r.client.ctx, func() error {
		var e error
		announcement, _, e = r.client.ld.AnnouncementsApi.CreateAnnouncementPublic(r.client.ctx).CreateAnnouncementBody(*post).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to create announcement", err)
		return
	}

	announcementToModel(announcement, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AnnouncementResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AnnouncementResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readIntoModel(data.ID.ValueString(), &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AnnouncementResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state AnnouncementResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	title := plan.Title.ValueString()
	message := plan.Message.ValueString()
	severity := plan.Severity.ValueString()
	isDismissible := plan.IsDismissible.ValueBool()
	startTime := plan.StartTime.ValueInt64()

	patch := []ldapi.AnnouncementPatchOperation{
		announcementPatchReplace("/title", title),
		announcementPatchReplace("/message", message),
		announcementPatchReplace("/severity", severity),
		announcementPatchReplace("/isDismissible", isDismissible),
		announcementPatchReplace("/startTime", startTime),
	}
	// endTime is optional: "add" upserts the field when present, "remove"
	// clears it when the user drops a previously-set value.
	if !plan.EndTime.IsNull() && !plan.EndTime.IsUnknown() {
		patch = append(patch, announcementPatchAdd("/endTime", plan.EndTime.ValueInt64()))
	} else if !state.EndTime.IsNull() {
		patch = append(patch, announcementPatchRemove("/endTime"))
	}

	var announcement *ldapi.AnnouncementResponse
	err := r.client.withConcurrency(r.client.ctx, func() error {
		var e error
		announcement, _, e = r.client.ld.AnnouncementsApi.UpdateAnnouncementPublic(r.client.ctx, plan.ID.ValueString()).AnnouncementPatchOperation(patch).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to update announcement", err)
		return
	}

	announcementToModel(announcement, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AnnouncementResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AnnouncementResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var res *http.Response
	err := r.client.withConcurrency(r.client.ctx, func() error {
		var e error
		res, e = r.client.ld.AnnouncementsApi.DeleteAnnouncementPublic(r.client.ctx, data.ID.ValueString()).Execute()
		return e
	})
	if err != nil {
		if isStatusNotFound(res) {
			return
		}
		addLdapiError(&resp.Diagnostics, "Failed to delete announcement", err)
	}
}

func (r *AnnouncementResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root(ID), req, resp)
}

// readIntoModel resolves an announcement by ID through the list endpoint
// (the API exposes no GET-by-ID) and populates data. If the announcement is
// not found, data.ID is set null so Read can remove the resource from state.
func (r *AnnouncementResource) readIntoModel(id string, data *AnnouncementResourceModel, diags *diag.Diagnostics) {
	announcement, found, err := r.getAnnouncementByID(id)
	if err != nil {
		diags.AddError("Failed to get announcement", handleLdapiErr(err).Error())
		return
	}
	if !found {
		data.ID = types.StringNull()
		return
	}
	announcementToModel(announcement, data)
}

// getAnnouncementByID pages through the announcements list endpoint looking
// for a matching ID. NOTE (autogen stage 2, 2026-06-17): the list defaults to
// returning all statuses here (the status filter is left unset); a human
// reviewer should confirm with a real apply that the default does not exclude
// expired/inactive announcements, which would make Read spuriously 404 a
// managed announcement and recreate it.
func (r *AnnouncementResource) getAnnouncementByID(id string) (*ldapi.AnnouncementResponse, bool, error) {
	var offset int32
	for {
		var page *ldapi.GetAnnouncementsPublic200Response
		err := r.client.withConcurrency(r.client.ctx, func() error {
			var e error
			page, _, e = r.client.ld.AnnouncementsApi.GetAnnouncementsPublic(r.client.ctx).
				Limit(announcementListPageSize).
				Offset(offset).
				Execute()
			return e
		})
		if err != nil {
			return nil, false, err
		}
		for i := range page.Items {
			if page.Items[i].Id == id {
				return &page.Items[i], true, nil
			}
		}
		if len(page.Items) < announcementListPageSize {
			return nil, false, nil
		}
		offset += announcementListPageSize
	}
}

// announcementToModel copies an API response into the framework model.
func announcementToModel(announcement *ldapi.AnnouncementResponse, data *AnnouncementResourceModel) {
	data.ID = types.StringValue(announcement.Id)
	data.IsDismissible = types.BoolValue(announcement.IsDismissible)
	data.Title = types.StringValue(announcement.Title)
	data.Message = types.StringValue(announcement.Message)
	data.StartTime = types.Int64Value(announcement.StartTime)
	if announcement.EndTime != nil {
		data.EndTime = types.Int64Value(*announcement.EndTime)
	} else {
		data.EndTime = types.Int64Null()
	}
	data.Severity = types.StringValue(announcement.Severity)
	data.Status = types.StringValue(announcement.Status)
}

func announcementPatchReplace(path string, value interface{}) ldapi.AnnouncementPatchOperation {
	return ldapi.AnnouncementPatchOperation{Op: "replace", Path: path, Value: &value}
}

func announcementPatchAdd(path string, value interface{}) ldapi.AnnouncementPatchOperation {
	return ldapi.AnnouncementPatchOperation{Op: "add", Path: path, Value: &value}
}

func announcementPatchRemove(path string) ldapi.AnnouncementPatchOperation {
	return ldapi.AnnouncementPatchOperation{Op: "remove", Path: path}
}
