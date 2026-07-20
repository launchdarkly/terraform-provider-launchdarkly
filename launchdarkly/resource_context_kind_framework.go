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
	ldapi "github.com/launchdarkly/api-client-go/v23"
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
		MarkdownDescription: "Manages a LaunchDarkly [context kind](https://launchdarkly.com/docs/home/flags/context-kinds). " +
			"`terraform destroy` archives the kind rather than deleting it. LaunchDarkly does not expose a delete endpoint for " +
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
			PROJECT_KEY: schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The LaunchDarkly project key that scopes the context kind.",
				Validators:          []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			KEY: schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The unique identifier of the context kind within the project. The built-in `user` kind cannot be managed by this resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					keyValidator(),
					contextKindKeyValidator{},
				},
			},
			NAME: schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The human-readable name of the context kind.",
			},
			DESCRIPTION: schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "A description of the context kind.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			HIDE_IN_TARGETING: schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Server-side mirror of `archived`. LaunchDarkly aliases `hideInTargeting` to `archived` on writes, so this attribute is read-only. Use `archived` to control targeting visibility.",
				PlanModifiers: []planmodifier.Bool{
					mirrorArchivedPlanModifier{},
				},
			},
			ARCHIVED: schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				MarkdownDescription: "Whether the context kind is archived. Archived kinds are unavailable for targeting. " +
					"`terraform destroy` sets this to `true` because the LaunchDarkly API does not expose a delete operation for context kinds.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			VERSION: schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The LaunchDarkly-assigned version of the context kind. Incremented on every server-side mutation.",
			},
			CREATION_DATE: schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Unix epoch (milliseconds) at which the context kind was created.",
			},
			LAST_MODIFIED: schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Unix epoch (milliseconds) of the last server-side modification.",
			},
			CREATED_FROM: schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "How the context kind was first created. For example, `api`, `ui`, or `sdk`.",
			},
			ID: schema.StringAttribute{
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
	r.client = configureResourceClient(req, resp)
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
	// Resurrecting an archived kind via PUT bumps its existing version. Track the pre-PUT
	// version so the post-write hydrate doesn't accept a stale list entry left over from
	// before this Create.
	var priorVersion int64
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
		priorVersion = int64(found.Version)
	}

	payload := buildUpsertContextKindPayload(
		data.Name.ValueString(),
		stringPointerFromAttr(data.Description),
		boolPointerFromAttr(data.HideInTargeting),
		boolPointerFromAttr(data.Archived),
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

	r.hydrateFromAPI(ctx, projectKey, key, &data, &resp.Diagnostics, false, priorVersion+1)
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
	// Require the list endpoint to be at least as fresh as our existing state. Without this
	// floor, the refresh that runs between Apply and Plan can read a pre-PUT list and overwrite
	// the user-controlled fields that Update just trusted from the plan, producing a spurious
	// non-empty plan in the next step. state.Version is 0 on import (no floor enforced).
	minVersion := data.Version.ValueInt64()
	removed := false
	r.hydrateFromAPI(ctx, projectKey, key, &data, &resp.Diagnostics, true, minVersion)
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

	var state ContextKindResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	key := data.Key.ValueString()

	payload := buildUpsertContextKindPayload(
		data.Name.ValueString(),
		stringPointerFromAttr(data.Description),
		boolPointerFromAttr(data.HideInTargeting),
		boolPointerFromAttr(data.Archived),
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

	r.hydrateFromAPI(ctx, projectKey, key, &data, &resp.Diagnostics, false, state.Version.ValueInt64()+1)
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
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(KEY), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(ID), req.ID)...)
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
// into data. allowMissing=true is the Read path: a missing kind sets data.Key to null so the
// caller can distinguish "not found" from "error".
//
// minVersion is the freshness floor: the loop treats a kind whose API version is below
// minVersion as "not visible yet" and keeps retrying. LD bumps version on every PUT, so the
// post-write callers (Create / Update) pass priorVersion+1 to ensure they don't read pre-PUT
// state through the project-scoped list, which lags writes by ~150ms-5s. Read passes the
// current state.Version so a refresh between Update and Plan can't accept a stale list result
// that would mask the apply's user-controlled fields.
//
// allowMissing=true callers accept a stale-but-found result if the retry budget is exhausted —
// better to refresh with the last value LD gave us than to error a Read on transient lag.
// allowMissing=false callers surface a diagnostic on exhaustion because the apply already
// claimed success.
func (r *ContextKindResource) hydrateFromAPI(ctx context.Context, projectKey, key string, data *ContextKindResourceModel, diags *diag.Diagnostics, allowMissing bool, minVersion int64) {
	const maxAttempts = 6
	backoff := 200 * time.Millisecond

	var (
		kind     *ldapi.ContextKindRep
		found    bool
		stale    bool
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
				if int64(k.Version) >= minVersion {
					stale = false
					break
				}
				stale = true
			}
		}
		if !found && allowMissing {
			break // Read path: kind genuinely absent; surface deletion.
		}
		log.Printf("[DEBUG] context_kind hydrate retry: project=%s want=%s minVersion=%d attempt=%d stale=%v keys=%v err=%v", projectKey, key, minVersion, attempt, stale, lastKeys, err)
		time.Sleep(backoff)
		if backoff < 2*time.Second {
			backoff *= 2
		}
	}

	if lastErr != nil && !found {
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
	if stale && !allowMissing {
		diags.AddError(
			"Context kind not yet visible at expected version",
			fmt.Sprintf("Wrote context kind %q to project %q but after %d attempts the project-scoped list still returns version %d (need >= %d). LD's list endpoint is eventually consistent — re-run terraform apply.", key, projectKey, maxAttempts, kind.Version, minVersion),
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

// mirrorArchivedPlanModifier sets hide_in_targeting's plan value to whatever archived resolves
// to in the same plan. LD aliases the two server-side, so a plan that left hide_in_targeting
// at its prior state value would mispredict the API's response and trip terraform-core's
// post-apply consistency check (was X, but now Y).
type mirrorArchivedPlanModifier struct{}

func (mirrorArchivedPlanModifier) Description(_ context.Context) string {
	return "mirrors the plan's archived value into hide_in_targeting (LD aliases the two server-side)"
}

func (mirrorArchivedPlanModifier) MarkdownDescription(_ context.Context) string {
	return "Mirrors the plan's `archived` value into `hide_in_targeting` (LD aliases the two server-side)."
}

func (mirrorArchivedPlanModifier) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	var planArchived types.Bool
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root(ARCHIVED), &planArchived)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if planArchived.IsNull() || planArchived.IsUnknown() {
		return
	}
	resp.PlanValue = planArchived
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
