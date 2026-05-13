package launchdarkly

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"

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
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

var (
	_ resource.Resource                     = &AIConfigVariationResource{}
	_ resource.ResourceWithImportState      = &AIConfigVariationResource{}
	_ resource.ResourceWithConfigValidators = &AIConfigVariationResource{}
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
		Description: "Provides a LaunchDarkly AI Config variation resource.\n\nThis resource allows you to create and manage AI Config variations within your LaunchDarkly project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The project key. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			AI_CONFIG_KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The AI Config key that this variation belongs to. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The variation's unique key. A change in this field will force the destruction of the existing resource and the creation of a new one.",
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
				Description: "The variation's description (used in agent mode).",
			},
			INSTRUCTIONS: schema.StringAttribute{
				Optional:    true,
				Description: "The variation's instructions (used in agent mode).",
			},
			TOOL_KEYS: schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "A set of AI tool keys to associate with this variation. **Note:** The API does not currently return tool associations on read, so Terraform cannot detect drift for this field. Changes made outside of Terraform will not be reflected in state.",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
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
			},
			VERSION: schema.Int64Attribute{
				Computed:    true,
				Description: "The version number of the variation.",
			},
			CREATION_DATE: schema.Int64Attribute{
				Computed:    true,
				Description: "The creation timestamp of the variation.",
			},
		},
		Blocks: map[string]schema.Block{
			MESSAGES: schema.ListNestedBlock{
				Description: "A list of messages for completion mode. Each message has a `role` and `content`.",
				NestedObject: schema.NestedBlockObject{
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

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.AIConfigsApi.PostAIConfigVariation(r.client.ctx, projectKey, configKey).AIConfigVariationPost(*post).Execute()
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

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.AIConfigsApi.PatchAIConfigVariation(r.client.ctx, projectKey, configKey, variationKey).AIConfigVariationPatch(*patch).Execute()
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
		res, e = r.client.ld.AIConfigsApi.DeleteAIConfigVariation(r.client.ctx, projectKey, configKey, variationKey).Execute()
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
		variationsResp, res, err = r.client.ld.AIConfigsApi.GetAIConfigVariation(r.client.ctx, projectKey, configKey, variationKey).Execute()
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

	data.Description = stringValueOrNullFromPointer(variation.Description)
	data.Instructions = stringValueOrNullFromPointer(variation.Instructions)
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

	// Messages
	msgObj := types.ObjectType{AttrTypes: aiConfigVariationMessageAttrTypes}
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

	// Tool keys: SDKv2 preserved prior value when API returned empty.
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
