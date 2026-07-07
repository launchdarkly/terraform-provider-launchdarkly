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
	ldapi "github.com/launchdarkly/api-client-go/v23"
)

var (
	_ resource.Resource                = &BigSegmentStoreIntegrationResource{}
	_ resource.ResourceWithImportState = &BigSegmentStoreIntegrationResource{}
)

type BigSegmentStoreIntegrationResource struct {
	client *Client
	beta   *Client
}

type BigSegmentStoreIntegrationResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProjectKey     types.String `tfsdk:"project_key"`
	EnvironmentKey types.String `tfsdk:"environment_key"`
	IntegrationKey types.String `tfsdk:"integration_key"`
	IntegrationID  types.String `tfsdk:"integration_id"`
	Name           types.String `tfsdk:"name"`
	On             types.Bool   `tfsdk:"on"`
	Config         types.String `tfsdk:"config"`
	Tags           types.Set    `tfsdk:"tags"`
	Version        types.Int64  `tfsdk:"version"`
}

func NewBigSegmentStoreIntegrationResource() resource.Resource {
	return &BigSegmentStoreIntegrationResource{}
}

func (r *BigSegmentStoreIntegrationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_big_segment_store_integration"
}

func (r *BigSegmentStoreIntegrationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provides a LaunchDarkly big segment store (persistent store) integration resource.

~> **Beta:** This resource wraps a beta LaunchDarkly API. Beta resources may change or be removed in future provider versions.

This resource lets you create and manage a persistent store integration for an environment. Server-side SDKs use a persistent store, backed by Redis or DynamoDB in your own infrastructure, to evaluate segments synced from external tools and larger list-based segments. To learn more, read [Segment configuration](https://launchdarkly.com/docs/home/flags/segment-config).`,
		Attributes: bigSegmentStoreIntegrationSchemaAttributes(),
	}
}

func bigSegmentStoreIntegrationSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:      true,
			Description:   "The ID of this resource in the format `project_key/environment_key/integration_key/integration_id`.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		PROJECT_KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The key of the project the integration belongs to.", true),
			Validators:    []validator.String{keyValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		ENVIRONMENT_KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The key of the environment the integration belongs to. Persistent store integrations are environment-scoped.", true),
			Validators:    []validator.String{keyValidator()},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		INTEGRATION_KEY: schema.StringAttribute{
			Required:      true,
			Description:   addForceNewDescription("The persistent store technology to use. Must be one of `redis` or `dynamodb`.", true),
			Validators:    []validator.String{oneOfValidator{allowed: []string{BIG_SEGMENT_STORE_INTEGRATION_KEY_REDIS, BIG_SEGMENT_STORE_INTEGRATION_KEY_DYNAMODB}}},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		INTEGRATION_ID: schema.StringAttribute{
			Computed:      true,
			Description:   "The server-assigned ID of the integration configuration.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		NAME: schema.StringAttribute{
			Optional:    true,
			Description: "A human-friendly name for the integration configuration.",
		},
		ON: schema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(false),
			Description: "Whether the integration is turned on. Defaults to `false`.",
		},
		CONFIG: schema.StringAttribute{
			Required:    true,
			Sensitive:   true,
			Description: "A JSON string holding the store-specific configuration. All values are strings except `tlsEnabled`, which is a boolean. For `redis`, the fields are `host`, `port`, `tlsEnabled`, `username`, and `password`. For `dynamodb`, they are `tableName`, `region`, `roleArn`, and `externalId`, a UUID. Marked sensitive because it carries credentials. The API redacts secrets and normalizes this value on read, so it is treated as write-only: Terraform stores and diffs the value you provide and does not reconcile it against the server, so configuration changes made outside Terraform are not detected as drift.",
			Validators:  []validator.String{jsonStringValidator{}},
			PlanModifiers: []planmodifier.String{
				jsonNormalizePlanModifier{},
			},
		},
		TAGS: schema.SetAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: "Tags associated with the integration configuration.",
			Validators: []validator.Set{
				setvalidator.ValueStringsAre(tagValidator()),
			},
		},
		VERSION: schema.Int64Attribute{
			Computed:    true,
			Description: "The version of the integration configuration.",
		},
	}
}

func (r *BigSegmentStoreIntegrationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
	if r.client == nil {
		return
	}
	beta, err := newBigSegmentStoreIntegrationBetaClient(r.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build LaunchDarkly beta client", err.Error())
		return
	}
	r.beta = beta
}

func (r *BigSegmentStoreIntegrationResource) betaClient() (*Client, error) {
	if r.beta != nil {
		return r.beta, nil
	}
	return newBigSegmentStoreIntegrationBetaClient(r.client)
}

func (r *BigSegmentStoreIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BigSegmentStoreIntegrationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := plan.ProjectKey.ValueString()
	if exists, err := projectExists(projectKey, r.client); !exists {
		if err != nil {
			resp.Diagnostics.AddError("Failed to check project", err.Error())
			return
		}
		resp.Diagnostics.AddError("Project not found", fmt.Sprintf("cannot find project with key %q", projectKey))
		return
	}

	beta, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}

	environmentKey := plan.EnvironmentKey.ValueString()
	integrationKey := plan.IntegrationKey.ValueString()

	config, err := jsonStringToMap(plan.Config.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid config", err.Error())
		return
	}

	tags, d := stringSliceFromSet(ctx, plan.Tags)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	post := ldapi.IntegrationDeliveryConfigurationPost{
		Config: config,
		On:     plan.On.ValueBoolPointer(),
		Tags:   tags,
	}
	if name := plan.Name.ValueString(); name != "" {
		post.Name = &name
	}

	var created *ldapi.BigSegmentStoreIntegration
	err = beta.withConcurrency(beta.ctx, func() error {
		var e error
		created, _, e = beta.ld.PersistentStoreIntegrationsBetaApi.CreateBigSegmentStoreIntegration(beta.ctx, projectKey, environmentKey, integrationKey).IntegrationDeliveryConfigurationPost(post).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error creating big segment store integration in %q/%q (%s)", projectKey, environmentKey, integrationKey), err)
		return
	}

	integrationID := created.GetId()
	plan.IntegrationID = types.StringValue(integrationID)
	plan.ID = types.StringValue(bigSegmentStoreIntegrationID(projectKey, environmentKey, integrationKey, integrationID))
	r.readIntoModel(ctx, projectKey, environmentKey, integrationKey, integrationID, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BigSegmentStoreIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BigSegmentStoreIntegrationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readIntoModel(
		ctx,
		data.ProjectKey.ValueString(),
		data.EnvironmentKey.ValueString(),
		data.IntegrationKey.ValueString(),
		data.IntegrationID.ValueString(),
		&data,
		&resp.Diagnostics,
	)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BigSegmentStoreIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state BigSegmentStoreIntegrationResourceModel
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

	projectKey := plan.ProjectKey.ValueString()
	environmentKey := plan.EnvironmentKey.ValueString()
	integrationKey := plan.IntegrationKey.ValueString()
	integrationID := state.IntegrationID.ValueString()

	var patch []ldapi.PatchOperation
	if !plan.Name.Equal(state.Name) {
		patch = append(patch, patchReplace("/name", plan.Name.ValueString()))
	}
	if !plan.On.Equal(state.On) {
		patch = append(patch, patchReplace("/on", plan.On.ValueBool()))
	}
	if !plan.Config.Equal(state.Config) {
		config, err := jsonStringToMap(plan.Config.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid config", err.Error())
			return
		}
		patch = append(patch, patchReplace("/config", config))
	}
	if !plan.Tags.Equal(state.Tags) {
		tags, d := stringSliceFromSet(ctx, plan.Tags)
		resp.Diagnostics.Append(d...)
		if tags == nil {
			tags = []string{}
		}
		patch = append(patch, patchReplace("/tags", tags))
	}

	if resp.Diagnostics.HasError() {
		return
	}

	if len(patch) > 0 {
		err = beta.withConcurrency(beta.ctx, func() error {
			_, _, e := beta.ld.PersistentStoreIntegrationsBetaApi.PatchBigSegmentStoreIntegration(beta.ctx, projectKey, environmentKey, integrationKey, integrationID).PatchOperation(patch).Execute()
			return e
		})
		if err != nil {
			addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error updating big segment store integration %q in %q/%q", integrationID, projectKey, environmentKey), err)
			return
		}
	}

	r.readIntoModel(ctx, projectKey, environmentKey, integrationKey, integrationID, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BigSegmentStoreIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BigSegmentStoreIntegrationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	beta, err := r.betaClient()
	if err != nil {
		resp.Diagnostics.AddError("Failed to build beta client", err.Error())
		return
	}
	var res *http.Response
	err = beta.withConcurrency(beta.ctx, func() error {
		var e error
		res, e = beta.ld.PersistentStoreIntegrationsBetaApi.DeleteBigSegmentStoreIntegration(
			beta.ctx,
			data.ProjectKey.ValueString(),
			data.EnvironmentKey.ValueString(),
			data.IntegrationKey.ValueString(),
			data.IntegrationID.ValueString(),
		).Execute()
		return e
	})
	if err != nil {
		if isStatusNotFound(res) {
			return
		}
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("Error deleting big segment store integration %q", data.IntegrationID.ValueString()), err)
	}
}

func (r *BigSegmentStoreIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectKey, environmentKey, integrationKey, integrationID, err := bigSegmentStoreIntegrationIDToKeys(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), projectKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(ENVIRONMENT_KEY), environmentKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(INTEGRATION_KEY), integrationKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(INTEGRATION_ID), integrationID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *BigSegmentStoreIntegrationResource) readIntoModel(
	ctx context.Context,
	projectKey, environmentKey, integrationKey, integrationID string,
	data *BigSegmentStoreIntegrationResourceModel,
	diags *diag.Diagnostics,
) {
	beta, err := r.betaClient()
	if err != nil {
		diags.AddError("Failed to build beta client", err.Error())
		return
	}

	var integration *ldapi.BigSegmentStoreIntegration
	var res *http.Response
	err = beta.withConcurrency(beta.ctx, func() error {
		integration, res, err = beta.ld.PersistentStoreIntegrationsBetaApi.GetBigSegmentStoreIntegration(beta.ctx, projectKey, environmentKey, integrationKey, integrationID).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError(fmt.Sprintf("Failed to get big segment store integration %q", integrationID), handleLdapiErr(err).Error())
		return
	}

	data.ID = types.StringValue(bigSegmentStoreIntegrationID(projectKey, environmentKey, integrationKey, integration.GetId()))
	data.ProjectKey = types.StringValue(integration.GetProjectKey())
	data.EnvironmentKey = types.StringValue(integration.GetEnvironmentKey())
	data.IntegrationKey = types.StringValue(integration.GetIntegrationKey())
	data.IntegrationID = types.StringValue(integration.GetId())
	data.On = types.BoolValue(integration.GetOn())
	data.Version = types.Int64Value(int64(integration.GetVersion()))

	// Optional-only attr: null-when-empty for plan-apply consistency.
	data.Name = stringValueOrNull(integration.GetName())

	// config is intentionally NOT read back from the API. Confirmed against a
	// live environment: the API redacts secret values (e.g. the Redis
	// `password` is returned as `sup********123`) and normalizes keys/types
	// (e.g. the input `port` number is stored as a string, and connection-TLS
	// is keyed `tlsEnabled`). Overwriting data.Config with the API response
	// therefore produces "inconsistent values for sensitive attribute" on
	// apply and a perpetual diff thereafter. We preserve the user-supplied
	// value already present in `data` (the plan on create/update, the prior
	// state on read) instead — mirroring how the destination resource keeps
	// obfuscated secrets via preserveObfuscatedDestinationAttributes. The
	// trade-off is no server-side drift detection for `config`, which is
	// acceptable for a write-mostly, secret-bearing attribute.

	// Optional-only Set attr: preserve the config's null-vs-empty intent so an
	// omitted `tags` reads back as null, not an empty set.
	tagsSet, d := setFromStringSlicePreservingPlan(ctx, integration.Tags, data.Tags)
	diags.Append(d...)
	data.Tags = tagsSet
}
