package launchdarkly

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v23"
)

var (
	_ resource.Resource                     = &AIConfigVariationResource{}
	_ resource.ResourceWithImportState      = &AIConfigVariationResource{}
	_ resource.ResourceWithConfigValidators = &AIConfigVariationResource{}
	_ resource.ResourceWithUpgradeState     = &AIConfigVariationResource{}
)

type AIConfigVariationResource struct {
	client *Client
}

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
	Judges         types.Map    `tfsdk:"judges"`
	State          types.String `tfsdk:"state"`
	VariationID    types.String `tfsdk:"variation_id"`
	Version        types.Int64  `tfsdk:"version"`
	CreationDate   types.Int64  `tfsdk:"creation_date"`
}

func NewAIConfigVariationResource() resource.Resource { return &AIConfigVariationResource{} }

func (r *AIConfigVariationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_config_variation"
}

// aiConfigVariationMessageAttrTypes is defined in data_source_ai_config_variation_framework.go.

func (r *AIConfigVariationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Provides a LaunchDarkly AI Config variation resource.\n\nThis resource allows you to create and manage AI Config variations within your LaunchDarkly project.",
		Attributes:  aiConfigVariationSchemaAttributes(),
	}
}

func aiConfigVariationSchemaAttributes() map[string]schema.Attribute {
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
		AI_CONFIG_KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The AI Config key that this variation belongs to.", true),
			Validators:    []validator.String{keyValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The variation's unique key.", true),
			Validators:    []validator.String{keyValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		NAME: schema.StringAttribute{
			Required:    true,
			Description: "The variation's human-readable name.",
		},
		MODEL: schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "A JSON string representing the inline model configuration for the variation. Conflicts with `model_config_key`.",
			Validators:  []validator.String{jsonStringValidator{}},
			PlanModifiers: []planmodifier.String{
				jsonNormalizePlanModifier{},
			},
			// Intentionally no UseStateForUnknown: model and
			// model_config_key are mutually exclusive; switching
			// from inline model to model_config_key must let plan
			// recompute.
		},
		MODEL_CONFIG_KEY: schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "The key of a model config resource to use for this variation. Conflicts with `model`.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		DESCRIPTION: schema.StringAttribute{
			Optional:    true,
			Description: "The variation's description. Used in agent mode.",
		},
		INSTRUCTIONS: schema.StringAttribute{
			Optional:    true,
			Description: "The variation's instructions. Used in agent mode.",
		},
		TOOL_KEYS: schema.SetAttribute{
			Optional:    true,
			Computed:    true,
			ElementType: types.StringType,
			Description: "A set of AI tool keys to associate with this variation.",
			PlanModifiers: []planmodifier.Set{
				setplanmodifier.UseStateForUnknown(),
			},
		},
		JUDGES: schema.MapNestedAttribute{
			Optional:    true,
			Description: "The judges attached to this variation, keyed by the key of the judge AI Config (an AI Config with `mode = \"judge\"`). Applying this attribute replaces all judge attachments on the variation; removing it detaches all judges.",
			Validators: []validator.Map{
				// Reject `judges = {}`: the Read path reports a variation
				// with no judge attachments as null, so allowing an explicit
				// empty map would cause a plan/apply inconsistency. Omit the
				// attribute to detach all judges.
				mapvalidator.SizeAtLeast(1),
				mapvalidator.KeysAre(keyValidator()),
			},
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					SAMPLING_RATE: schema.Float64Attribute{
						Required:    true,
						Description: "The fraction of generations this judge evaluates. Must be between `0.0` and `1.0`. Stored with 32-bit float precision.",
						Validators:  []validator.Float64{float64validator.Between(0, 1)},
						PlanModifiers: []planmodifier.Float64{
							float32PrecisionPlanModifier{},
						},
					},
				},
			},
		},
		STATE: schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Description: "The state of the variation. Must be `archived` or `published`.",
			Validators:  []validator.String{oneOfValidator{allowed: []string{"archived", "published"}}},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		VARIATION_ID: schema.StringAttribute{
			Computed:    true,
			Description: "The internal ID of the variation.",
			// Intentionally no UseStateForUnknown: variation_id can
			// change between PATCHes because the AI Config API
			// versions variations as immutable entities — every
			// update creates a new variation row with a new ID.
		},
		VERSION: schema.Int64Attribute{
			Computed:    true,
			Description: "The version number of the variation.",
			// Increments on every PATCH; plan flap is the intended
			// signal.
		},
		CREATION_DATE: schema.Int64Attribute{
			Computed:    true,
			Description: "The creation timestamp of the variation.",
			// Refreshes on every PATCH (new version row).
		},
		MESSAGES: schema.ListNestedAttribute{
			Optional:    true,
			Description: "A list of messages for completion mode. Each message has a `role` and `content`.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					ROLE: schema.StringAttribute{
						Required:    true,
						Description: "The role of the message. Must be one of `system`, `user`, `assistant`, or `developer`.",
						Validators:  []validator.String{oneOfValidator{allowed: []string{"system", "user", "assistant", "developer"}}},
					},
					CONTENT: schema.StringAttribute{
						Required:    true,
						Description: "The content of the message.",
					},
				},
			},
		},
	}
}

func (r *AIConfigVariationResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	priorSchema := schema.Schema{Attributes: aiConfigVariationSchemaAttributes()}
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &priorSchema,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var data AIConfigVariationResourceModel
				resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
				if resp.Diagnostics.HasError() {
					return
				}
				data.Description = nullIfEmptyString(data.Description)
				data.Instructions = nullIfEmptyString(data.Instructions)
				data.Model = nullIfEmptyString(data.Model)
				data.ModelConfigKey = nullIfEmptyString(data.ModelConfigKey)
				data.State = nullIfEmptyString(data.State)
				data.Messages = nullIfEmptyList(ctx, data.Messages)
				data.ToolKeys = nullIfEmptySet(ctx, data.ToolKeys)
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			},
		},
	}
}

func (r *AIConfigVariationResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{modelVsModelConfigKeyValidator{}}
}

type modelVsModelConfigKeyValidator struct{}

func (modelVsModelConfigKeyValidator) Description(context.Context) string {
	return "model and model_config_key are mutually exclusive"
}
func (modelVsModelConfigKeyValidator) MarkdownDescription(context.Context) string { return "" }
func (modelVsModelConfigKeyValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data AIConfigVariationResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	modelSet := !data.Model.IsNull() && !data.Model.IsUnknown() && data.Model.ValueString() != ""
	keySet := !data.ModelConfigKey.IsNull() && !data.ModelConfigKey.IsUnknown() && data.ModelConfigKey.ValueString() != ""
	if modelSet && keySet {
		resp.Diagnostics.AddAttributeError(
			path.Root(MODEL_CONFIG_KEY),
			"Conflicting model fields",
			"model and model_config_key are mutually exclusive; set only one.",
		)
	}
}

func (r *AIConfigVariationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *AIConfigVariationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AIConfigVariationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	configKey := plan.AIConfigKey.ValueString()
	variationKey := plan.Key.ValueString()
	name := plan.Name.ValueString()

	post := ldapi.NewAIConfigVariationPost(variationKey, name)
	post.Model = map[string]interface{}{}

	if !plan.Model.IsNull() && !plan.Model.IsUnknown() && plan.Model.ValueString() != "" {
		m, err := jsonStringToMap(plan.Model.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid model JSON", err.Error())
			return
		}
		if m != nil {
			post.Model = m
		}
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() && plan.Description.ValueString() != "" {
		v := plan.Description.ValueString()
		post.Description = &v
	}
	if !plan.Instructions.IsNull() && !plan.Instructions.IsUnknown() && plan.Instructions.ValueString() != "" {
		v := plan.Instructions.ValueString()
		post.Instructions = &v
	}
	if !plan.ModelConfigKey.IsNull() && !plan.ModelConfigKey.IsUnknown() && plan.ModelConfigKey.ValueString() != "" {
		v := plan.ModelConfigKey.ValueString()
		post.ModelConfigKey = &v
	}

	msgs, d := variationMessagesFromList(ctx, plan.Messages)
	resp.Diagnostics.Append(d...)
	if len(msgs) > 0 {
		post.Messages = msgs
	}

	toolKeys, d := stringSliceFromSet(ctx, plan.ToolKeys)
	resp.Diagnostics.Append(d...)
	if len(toolKeys) > 0 {
		post.ToolKeys = toolKeys
	}

	judges, d := variationJudgesFromMap(ctx, plan.Judges)
	resp.Diagnostics.Append(d...)
	if len(judges) > 0 {
		jc := ldapi.NewJudgeConfiguration()
		jc.Judges = judges
		post.JudgeConfiguration = jc
	}

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.AgentControlApi.PostAIConfigVariation(r.client.ctx, projectKey, configKey).AIConfigVariationPost(*post).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("failed to create AI config variation with key %q in config %q project %q", variationKey, configKey, projectKey), err)
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", projectKey, configKey, variationKey))
	r.readIntoModelWithRetry(ctx, projectKey, configKey, variationKey, 0, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AIConfigVariationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AIConfigVariationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readIntoModel(ctx, data.ProjectKey.ValueString(), data.AIConfigKey.ValueString(), data.Key.ValueString(), &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AIConfigVariationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state AIConfigVariationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	configKey := plan.AIConfigKey.ValueString()
	variationKey := plan.Key.ValueString()

	patch := ldapi.NewAIConfigVariationPatch()
	if !plan.Name.Equal(state.Name) {
		v := plan.Name.ValueString()
		patch.Name = &v
	}
	if !plan.Description.Equal(state.Description) {
		v := plan.Description.ValueString()
		patch.Description = &v
	}
	if !plan.Instructions.Equal(state.Instructions) {
		v := plan.Instructions.ValueString()
		patch.Instructions = &v
	}
	if !plan.Model.Equal(state.Model) {
		m, err := jsonStringToMap(plan.Model.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid model JSON", err.Error())
			return
		}
		patch.Model = m
	}
	if !plan.ModelConfigKey.Equal(state.ModelConfigKey) {
		v := plan.ModelConfigKey.ValueString()
		patch.ModelConfigKey = &v
	}
	if !plan.Messages.Equal(state.Messages) {
		msgs, d := variationMessagesFromList(ctx, plan.Messages)
		resp.Diagnostics.Append(d...)
		patch.Messages = msgs
	}
	if !plan.State.Equal(state.State) {
		v := plan.State.ValueString()
		patch.State = &v
	}
	if !plan.ToolKeys.Equal(state.ToolKeys) {
		tks, d := stringSliceFromSet(ctx, plan.ToolKeys)
		resp.Diagnostics.Append(d...)
		patch.ToolKeys = tks
	}
	if !plan.Judges.Equal(state.Judges) {
		judges, d := variationJudgesFromMap(ctx, plan.Judges)
		resp.Diagnostics.Append(d...)
		if judges == nil {
			// A non-nil empty slice serializes as `"judges": []`, which the
			// API treats as "remove all judge attachments".
			judges = []ldapi.JudgeAttachment{}
		}
		jc := ldapi.NewJudgeConfiguration()
		jc.Judges = judges
		patch.JudgeConfiguration = jc
	}

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.AgentControlApi.PatchAIConfigVariation(r.client.ctx, projectKey, configKey, variationKey).AIConfigVariationPatch(*patch).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("failed to update AI config variation with key %q in config %q project %q", variationKey, configKey, projectKey), err)
		return
	}

	previousVersion := state.Version.ValueInt64()
	r.readIntoModelWithRetry(ctx, projectKey, configKey, variationKey, previousVersion, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AIConfigVariationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AIConfigVariationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectKey := data.ProjectKey.ValueString()
	configKey := data.AIConfigKey.ValueString()
	variationKey := data.Key.ValueString()

	var res *http.Response
	err := r.client.withConcurrency(r.client.ctx, func() error {
		var e error
		res, e = r.client.ld.AgentControlApi.DeleteAIConfigVariation(r.client.ctx, projectKey, configKey, variationKey).Execute()
		return e
	})
	if err == nil || isStatusNotFound(res) {
		return
	}
	// "Cannot delete the last variation" — parent AI config delete will cascade.
	if strings.Contains(handleLdapiErr(err).Error(), "Cannot delete the last variation") {
		log.Printf("[WARN] cannot delete last variation %q in config %q project %q — will be removed when parent AI config is deleted", variationKey, configKey, projectKey)
		return
	}
	addLdapiError(&resp.Diagnostics, fmt.Sprintf("failed to delete AI config variation with key %q in config %q project %q", variationKey, configKey, projectKey), err)
}

func (r *AIConfigVariationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectKey, configKey, variationKey, err := variationIdToKeys(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), projectKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(AI_CONFIG_KEY), configKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(KEY), variationKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// readIntoModelWithRetry calls readIntoModel up to 10 times, waiting for
// the API's GET to return a version greater than `previousVersion`. Each
// PATCH creates a new version server-side, and the GET may briefly
// return the prior version due to eventual consistency.
func (r *AIConfigVariationResource) readIntoModelWithRetry(
	ctx context.Context,
	projectKey, configKey, variationKey string,
	previousVersion int64,
	data *AIConfigVariationResourceModel,
	diags *diag.Diagnostics,
) {
	const maxAttempts = 10
	deadline := time.Now().Add(30 * time.Second)
	for attempt := 0; attempt < maxAttempts; attempt++ {
		var local diag.Diagnostics
		r.readIntoModel(ctx, projectKey, configKey, variationKey, data, &local)
		if local.HasError() {
			diags.Append(local...)
			return
		}
		current := data.Version.ValueInt64()
		if previousVersion == 0 || current > previousVersion {
			diags.Append(local...)
			return
		}
		if time.Now().After(deadline) {
			diags.AddError(
				"version did not advance",
				fmt.Sprintf("AI config variation %q: version did not advance past %d after %d reads (current %d)", variationKey, previousVersion, attempt+1, current),
			)
			return
		}
		log.Printf("[DEBUG] AI config variation %q: version %d has not advanced past %d, retrying read", variationKey, current, previousVersion)
		time.Sleep(2 * time.Second)
	}
}

func (r *AIConfigVariationResource) readIntoModel(
	ctx context.Context,
	projectKey, configKey, variationKey string,
	data *AIConfigVariationResourceModel,
	diags *diag.Diagnostics,
) {
	var variationsResp *ldapi.AIConfigVariationsResponse
	var res *http.Response
	var err error
	err = r.client.withConcurrency(r.client.ctx, func() error {
		variationsResp, res, err = r.client.ld.AgentControlApi.GetAIConfigVariation(r.client.ctx, projectKey, configKey, variationKey).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError(fmt.Sprintf("failed to get AI config variation with key %q in config %q project %q", variationKey, configKey, projectKey), handleLdapiErr(err).Error())
		return
	}
	if variationsResp == nil || len(variationsResp.Items) == 0 {
		data.ID = types.StringNull()
		return
	}

	// Pick the highest-version item.
	variation := variationsResp.Items[0]
	for _, v := range variationsResp.Items[1:] {
		if v.Version > variation.Version {
			variation = v
		}
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", projectKey, configKey, variationKey))
	data.ProjectKey = types.StringValue(projectKey)
	data.AIConfigKey = types.StringValue(configKey)
	data.Key = types.StringValue(variation.Key)
	data.Name = types.StringValue(variation.Name)
	data.VariationID = types.StringValue(variation.Id)
	data.Version = types.Int64Value(int64(variation.Version))
	data.CreationDate = types.Int64Value(variation.CreatedAt)

	// description and instructions are returned by POST/PATCH but not by
	// GET /variations — see https://app.launchdarkly.com/api/v2 schema:
	// the items array elides them. When the API omits them, preserve
	// the caller-supplied value (plan during Update, state during
	// Refresh) so terraform doesn't see a write-only attribute as drift.
	if variation.Description != nil {
		data.Description = types.StringValue(*variation.Description)
	}
	if variation.Instructions != nil {
		data.Instructions = types.StringValue(*variation.Instructions)
	}
	if variation.ModelConfigKey != nil {
		data.ModelConfigKey = types.StringValue(*variation.ModelConfigKey)
	} else {
		data.ModelConfigKey = types.StringValue("")
	}
	if variation.State != nil {
		data.State = types.StringValue(*variation.State)
	} else {
		data.State = types.StringValue("")
	}

	// Model: strip empty defaults the API may inject so user state stays
	// stable. If the result is empty, set model to null.
	cleaned := stripEmptyMapValues(variation.Model)
	if len(cleaned) > 0 && !isEmptyModelMap(cleaned) {
		jsonStr, jerr := mapToJsonString(cleaned)
		if jerr != nil {
			diags.AddError(fmt.Sprintf("failed to serialize model for AI config variation %q", variationKey), jerr.Error())
			return
		}
		// Preserve prior state value if semantically equivalent (keeps
		// user's formatting/key ordering).
		if !data.Model.IsNull() && !data.Model.IsUnknown() && data.Model.ValueString() != "" {
			if jsonEqual(data.Model.ValueString(), jsonStr) {
				// keep existing value
			} else {
				data.Model = types.StringValue(jsonStr)
			}
		} else {
			data.Model = types.StringValue(jsonStr)
		}
	} else {
		data.Model = types.StringNull()
	}

	// Messages — emit null when the API returns no messages so plan
	// parity holds for Agent-mode variations (which never carry messages
	// but contain other sensitive-adjacent state).
	msgObj := types.ObjectType{AttrTypes: aiConfigVariationMessageAttrTypes}
	if len(variation.Messages) == 0 {
		data.Messages = types.ListNull(msgObj)
	} else {
		elems := make([]attr.Value, 0, len(variation.Messages))
		for _, m := range variation.Messages {
			obj, d := types.ObjectValue(aiConfigVariationMessageAttrTypes, map[string]attr.Value{
				ROLE:    types.StringValue(m.Role),
				CONTENT: types.StringValue(m.Content),
			})
			diags.Append(d...)
			elems = append(elems, obj)
		}
		list, d := types.ListValue(msgObj, elems)
		diags.Append(d...)
		data.Messages = list
	}

	// Tool keys: preserve prior value when the API returns empty.
	if len(variation.Tools) > 0 {
		tks := make([]string, len(variation.Tools))
		for i, t := range variation.Tools {
			tks[i] = t.Key
		}
		set, d := setFromStringSlice(ctx, tks)
		diags.Append(d...)
		data.ToolKeys = set
	} else if data.ToolKeys.IsNull() || data.ToolKeys.IsUnknown() {
		empty, _ := setFromStringSlice(ctx, []string{})
		data.ToolKeys = empty
	}

	judgesMap, d := variationJudgesToMap(variation.JudgeConfiguration)
	diags.Append(d...)
	data.Judges = judgesMap
}

// variationMessagesFromList converts the framework list of message
// objects into []ldapi.Message.
func variationMessagesFromList(ctx context.Context, list types.List) ([]ldapi.Message, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}
	type messageModel struct {
		Role    string `tfsdk:"role"`
		Content string `tfsdk:"content"`
	}
	var raw []messageModel
	diags.Append(list.ElementsAs(ctx, &raw, false)...)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]ldapi.Message, len(raw))
	for i, m := range raw {
		out[i] = *ldapi.NewMessage(m.Content, m.Role)
	}
	return out, diags
}

// variationJudgesFromMap converts the framework judges map (keyed by judge
// AI Config key) into []ldapi.JudgeAttachment, sorted by key for a stable
// request body.
func variationJudgesFromMap(ctx context.Context, m types.Map) ([]ldapi.JudgeAttachment, diag.Diagnostics) {
	var diags diag.Diagnostics
	if m.IsNull() || m.IsUnknown() {
		return nil, diags
	}
	type judgeModel struct {
		SamplingRate float64 `tfsdk:"sampling_rate"`
	}
	var raw map[string]judgeModel
	diags.Append(m.ElementsAs(ctx, &raw, false)...)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]ldapi.JudgeAttachment, 0, len(raw))
	for key, j := range raw {
		out = append(out, *ldapi.NewJudgeAttachment(key, float32(j.SamplingRate)))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].JudgeConfigKey < out[j].JudgeConfigKey })
	return out, diags
}

// variationJudgesToMap converts the API's judge configuration into the
// framework judges map. A nil configuration or empty judges list maps to
// null so the attribute round-trips when omitted from config.
func variationJudgesToMap(jc *ldapi.JudgeConfiguration) (types.Map, diag.Diagnostics) {
	var diags diag.Diagnostics
	objType := types.ObjectType{AttrTypes: aiConfigVariationJudgeAttrTypes}
	if jc == nil || len(jc.Judges) == 0 {
		return types.MapNull(objType), diags
	}
	els := make(map[string]attr.Value, len(jc.Judges))
	for _, j := range jc.Judges {
		obj, d := types.ObjectValue(aiConfigVariationJudgeAttrTypes, map[string]attr.Value{
			SAMPLING_RATE: types.Float64Value(float64ThroughFloat32(float64(j.SamplingRate))),
		})
		diags.Append(d...)
		els[j.JudgeConfigKey] = obj
	}
	m, d := types.MapValue(objType, els)
	diags.Append(d...)
	return m, diags
}

// float32PrecisionPlanModifier maps the planned value through a float32
// round-trip so the plan matches what the API — which stores sampling rates
// as 32-bit floats — returns on read.
type float32PrecisionPlanModifier struct{}

func (float32PrecisionPlanModifier) Description(context.Context) string {
	return "Normalizes the value to 32-bit float precision to match the API's storage."
}

func (m float32PrecisionPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (float32PrecisionPlanModifier) PlanModifyFloat64(_ context.Context, req planmodifier.Float64Request, resp *planmodifier.Float64Response) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	resp.PlanValue = types.Float64Value(float64ThroughFloat32(req.PlanValue.ValueFloat64()))
}

// float64ThroughFloat32 returns the float64 closest to the shortest decimal
// representation of v rounded to float32 — i.e. what a user-written literal
// looks like after an API float32 round-trip (0.1 stays 0.1 instead of
// becoming 0.10000000149011612).
func float64ThroughFloat32(v float64) float64 {
	out, err := strconv.ParseFloat(strconv.FormatFloat(v, 'f', -1, 32), 64)
	if err != nil {
		return v
	}
	return out
}

// jsonEqual returns true if two JSON strings parse to the same value.
func jsonEqual(a, b string) bool {
	if a == "" && b == "" {
		return true
	}
	if a == "" || b == "" {
		return false
	}
	var av, bv interface{}
	if json.Unmarshal([]byte(a), &av) != nil {
		return false
	}
	if json.Unmarshal([]byte(b), &bv) != nil {
		return false
	}
	return reflect.DeepEqual(av, bv)
}
