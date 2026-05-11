package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var (
	_ resource.Resource                = &ContextKindResource{}
	_ resource.ResourceWithConfigure   = &ContextKindResource{}
	_ resource.ResourceWithImportState = &ContextKindResource{}
)

type ContextKindResource struct {
	client *Client
}

type ContextKindResourceModel struct {
	ProjectKey      types.String `tfsdk:"project_key"`
	Key             types.String `tfsdk:"key"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	HideInTargeting types.Bool   `tfsdk:"hide_in_targeting"`
	Archived        types.Bool   `tfsdk:"archived"`
	Version         types.Int64  `tfsdk:"version"`
	CreationDate    types.Int64  `tfsdk:"creation_date"`
	LastModified    types.Int64  `tfsdk:"last_modified"`
	CreatedFrom     types.String `tfsdk:"created_from"`
	ID              types.String `tfsdk:"id"`
}

func NewContextKindResource() resource.Resource {
	return &ContextKindResource{}
}

func (r *ContextKindResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_context_kind"
}

func (r *ContextKindResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a LaunchDarkly [context kind](https://launchdarkly.com/docs/home/observability/contexts/context-kinds). " +
			"`terraform destroy` archives the kind rather than deleting it — LaunchDarkly does not expose a delete endpoint for " +
			"context kinds. Archived kinds remain in the project but are unavailable for targeting.\n\n" +
			"### Migrating from the `restapi` provider\n\n" +
			"If you currently manage context kinds via the Mastercard `restapi_object` resource, follow this sequence to migrate " +
			"without losing state:\n\n" +
			"1. Add a `launchdarkly_context_kind` resource matching the existing kind.\n" +
			"2. `terraform import launchdarkly_context_kind.<name> <project_key>/<kind_key>`.\n" +
			"3. `terraform plan` to confirm no diff.\n" +
			"4. Remove the `restapi_object` block.\n" +
			"5. Apply.",
		Attributes: map[string]schema.Attribute{
			"project_key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The LaunchDarkly project key that scopes the context kind.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The unique identifier of the context kind within the project. The built-in `user` kind cannot be managed by this resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					contextKindKeyValidator{},
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The human-readable name of the context kind.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "A description of the context kind.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"hide_in_targeting": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Server-side mirror of `archived`. LaunchDarkly aliases `hideInTargeting` to `archived` on writes, so this attribute is read-only. Use `archived` to control targeting visibility.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"archived": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				MarkdownDescription: "Whether the context kind is archived. Archived kinds are unavailable for targeting. " +
					"`terraform destroy` sets this to `true` because the LaunchDarkly API does not expose a delete operation for context kinds.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The LaunchDarkly-assigned version of the context kind. Incremented on every server-side mutation.",
			},
			"creation_date": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Unix epoch (milliseconds) at which the context kind was created.",
			},
			"last_modified": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Unix epoch (milliseconds) of the last server-side modification.",
			},
			"created_from": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "How the context kind was first created (e.g. `api`, `ui`, `sdk`).",
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The composite identifier `<project_key>/<key>`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ContextKindResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData))
		return
	}
	r.client = client
}

func (r *ContextKindResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ContextKindResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	key := data.Key.ValueString()

	existing, _, listErr := r.listContextKinds(ctx, projectKey)
	if listErr != nil {
		resp.Diagnostics.AddError(
			"Unable to verify context kind uniqueness",
			fmt.Sprintf("Received an error listing context kinds for project %q: %s", projectKey, listErr),
		)
		return
	}
	if found, ok := findContextKindByKey(existing, key); ok {
		archived := false
		if found.Archived != nil {
			archived = *found.Archived
		}
		if !archived {
			resp.Diagnostics.AddError(
				"Context kind already exists",
				fmt.Sprintf("A context kind with key %q already exists in project %q. To bring it under Terraform management, run:\n\n  terraform import launchdarkly_context_kind.<name> %s/%s",
					key, projectKey, projectKey, key),
			)
			return
		}
	}

	payload := buildUpsertContextKindPayload(
		data.Name.ValueString(),
		optionalString(data.Description),
		optionalBool(data.HideInTargeting),
		optionalBool(data.Archived),
	)

	if err := r.putContextKind(ctx, projectKey, key, payload); err != nil {
		resp.Diagnostics.AddError(
			"Unable to create context kind",
			fmt.Sprintf("Received an error creating context kind %q in project %q: %s", key, projectKey, err),
		)
		return
	}

	planName := data.Name
	planDescription := data.Description
	planArchived := data.Archived

	r.hydrateFromAPI(ctx, projectKey, key, &data, &resp.Diagnostics, false)
	if resp.Diagnostics.HasError() {
		return
	}

	// LaunchDarkly's project-scoped list endpoint lags writes by ~1-5s. Trust the plan for
	// attributes the user controls so the saved state matches what was requested; the next
	// Read will refresh from the canonical API.
	data.Name = planName
	if !planDescription.IsNull() && !planDescription.IsUnknown() {
		data.Description = planDescription
	}
	if !planArchived.IsNull() && !planArchived.IsUnknown() {
		data.Archived = planArchived
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContextKindResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ContextKindResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectKey := data.ProjectKey.ValueString()
	key := data.Key.ValueString()
	removed := false
	r.hydrateFromAPI(ctx, projectKey, key, &data, &resp.Diagnostics, true)
	if data.Key.IsNull() {
		removed = true
	}
	if removed {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContextKindResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ContextKindResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	key := data.Key.ValueString()

	payload := buildUpsertContextKindPayload(
		data.Name.ValueString(),
		optionalString(data.Description),
		optionalBool(data.HideInTargeting),
		optionalBool(data.Archived),
	)

	if err := r.putContextKind(ctx, projectKey, key, payload); err != nil {
		resp.Diagnostics.AddError(
			"Unable to update context kind",
			fmt.Sprintf("Received an error updating context kind %q in project %q: %s", key, projectKey, err),
		)
		return
	}

	planName := data.Name
	planDescription := data.Description
	planArchived := data.Archived

	r.hydrateFromAPI(ctx, projectKey, key, &data, &resp.Diagnostics, false)
	if resp.Diagnostics.HasError() {
		return
	}

	// LaunchDarkly's project-scoped list endpoint lags writes by ~1-5s and routinely returns
	// the previous version of the kind right after a PUT. Trust the plan for the attributes
	// the user controls so saved state matches the apply; computed fields will refresh on
	// the next Read.
	data.Name = planName
	if !planDescription.IsNull() && !planDescription.IsUnknown() {
		data.Description = planDescription
	}
	if !planArchived.IsNull() && !planArchived.IsUnknown() {
		data.Archived = planArchived
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContextKindResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ContextKindResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	key := data.Key.ValueString()

	archived := true
	hide := true
	payload := ldapi.UpsertContextKindPayload{
		Name:            data.Name.ValueString(),
		Archived:        &archived,
		HideInTargeting: &hide,
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		s := data.Description.ValueString()
		payload.Description = &s
	}

	if err := r.putContextKind(ctx, projectKey, key, payload); err != nil {
		resp.Diagnostics.AddError(
			"Unable to archive context kind",
			fmt.Sprintf("Received an error archiving context kind %q in project %q. LaunchDarkly does not expose a delete endpoint; "+
				"`terraform destroy` archives instead. Error: %s", key, projectKey, err),
		)
		return
	}
}

func (r *ContextKindResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected import ID in the form <project_key>/<context_kind_key>, got %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_key"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *ContextKindResource) listContextKinds(_ context.Context, projectKey string) ([]ldapi.ContextKindRep, *http.Response, error) {
	var items []ldapi.ContextKindRep
	var res *http.Response
	err := r.client.withConcurrency(r.client.ctx, func() error {
		rep, httpRes, listErr := r.client.ld.ContextsApi.GetContextKindsByProjectKey(r.client.ctx, projectKey).Execute()
		res = httpRes
		if listErr != nil {
			return listErr
		}
		if rep != nil {
			items = rep.Items
		}
		return nil
	})
	if err != nil {
		return nil, res, handleLdapiErr(err)
	}
	return items, res, nil
}

func (r *ContextKindResource) putContextKind(_ context.Context, projectKey, key string, payload ldapi.UpsertContextKindPayload) error {
	return r.client.withConcurrency(r.client.ctx, func() error {
		_, _, err := r.client.ld.ContextsApi.PutContextKind(r.client.ctx, projectKey, key).UpsertContextKindPayload(payload).Execute()
		if err != nil {
			return handleLdapiErr(err)
		}
		return nil
	})
}

// hydrateFromAPI reads the canonical kind from the project list and writes the result back
// into data. If allowMissing is true, a missing kind sets data.Key to null so the caller can
// distinguish "not found" from "error" (Read path).
//
// If allowMissing is false (post-write), the call retries with bounded backoff to absorb the
// eventual-consistency window between PUT /context-kinds/<key> and the project-scoped list
// reflecting the write — observed empirically at ~150-500ms on prod LD.
func (r *ContextKindResource) hydrateFromAPI(ctx context.Context, projectKey, key string, data *ContextKindResourceModel, diags *diag.Diagnostics, allowMissing bool) {
	const maxAttempts = 6
	backoff := 200 * time.Millisecond

	var (
		kind     *ldapi.ContextKindRep
		found    bool
		lastErr  error
		lastRes  *http.Response
		lastKeys []string
	)
	for attempt := 0; attempt < maxAttempts; attempt++ {
		items, res, err := r.listContextKinds(ctx, projectKey)
		lastErr = err
		lastRes = res
		if err == nil {
			lastKeys = lastKeys[:0]
			for i := range items {
				lastKeys = append(lastKeys, items[i].Key)
			}
			if k, ok := findContextKindByKey(items, key); ok {
				kind = k
				found = true
				break
			}
		}
		if allowMissing {
			break // Read path: single-shot.
		}
		log.Printf("[DEBUG] context_kind hydrate retry: project=%s want=%s attempt=%d keys=%v err=%v", projectKey, key, attempt, lastKeys, err)
		time.Sleep(backoff)
		if backoff < 2*time.Second {
			backoff *= 2
		}
	}

	if lastErr != nil {
		if isStatusNotFound(lastRes) && allowMissing {
			data.Key = types.StringNull()
			return
		}
		diags.AddError(
			"Unable to read context kinds",
			fmt.Sprintf("Received an error listing context kinds for project %q: %s", projectKey, lastErr),
		)
		return
	}
	if !found {
		if allowMissing {
			data.Key = types.StringNull()
			return
		}
		diags.AddError(
			"Context kind not found after write",
			fmt.Sprintf("Wrote context kind %q to project %q but the project-scoped list still does not return it after %d attempts. Keys observed: %v", key, projectKey, maxAttempts, lastKeys),
		)
		return
	}

	data.ProjectKey = types.StringValue(projectKey)
	data.Key = types.StringValue(kind.Key)
	data.Name = types.StringValue(kind.Name)
	data.Description = types.StringValue(kind.Description)
	if kind.HideInTargeting != nil {
		data.HideInTargeting = types.BoolValue(*kind.HideInTargeting)
	} else {
		data.HideInTargeting = types.BoolValue(false)
	}
	if kind.Archived != nil {
		data.Archived = types.BoolValue(*kind.Archived)
	} else {
		data.Archived = types.BoolValue(false)
	}
	data.Version = types.Int64Value(int64(kind.Version))
	data.CreationDate = types.Int64Value(kind.CreationDate)
	data.LastModified = types.Int64Value(kind.LastModified)
	data.CreatedFrom = types.StringValue(kind.CreatedFrom)
	data.ID = types.StringValue(projectKey + "/" + key)
}

func optionalString(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

func optionalBool(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	b := v.ValueBool()
	return &b
}

// contextKindKeyValidator rejects keys that would shadow LaunchDarkly's built-in `user` kind.
// The API will accept PUTs against `user` and mutate it; protecting against that at plan time
// is friendlier than discovering the corruption at apply time.
type contextKindKeyValidator struct{}

func (contextKindKeyValidator) Description(_ context.Context) string {
	return "rejects context kind keys that conflict with LaunchDarkly built-ins"
}

func (contextKindKeyValidator) MarkdownDescription(_ context.Context) string {
	return "Rejects context kind keys that conflict with LaunchDarkly built-ins (currently: `user`)."
}

func (contextKindKeyValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if req.ConfigValue.ValueString() == "user" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Cannot manage the built-in `user` context kind",
			"LaunchDarkly provisions the `user` context kind in every project automatically. Managing it through "+
				"`launchdarkly_context_kind` risks renaming or archiving a kind that flag evaluations depend on. "+
				"If you need to inspect the `user` kind, use the `launchdarkly_context_kind` data source instead.",
		)
	}
}
