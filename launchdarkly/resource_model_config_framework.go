package launchdarkly

// resource_model_config_framework.go is the terraform-plugin-framework
// implementation of launchdarkly_model_config. All attributes are flat
// (no nested blocks). The API has no update operation; every attribute
// is ForceNew, surfaced here via stringplanmodifier.RequiresReplace().

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
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
	_ resource.Resource                = &ModelConfigResource{}
	_ resource.ResourceWithImportState = &ModelConfigResource{}
)

type ModelConfigResource struct {
	client *Client
}

type ModelConfigResourceModel struct {
	ID                 types.String  `tfsdk:"id"`
	ProjectKey         types.String  `tfsdk:"project_key"`
	Key                types.String  `tfsdk:"key"`
	Name               types.String  `tfsdk:"name"`
	ModelID            types.String  `tfsdk:"model_id"`
	Icon               types.String  `tfsdk:"icon"`
	ProviderName       types.String  `tfsdk:"model_provider"`
	Global             types.Bool    `tfsdk:"global"`
	Params             types.String  `tfsdk:"params"`
	CustomParameters   types.String  `tfsdk:"custom_parameters"`
	Tags               types.Set     `tfsdk:"tags"`
	Version            types.Int64   `tfsdk:"version"`
	CostPerInputToken  types.Float64 `tfsdk:"cost_per_input_token"`
	CostPerOutputToken types.Float64 `tfsdk:"cost_per_output_token"`
}

func NewModelConfigResource() resource.Resource {
	return &ModelConfigResource{}
}

func (r *ModelConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_model_config"
}

func (r *ModelConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly model config resource.\n\nThis resource allows you to create and manage AI model configurations within your LaunchDarkly project. Since the API does not support updates, any field change will force recreation of the resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID in the format `project_key/key`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:    true,
				Description: addForceNewDescription("The project key.", true),
				Validators:  []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			KEY: schema.StringAttribute{
				Required:    true,
				Description: addForceNewDescription("The model config's unique key.", true),
				Validators:  []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			NAME: schema.StringAttribute{
				Required:    true,
				Description: addForceNewDescription("The model config's human-readable name.", true),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			MODEL_ID: schema.StringAttribute{
				Required:    true,
				Description: addForceNewDescription("The model identifier (e.g. `gpt-4`, `claude-3`).", true),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			ICON: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: addForceNewDescription("The icon for the model config.", true),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			PROVIDER_NAME: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: addForceNewDescription("The provider name for the model config (e.g. `openai`, `anthropic`).", true),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			GLOBAL: schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the model config is available globally.",
			},
			PARAMS: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: addForceNewDescription("A JSON string representing the model parameters (e.g. `{\"temperature\": 0.7, \"maxTokens\": 4096}`).", true),
				Validators:  []validator.String{jsonStringValidator{}},
				PlanModifiers: []planmodifier.String{
					jsonNormalizePlanModifier{},
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			CUSTOM_PARAMETERS: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: addForceNewDescription("A JSON string representing custom parameters for the model config.", true),
				Validators:  []validator.String{jsonStringValidator{}},
				PlanModifiers: []planmodifier.String{
					jsonNormalizePlanModifier{},
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			TAGS: schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: addForceNewDescription("Tags associated with your resource.", true),
				Validators:  []validator.Set{setvalidator.ValueStringsAre(tagValidator())},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
					setplanmodifier.UseStateForUnknown(),
				},
			},
			VERSION: schema.Int64Attribute{
				Computed:    true,
				Description: "The version of the model config.",
			},
			COST_PER_INPUT_TOKEN: schema.Float64Attribute{
				Optional:    true,
				Computed:    true,
				Description: addForceNewDescription("The cost per input token for the model.", true),
			},
			COST_PER_OUTPUT_TOKEN: schema.Float64Attribute{
				Optional:    true,
				Computed:    true,
				Description: addForceNewDescription("The cost per output token for the model.", true),
			},
		},
	}
}

func (r *ModelConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *ModelConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ModelConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	key := plan.Key.ValueString()
	name := plan.Name.ValueString()
	modelID := plan.ModelID.ValueString()

	post := *ldapi.NewModelConfigPost(name, key, modelID)

	if !plan.Icon.IsNull() && !plan.Icon.IsUnknown() {
		icon := plan.Icon.ValueString()
		post.Icon = &icon
	}
	if !plan.ProviderName.IsNull() && !plan.ProviderName.IsUnknown() {
		provider := plan.ProviderName.ValueString()
		post.Provider = &provider
	}
	if !plan.Params.IsNull() && !plan.Params.IsUnknown() {
		params, err := jsonStringToMap(plan.Params.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid params JSON", err.Error())
			return
		}
		if params != nil {
			post.Params = params
		}
	}
	if !plan.CustomParameters.IsNull() && !plan.CustomParameters.IsUnknown() {
		customParams, err := jsonStringToMap(plan.CustomParameters.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid custom_parameters JSON", err.Error())
			return
		}
		if customParams != nil {
			post.CustomParams = customParams
		}
	}
	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		tags, diags := stringSliceFromSet(ctx, plan.Tags)
		resp.Diagnostics.Append(diags...)
		post.Tags = tags
	}
	if !plan.CostPerInputToken.IsNull() && !plan.CostPerInputToken.IsUnknown() {
		v := plan.CostPerInputToken.ValueFloat64()
		post.CostPerInputToken = &v
	}
	if !plan.CostPerOutputToken.IsNull() && !plan.CostPerOutputToken.IsUnknown() {
		v := plan.CostPerOutputToken.ValueFloat64()
		post.CostPerOutputToken = &v
	}

	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, _, err := r.client.ld.AIConfigsApi.PostModelConfig(r.client.ctx, projectKey).ModelConfigPost(post).Execute()
		return err
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, "Failed to create model config", err)
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/%s", projectKey, key))

	r.readIntoModel(ctx, projectKey, key, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ModelConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ModelConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	key := data.Key.ValueString()

	r.readIntoModel(ctx, projectKey, key, &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ModelConfigResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Unexpected Update call",
		"All attributes of launchdarkly_model_config are ForceNew; Update should never be invoked. This is a provider bug.",
	)
}

func (r *ModelConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ModelConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	key := data.Key.ValueString()

	var res *http.Response
	err := r.client.withConcurrency(r.client.ctx, func() error {
		var e error
		res, e = r.client.ld.AIConfigsApi.DeleteModelConfig(r.client.ctx, projectKey, key).Execute()
		return e
	})
	if err != nil {
		if isStatusNotFound(res) {
			return
		}
		errMsg := handleLdapiErr(err).Error()
		if strings.Contains(errMsg, "model config is still in use") {
			resp.Diagnostics.AddError(
				"Failed to delete model config",
				fmt.Sprintf("model config %q in project %q is still in use by one or more AI config variations. Use a Terraform resource reference for model_config_key (not a literal string) so Terraform can order destruction correctly.", key, projectKey),
			)
			return
		}
		addLdapiError(&resp.Diagnostics, "Failed to delete model config", err)
	}
}

func (r *ModelConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := req.ID
	if id == "" {
		resp.Diagnostics.AddError("Invalid import ID", "import ID cannot be empty")
		return
	}
	projectKey, key, err := modelConfigIdToKeys(id)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), projectKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(KEY), key)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// readIntoModel populates the supplied model with the model config's
// current LD-side state. Sets ID to null when the model config is
// missing so Read can drop the resource from state.
func (r *ModelConfigResource) readIntoModel(
	ctx context.Context,
	projectKey, key string,
	data *ModelConfigResourceModel,
	diags *diag.Diagnostics,
) {
	var modelConfig *ldapi.ModelConfig
	var res *http.Response
	var err error
	err = r.client.withConcurrency(r.client.ctx, func() error {
		modelConfig, res, err = r.client.ld.AIConfigsApi.GetModelConfig(r.client.ctx, projectKey, key).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError("Failed to get model config", handleLdapiErr(err).Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", projectKey, key))
	data.ProjectKey = types.StringValue(projectKey)
	data.Key = types.StringValue(modelConfig.Key)
	data.Name = types.StringValue(modelConfig.Name)
	data.ModelID = types.StringValue(modelConfig.Id)
	data.Global = types.BoolValue(modelConfig.Global)
	data.Version = types.Int64Value(int64(modelConfig.Version))
	if modelConfig.Icon != nil {
		data.Icon = types.StringValue(*modelConfig.Icon)
	} else {
		data.Icon = types.StringValue("")
	}
	if modelConfig.Provider != nil {
		data.ProviderName = types.StringValue(*modelConfig.Provider)
	} else {
		data.ProviderName = types.StringValue("")
	}

	paramsJSON, err := mapToJsonString(modelConfig.Params)
	if err != nil {
		diags.AddError("Failed to serialise params", err.Error())
		return
	}
	data.Params = types.StringValue(paramsJSON)

	customParamsJSON, err := mapToJsonString(modelConfig.CustomParams)
	if err != nil {
		diags.AddError("Failed to serialise custom_parameters", err.Error())
		return
	}
	data.CustomParameters = types.StringValue(customParamsJSON)

	tagsSet, d := setFromStringSlice(ctx, modelConfig.Tags)
	diags.Append(d...)
	data.Tags = tagsSet

	if modelConfig.CostPerInputToken != nil {
		data.CostPerInputToken = types.Float64Value(*modelConfig.CostPerInputToken)
	} else {
		data.CostPerInputToken = types.Float64Value(0)
	}
	if modelConfig.CostPerOutputToken != nil {
		data.CostPerOutputToken = types.Float64Value(*modelConfig.CostPerOutputToken)
	} else {
		data.CostPerOutputToken = types.Float64Value(0)
	}
}
