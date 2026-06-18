package launchdarkly

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                = &IntegrationDeliveryConfigurationResource{}
	_ resource.ResourceWithImportState = &IntegrationDeliveryConfigurationResource{}
)

type IntegrationDeliveryConfigurationResource struct {
	client *Client
	beta   *Client
}

type IntegrationDeliveryConfigurationResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectKey     types.String `tfsdk:"project_key"`
	EnvKey         types.String `tfsdk:"env_key"`
	IntegrationKey types.String `tfsdk:"integration_key"`
	ConfigID       types.String `tfsdk:"config_id"`
	Name           types.String `tfsdk:"name"`
	Config         types.String `tfsdk:"config"`
	On             types.Bool   `tfsdk:"on"`
	Tags           types.Set    `tfsdk:"tags"`
	Version        types.Int64  `tfsdk:"version"`
}

func NewIntegrationDeliveryConfigurationResource() resource.Resource {
	return &IntegrationDeliveryConfigurationResource{}
}

func (r *IntegrationDeliveryConfigurationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_delivery_configuration"
}

func (r *IntegrationDeliveryConfigurationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provides a LaunchDarkly integration delivery configuration resource.

~> **Beta:** This resource wraps a beta LaunchDarkly API. Beta resources may change or be removed in future versions, and the provider sends the ` + "`LD-API-Version: beta`" + ` header on every request to this endpoint.

Integration delivery configurations connect a LaunchDarkly project environment to a persistent feature store integration (for example a Relay Proxy feature store such as ` + "`redis`" + ` or ` + "`dynamodb`" + `), so flag and segment data is delivered to that destination. A configuration is scoped to a single project environment and integration. To learn more, read [Persistent store integrations](https://docs.launchdarkly.com/home/relay-proxy/persistent-store-integrations).`,
		Attributes: integrationDeliveryConfigurationSchemaAttributes(),
	}
}

func integrationDeliveryConfigurationSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:      true,
			Description:   "The ID of this resource in the format `project_key/env_key/integration_key/config_id`.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		PROJECT_KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The project key. The integration delivery configuration is scoped to this project.", true),
			Validators:    []validator.String{keyValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		ENV_KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The environment key. The integration delivery configuration is scoped to this environment.", true),
			Validators:    []validator.String{keyValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		INTEGRATION_KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The integration key identifying the persistent feature store integration this configuration delivers to (for example `redis` or `dynamodb`).", true),
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		CONFIG_ID: schema.StringAttribute{
			Computed:      true,
			Description:   "The unique server-assigned ID of the delivery configuration.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		NAME: schema.StringAttribute{
			Optional:      true,
			Computed:      true,
			Description:   "A human-friendly name for the delivery configuration. If not set, LaunchDarkly assigns one based on the integration.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		CONFIG: schema.StringAttribute{
			Required:    true,
			Description: "A JSON string representing the integration-specific configuration. The accepted fields are defined by the integration's manifest (for example connection and authentication details). Secret fields may be returned obfuscated by the API.",
			Validators:  []validator.String{jsonStringValidator{}},
			PlanModifiers: []planmodifier.String{
				jsonNormalizePlanModifier{},
			},
		},
		ON: schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
			Description: "Whether the delivery configuration is turned on. Defaults to `false`.",
		},
		TAGS: schema.SetAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: "Tags associated with the delivery configuration.",
			Validators: []validator.Set{
				setvalidator.ValueStringsAre(tagValidator()),
			},
		},
		VERSION: schema.Int64Attribute{
			Computed:    true,
			Description: "The version of the delivery configuration.",
		},
	}
}

func (r *IntegrationDeliveryConfigurationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
	if r.client == nil {
		return
	}
	beta, err := newIntegrationDeliveryConfigurationBetaClient(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build LaunchDarkly beta client", err.Error())
		return
	}
	r.beta = beta
}

func (r *IntegrationDeliveryConfigurationResource) betaClient() (*Client, error) {
	if r.beta != nil {
		return r.beta, nil
	}
	return newIntegrationDeliveryConfigurationBetaClient(r.client)
}

func (r *IntegrationDeliveryConfigurationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan IntegrationDeliveryConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	beta, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}

	configMap, err := jsonStringToMap(plan.Config.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid config", err.Error())
		return
	}
	if configMap == nil {
		configMap = map[string]interface{}{}
	}

	post := ldapi.NewIntegrationDeliveryConfigurationPost(configMap)
	if !plan.On.IsNull() && !plan.On.IsUnknown() {
		post.On = ldapi.PtrBool(plan.On.ValueBool())
	}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() && plan.Name.ValueString() != "" {
		post.Name = ldapi.PtrString(plan.Name.ValueString())
	}
	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		tags, d := stringSliceFromSet(ctx, plan.Tags)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		post.Tags = tags
	}

	projectKey := plan.ProjectKey.ValueString()
	envKey := plan.EnvKey.ValueString()
	integrationKey := plan.IntegrationKey.ValueString()

	var created *ldapi.IntegrationDeliveryConfiguration
	err = beta.withConcurrency(beta.ctx, func() error {
		var e error
		created, _, e = beta.ld.IntegrationDeliveryConfigurationsBetaApi.
			CreateIntegrationDeliveryConfiguration(beta.ctx, projectKey, envKey, integrationKey).
			IntegrationDeliveryConfigurationPost(*post).
			Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error creating integration delivery configuration for integration %q in project %q environment %q", integrationKey, projectKey, envKey), err)
		return
	}
	if created == nil || created.GetId() == "" {
		resp.Diagnostics.AddError("Missing configuration ID", "LaunchDarkly returned a delivery configuration without an ID")
		return
	}

	configID := created.GetId()
	plan.ConfigID = types.StringValue(configID)
	plan.ID = types.StringValue(integrationDeliveryConfigurationID(projectKey, envKey, integrationKey, configID))
	r.readIntoModel(ctx, projectKey, envKey, integrationKey, configID, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IntegrationDeliveryConfigurationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data IntegrationDeliveryConfigurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey, envKey, integrationKey, configID, err := integrationDeliveryConfigurationIDToKeys(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid integration delivery configuration ID", err.Error())
		return
	}

	r.readIntoModel(ctx, projectKey, envKey, integrationKey, configID, &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IntegrationDeliveryConfigurationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state IntegrationDeliveryConfigurationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	beta, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}

	projectKey, envKey, integrationKey, configID, err := integrationDeliveryConfigurationIDToKeys(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid integration delivery configuration ID", err.Error())
		return
	}

	var patch []ldapi.PatchOperation
	if !plan.Name.Equal(state.Name) && !plan.Name.IsUnknown() {
		patch = append(patch, patchReplace("/name", plan.Name.ValueString()))
	}
	if !plan.On.Equal(state.On) {
		patch = append(patch, patchReplace("/on", plan.On.ValueBool()))
	}
	if !plan.Config.Equal(state.Config) {
		configMap, err := jsonStringToMap(plan.Config.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid config", err.Error())
			return
		}
		if configMap == nil {
			configMap = map[string]interface{}{}
		}
		patch = append(patch, patchReplace("/config", configMap))
	}
	if !plan.Tags.Equal(state.Tags) {
		tags, d := stringSliceFromSet(ctx, plan.Tags)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		if tags == nil {
			tags = []string{}
		}
		patch = append(patch, patchReplace("/tags", tags))
	}

	if len(patch) > 0 {
		err = beta.withConcurrency(beta.ctx, func() error {
			_, _, e := beta.ld.IntegrationDeliveryConfigurationsBetaApi.
				PatchIntegrationDeliveryConfiguration(beta.ctx, projectKey, envKey, integrationKey, configID).
				PatchOperation(patch).
				Execute()
			return e
		})
		if err != nil {
			addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error updating integration delivery configuration %q in project %q environment %q", configID, projectKey, envKey), err)
			return
		}
	}

	r.readIntoModel(ctx, projectKey, envKey, integrationKey, configID, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IntegrationDeliveryConfigurationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data IntegrationDeliveryConfigurationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	beta, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}

	projectKey, envKey, integrationKey, configID, err := integrationDeliveryConfigurationIDToKeys(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid integration delivery configuration ID", err.Error())
		return
	}

	var res *http.Response
	err = beta.withConcurrency(beta.ctx, func() error {
		var e error
		res, e = beta.ld.IntegrationDeliveryConfigurationsBetaApi.
			DeleteIntegrationDeliveryConfiguration(beta.ctx, projectKey, envKey, integrationKey, configID).
			Execute()
		return e
	})
	if err != nil {
		if isStatusNotFound(res) {
			return
		}
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error deleting integration delivery configuration %q in project %q environment %q", configID, projectKey, envKey), err)
	}
}

// ImportState expects "project_key/env_key/integration_key/config_id".
func (r *IntegrationDeliveryConfigurationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectKey, envKey, integrationKey, configID, err := integrationDeliveryConfigurationIDToKeys(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), projectKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(ENV_KEY), envKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(INTEGRATION_KEY), integrationKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(CONFIG_ID), configID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *IntegrationDeliveryConfigurationResource) readIntoModel(
	ctx context.Context,
	projectKey, envKey, integrationKey, configID string,
	data *IntegrationDeliveryConfigurationResourceModel,
	diags *diag.Diagnostics,
) {
	beta, err := r.betaClient()
	if err != nil {
		diags.AddError("Failed to build beta client", err.Error())
		return
	}

	var cfg *ldapi.IntegrationDeliveryConfiguration
	var res *http.Response
	err = beta.withConcurrency(beta.ctx, func() error {
		cfg, res, err = beta.ld.IntegrationDeliveryConfigurationsBetaApi.
			GetIntegrationDeliveryConfigurationById(beta.ctx, projectKey, envKey, integrationKey, configID).
			Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError(fmt.Sprintf("Failed to get integration delivery configuration %q", configID), handleLdapiErr(err).Error())
		return
	}

	data.ID = types.StringValue(integrationDeliveryConfigurationID(projectKey, envKey, integrationKey, cfg.GetId()))
	data.ProjectKey = types.StringValue(cfg.GetProjectKey())
	data.EnvKey = types.StringValue(cfg.GetEnvironmentKey())
	data.IntegrationKey = types.StringValue(cfg.GetIntegrationKey())
	data.ConfigID = types.StringValue(cfg.GetId())
	data.Name = types.StringValue(cfg.GetName())
	data.On = types.BoolValue(cfg.GetOn())
	data.Version = types.Int64Value(int64(cfg.GetVersion()))

	configJSON, err := mapToJsonString(cfg.GetConfig())
	if err != nil {
		diags.AddError("Failed to serialise config", err.Error())
		return
	}
	data.Config = types.StringValue(configJSON)

	// Optional-only Set attr: preserve the config's null-vs-empty intent so an
	// omitted `tags` reads back as null, not an empty set.
	tagsSet, d := setFromStringSlicePreservingPlan(ctx, cfg.GetTags(), data.Tags)
	diags.Append(d...)
	data.Tags = tagsSet
}
