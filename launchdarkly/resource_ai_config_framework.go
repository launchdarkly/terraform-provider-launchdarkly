package launchdarkly

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v23"
)

var (
	_ resource.Resource                     = &AIConfigResource{}
	_ resource.ResourceWithImportState      = &AIConfigResource{}
	_ resource.ResourceWithConfigValidators = &AIConfigResource{}
	_ resource.ResourceWithUpgradeState     = &AIConfigResource{}
)

type AIConfigResource struct {
	client *Client
}

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
	Variations          types.List   `tfsdk:"variations"`
}

func NewAIConfigResource() resource.Resource { return &AIConfigResource{} }

func (r *AIConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_config"
}

// aiConfigVariationSummaryAttrTypes is defined in data_source_ai_config_framework.go.

func (r *AIConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Provides a LaunchDarkly AI Config resource.\n\nThis resource allows you to create and manage AI Configurations within your LaunchDarkly project.",
		Attributes:  aiConfigSchemaAttributes(),
	}
}

func aiConfigSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:      true,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		PROJECT_KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The project key.", true),
			Validators:    []validator.String{keyValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The AI Config's unique key.", true),
			Validators:    []validator.String{keyValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		NAME: schema.StringAttribute{
			Required:    true,
			Description: "The AI Config's human-readable name.",
		},
		DESCRIPTION: schema.StringAttribute{
			Optional:    true,
			Description: "The AI Config's description.",
		},
		MODE: schema.StringAttribute{
			Optional:      true,
			Computed:      true,
			Default:       stringdefault.StaticString("completion"),
			Description:   addForceNewDescription("The AI Config's mode. Must be `completion`, `agent`, or `judge`. Defaults to `completion`.", true),
			Validators:    []validator.String{oneOfValidator{allowed: []string{"completion", "agent", "judge"}}},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		TAGS: schema.SetAttribute{
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			Description: "Tags associated with this AI Config.",
		},
		MAINTAINER_ID: schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "The member ID of the maintainer for this AI Config. Conflicts with `maintainer_team_key`.",
			Validators:  []validator.String{idValidator()},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		MAINTAINER_TEAM_KEY: schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "The team key of the maintainer team for this AI Config. Conflicts with `maintainer_id`.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		EVALUATION_METRIC_KEY: schema.StringAttribute{
			Optional:    true,
			Description: "The key of the evaluation metric associated with this AI Config.",
		},
		IS_INVERTED: schema.BoolAttribute{
			Optional:    true,
			Description: "Whether the evaluation metric is inverted.",
		},
		VERSION: schema.Int64Attribute{
			Computed:    true,
			Description: "The version of the AI Config.",
		},
		CREATION_DATE: schema.Int64Attribute{
			Computed:    true,
			Description: "A timestamp of when the AI Config was created.",
		},
		VARIATIONS: schema.ListAttribute{
			Computed:    true,
			Description: "A list of variation summaries for this AI Config.",
			ElementType: types.ObjectType{AttrTypes: aiConfigVariationSummaryAttrTypes},
		},
	}
}

func (r *AIConfigResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	priorSchema := schema.Schema{Attributes: aiConfigSchemaAttributes()}
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &priorSchema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var data AIConfigResourceModel
				resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
				if resp.Diagnostics.HasError() {
					return
				}
				data.Description = nullIfEmptyString(data.Description)
				data.MaintainerID = nullIfEmptyString(data.MaintainerID)
				data.MaintainerTeamKey = nullIfEmptyString(data.MaintainerTeamKey)
				data.EvaluationMetricKey = nullIfEmptyString(data.EvaluationMetricKey)
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			},
		},
	}
}

func (r *AIConfigResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{aiConfigConflictValidator{}}
}

type aiConfigConflictValidator struct{}

func (aiConfigConflictValidator) Description(context.Context) string {
	return "maintainer_id / maintainer_team_key are mutually exclusive; is_inverted requires evaluation_metric_key"
}
func (aiConfigConflictValidator) MarkdownDescription(context.Context) string { return "" }

func (aiConfigConflictValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data AIConfigResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	idSet := !data.MaintainerID.IsNull() && !data.MaintainerID.IsUnknown() && data.MaintainerID.ValueString() != ""
	teamSet := !data.MaintainerTeamKey.IsNull() && !data.MaintainerTeamKey.IsUnknown() && data.MaintainerTeamKey.ValueString() != ""
	if idSet && teamSet {
		resp.Diagnostics.AddAttributeError(
			path.Root(MAINTAINER_TEAM_KEY),
			"Conflicting maintainer fields",
			"maintainer_id and maintainer_team_key are mutually exclusive; set only one.",
		)
	}

	if !data.IsInverted.IsNull() && !data.IsInverted.IsUnknown() && data.IsInverted.ValueBool() {
		metricSet := !data.EvaluationMetricKey.IsNull() && !data.EvaluationMetricKey.IsUnknown() && data.EvaluationMetricKey.ValueString() != ""
		if !metricSet {
			resp.Diagnostics.AddAttributeError(
				path.Root(IS_INVERTED),
				"is_inverted requires evaluation_metric_key",
				"is_inverted=true requires evaluation_metric_key to be set",
			)
		}
	}
}

func (r *AIConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *AIConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AIConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	configKey := plan.Key.ValueString()
	name := plan.Name.ValueString()

	post := *ldapi.NewAIConfigPost(configKey, name)
	// AIConfigPost ctor sets Description="" which API rejects; clear it
	// so the field is omitted unless the user sets it explicitly.
	post.Description = nil

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() && plan.Description.ValueString() != "" {
		v := plan.Description.ValueString()
		post.Description = &v
	}
	if !plan.Mode.IsNull() && !plan.Mode.IsUnknown() && plan.Mode.ValueString() != "" {
		v := plan.Mode.ValueString()
		post.Mode = &v
	}
	if !plan.MaintainerID.IsNull() && !plan.MaintainerID.IsUnknown() && plan.MaintainerID.ValueString() != "" {
		v := plan.MaintainerID.ValueString()
		post.MaintainerId = &v
	}
	if !plan.MaintainerTeamKey.IsNull() && !plan.MaintainerTeamKey.IsUnknown() && plan.MaintainerTeamKey.ValueString() != "" {
		v := plan.MaintainerTeamKey.ValueString()
		post.MaintainerTeamKey = &v
	}
	if !plan.EvaluationMetricKey.IsNull() && !plan.EvaluationMetricKey.IsUnknown() && plan.EvaluationMetricKey.ValueString() != "" {
		v := plan.EvaluationMetricKey.ValueString()
		post.EvaluationMetricKey = &v
		// is_inverted is meaningful only with an evaluation_metric_key.
		// Only send it when configured — the API defaults it to false.
		if !plan.IsInverted.IsNull() && !plan.IsInverted.IsUnknown() {
			isInverted := plan.IsInverted.ValueBool()
			post.IsInverted = &isInverted
		}
	}
	tags, d := stringSliceFromSet(ctx, plan.Tags)
	resp.Diagnostics.Append(d...)
	if len(tags) > 0 {
		post.Tags = tags
	}

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.AgentControlApi.PostAIConfig(r.client.ctx, projectKey).AIConfigPost(post).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("failed to create AI config with key %q in project %q", configKey, projectKey), err)
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/%s", projectKey, configKey))
	r.readIntoModel(ctx, projectKey, configKey, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AIConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AIConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readIntoModel(ctx, data.ProjectKey.ValueString(), data.Key.ValueString(), &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AIConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state AIConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	configKey := plan.Key.ValueString()

	patch := *ldapi.NewAIConfigPatch()
	hasChanges := false

	if !plan.Name.Equal(state.Name) {
		v := plan.Name.ValueString()
		patch.Name = &v
		hasChanges = true
	}
	if !plan.Description.Equal(state.Description) {
		v := plan.Description.ValueString()
		patch.Description = &v
		hasChanges = true
	}
	if !plan.MaintainerID.Equal(state.MaintainerID) {
		v := plan.MaintainerID.ValueString()
		patch.MaintainerId = &v
		hasChanges = true
	}
	if !plan.MaintainerTeamKey.Equal(state.MaintainerTeamKey) {
		v := plan.MaintainerTeamKey.ValueString()
		patch.MaintainerTeamKey = &v
		hasChanges = true
	}
	if !plan.Tags.Equal(state.Tags) {
		tags, d := stringSliceFromSet(ctx, plan.Tags)
		resp.Diagnostics.Append(d...)
		patch.Tags = tags
		hasChanges = true
	}
	if !plan.EvaluationMetricKey.Equal(state.EvaluationMetricKey) {
		v := plan.EvaluationMetricKey.ValueString()
		patch.EvaluationMetricKey = &v
		hasChanges = true
	}
	if !plan.IsInverted.Equal(state.IsInverted) {
		v := plan.IsInverted.ValueBool()
		patch.IsInverted = &v
		hasChanges = true
	}

	if resp.Diagnostics.HasError() {
		return
	}

	if hasChanges {
		err := r.client.withConcurrency(r.client.ctx, func() error {
			_, _, e := r.client.ld.AgentControlApi.PatchAIConfig(r.client.ctx, projectKey, configKey).AIConfigPatch(patch).Execute()
			return e
		})
		if err != nil {
			addLdapiError(&resp.Diagnostics, fmt.Sprintf("failed to update AI config with key %q in project %q", configKey, projectKey), err)
			return
		}
	}

	r.readIntoModel(ctx, projectKey, configKey, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete retries on a transient 400 "Could not delete AI config" — the
// LD API needs all variations dereferenced before the config itself
// can go.
func (r *AIConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AIConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectKey := data.ProjectKey.ValueString()
	configKey := data.Key.ValueString()

	deadline := time.Now().Add(aiConfigDeleteRetryTimeout)
	var lastErr error
	for {
		var res *http.Response
		err := r.client.withConcurrency(r.client.ctx, func() error {
			var e error
			res, e = r.client.ld.AgentControlApi.DeleteAIConfig(r.client.ctx, projectKey, configKey).Execute()
			return e
		})
		if err == nil || isStatusNotFound(res) {
			return
		}
		lastErr = err
		if !shouldRetryAIConfigDelete(res, err) || time.Now().After(deadline) {
			break
		}
		log.Printf("[DEBUG] retrying AI config delete for %q in project %q after transient 400: %s", configKey, projectKey, handleLdapiErr(err))
		time.Sleep(2 * time.Second)
	}
	addLdapiError(&resp.Diagnostics, fmt.Sprintf("failed to delete AI config with key %q in project %q", configKey, projectKey), lastErr)
}

func (r *AIConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectKey, configKey, err := aiConfigIdToKeys(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), projectKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(KEY), configKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *AIConfigResource) readIntoModel(
	ctx context.Context,
	projectKey, configKey string,
	data *AIConfigResourceModel,
	diags *diag.Diagnostics,
) {
	var cfg *ldapi.AIConfig
	var res *http.Response
	var err error
	err = r.client.withConcurrency(r.client.ctx, func() error {
		cfg, res, err = r.client.ld.AgentControlApi.GetAIConfig(r.client.ctx, projectKey, configKey).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError(fmt.Sprintf("failed to get AI config with key %q in project %q", configKey, projectKey), handleLdapiErr(err).Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", projectKey, configKey))
	data.ProjectKey = types.StringValue(projectKey)
	data.Key = types.StringValue(cfg.Key)
	data.Name = types.StringValue(cfg.Name)
	data.Description = stringValueOrNull(cfg.Description)
	data.Version = types.Int64Value(int64(cfg.Version))
	data.CreationDate = types.Int64Value(cfg.CreatedAt)

	mode := "completion"
	if cfg.Mode != nil {
		mode = *cfg.Mode
	}
	data.Mode = types.StringValue(mode)

	data.EvaluationMetricKey = stringValueOrNullFromPointer(cfg.EvaluationMetricKey)
	// The API reports isInverted=false even when it was never set. Preserve
	// null when the caller-supplied value (plan on writes, state on refresh)
	// is null and the API reports the default — false is not new information;
	// surfacing it would make an unset attribute fail plan/apply consistency.
	if cfg.IsInverted != nil && (*cfg.IsInverted || (!data.IsInverted.IsNull() && !data.IsInverted.IsUnknown())) {
		data.IsInverted = types.BoolValue(*cfg.IsInverted)
	} else if cfg.IsInverted == nil {
		data.IsInverted = types.BoolNull()
	}

	// Clear both maintainer fields first, then set the one returned by
	// the API — prevents stale values from persisting across maintainer
	// kind changes.
	data.MaintainerID = types.StringValue("")
	data.MaintainerTeamKey = types.StringValue("")
	maintainer := cfg.GetMaintainer()
	if maintainer.MaintainerMember != nil {
		data.MaintainerID = types.StringValue(maintainer.MaintainerMember.GetId())
	}
	if maintainer.AiConfigsMaintainerTeam != nil {
		data.MaintainerTeamKey = types.StringValue(maintainer.AiConfigsMaintainerTeam.GetKey())
	}

	tagsSet, d := setFromStringSlice(ctx, cfg.Tags)
	diags.Append(d...)
	data.Tags = tagsSet

	// Variations summary
	objectType := types.ObjectType{AttrTypes: aiConfigVariationSummaryAttrTypes}
	elems := make([]attr.Value, 0, len(cfg.Variations))
	for _, v := range cfg.Variations {
		obj, d := types.ObjectValue(aiConfigVariationSummaryAttrTypes, map[string]attr.Value{
			KEY:          types.StringValue(v.Key),
			NAME:         types.StringValue(v.Name),
			VARIATION_ID: types.StringValue(v.Id),
		})
		diags.Append(d...)
		elems = append(elems, obj)
	}
	list, d := types.ListValue(objectType, elems)
	diags.Append(d...)
	data.Variations = list
}

// aiConfigIdToKeys lives in ai_config_helper.go.

const aiConfigDeleteRetryTimeout = 45 * time.Second

func shouldRetryAIConfigDelete(res *http.Response, err error) bool {
	if err == nil || res == nil || res.StatusCode != http.StatusBadRequest {
		return false
	}
	errMsg := strings.ToLower(handleLdapiErr(err).Error())
	return strings.Contains(errMsg, "could not delete ai config")
}
