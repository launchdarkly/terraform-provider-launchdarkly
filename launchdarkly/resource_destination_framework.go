package launchdarkly

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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

var (
	_ resource.Resource                = &DestinationResource{}
	_ resource.ResourceWithImportState = &DestinationResource{}
)

type DestinationResource struct {
	client *Client
}

type DestinationResourceModel struct {
	ID         types.String `tfsdk:"id"`
	ProjectKey types.String `tfsdk:"project_key"`
	EnvKey     types.String `tfsdk:"env_key"`
	Name       types.String `tfsdk:"name"`
	Kind       types.String `tfsdk:"kind"`
	Config     types.Map    `tfsdk:"config"`
	On         types.Bool   `tfsdk:"on"`
	Tags       types.Set    `tfsdk:"tags"`
}

func NewDestinationResource() resource.Resource {
	return &DestinationResource{}
}

func (r *DestinationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_destination"
}

func (r *DestinationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provides a LaunchDarkly Data Export Destination resource.

-> **Note:** Data Export is available to customers on an Enterprise LaunchDarkly plan. To learn more, [read about our pricing](https://launchdarkly.com/pricing/). To upgrade your plan, [contact LaunchDarkly Sales](https://launchdarkly.com/contact-sales/).

Data Export Destinations are locations that receive exported data. This resource allows you to configure destinations for the export of raw analytics data, including feature flag requests, analytics events, custom events, and more.

To learn more about data export, read [Data Export Documentation](https://docs.launchdarkly.com/integrations/data-export).`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			PROJECT_KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The LaunchDarkly project key. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				Validators:    []validator.String{keyValidator()},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			ENV_KEY: schema.StringAttribute{
				Required:      true,
				Description:   "The environment key. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			NAME: schema.StringAttribute{
				Required:    true,
				Description: "A human-readable name for your data export destination.",
			},
			KIND: schema.StringAttribute{
				Required:      true,
				Description:   "The data export destination type. Available choices are `kinesis`, `google-pubsub`, `mparticle`, `azure-event-hubs`, and `segment`. A change in this field will force the destruction of the existing resource and the creation of a new one.",
				Validators:    []validator.String{oneOfValidator{allowed: []string{"kinesis", "google-pubsub", "mparticle", "azure-event-hubs", "segment"}}},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			CONFIG: schema.MapAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "The destination-specific configuration. To learn more, read [Destination-Specific Configs](#destination-specific-configs)",
			},
			ON: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the data export destination is on or not.",
			},
			TAGS: schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Tags associated with this resource.",
			},
		},
	}
}

func (r *DestinationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *DestinationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DestinationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rawConfig, d := mapStringFromAttr(ctx, plan.Config)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	kind := plan.Kind.ValueString()
	apiConfig, err := destinationConfigMapToAPI(kind, rawConfig)
	if err != nil {
		resp.Diagnostics.AddError("Invalid destination config", err.Error())
		return
	}

	name := plan.Name.ValueString()
	on := plan.On.ValueBool()
	post := ldapi.DestinationPost{
		Name:   &name,
		Kind:   &kind,
		Config: &apiConfig,
		On:     &on,
	}

	projectKey := plan.ProjectKey.ValueString()
	envKey := plan.EnvKey.ValueString()

	var dest *ldapi.Destination
	err = r.client.withConcurrency(r.client.ctx, func() error {
		var e error
		dest, _, e = r.client.ld.DataExportDestinationsApi.PostDestination(r.client.ctx, projectKey, envKey).DestinationPost(post).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("failed to create destination with project key %q and env key %q", projectKey, envKey), err)
		return
	}
	if dest.Id == nil {
		resp.Diagnostics.AddError("Missing destination ID", "LaunchDarkly returned a destination without an ID")
		return
	}

	plan.ID = types.StringValue(strings.Join([]string{projectKey, envKey, *dest.Id}, "/"))
	r.readIntoModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DestinationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DestinationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readIntoModel(ctx, &data, &resp.Diagnostics)
	if data.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DestinationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DestinationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rawConfig, d := mapStringFromAttr(ctx, plan.Config)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	kind := plan.Kind.ValueString()
	apiConfig, err := destinationConfigMapToAPI(kind, rawConfig)
	if err != nil {
		resp.Diagnostics.AddError("Invalid destination config", err.Error())
		return
	}

	_, _, destID, err := destinationImportIDtoKeys(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid destination ID", err.Error())
		return
	}
	projectKey := plan.ProjectKey.ValueString()
	envKey := plan.EnvKey.ValueString()
	name := plan.Name.ValueString()
	on := plan.On.ValueBool()

	patch := []ldapi.PatchOperation{
		patchReplace("/name", &name),
		patchReplace("/kind", &kind),
		patchReplace("/on", &on),
		patchReplace("/config", &apiConfig),
	}

	err = r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.DataExportDestinationsApi.PatchDestination(r.client.ctx, projectKey, envKey, destID).PatchOperation(patch).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("failed to update destination with id %q", destID), err)
		return
	}

	r.readIntoModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DestinationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DestinationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, _, destID, err := destinationImportIDtoKeys(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid destination ID", err.Error())
		return
	}
	err = r.client.withConcurrency(r.client.ctx, func() error {
		_, e := r.client.ld.DataExportDestinationsApi.DeleteDestination(r.client.ctx, data.ProjectKey.ValueString(), data.EnvKey.ValueString(), destID).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("failed to delete destination with id %q", destID), err)
	}
}

// ImportState expects "projectKey/envKey/destinationID".
func (r *DestinationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projKey, envKey, _, err := destinationImportIDtoKeys(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(PROJECT_KEY), projKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(ENV_KEY), envKey)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *DestinationResource) readIntoModel(ctx context.Context, data *DestinationResourceModel, diags *diag.Diagnostics) {
	projectKey := data.ProjectKey.ValueString()
	envKey := data.EnvKey.ValueString()
	_, _, destID, err := destinationImportIDtoKeys(data.ID.ValueString())
	if err != nil {
		diags.AddError("Invalid destination ID", err.Error())
		return
	}

	var dest *ldapi.Destination
	var res *http.Response
	err = r.client.withConcurrency(r.client.ctx, func() error {
		dest, res, err = r.client.ld.DataExportDestinationsApi.GetDestination(r.client.ctx, projectKey, envKey, destID).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError(fmt.Sprintf("failed to get destination with id %q", destID), handleLdapiErr(err).Error())
		return
	}
	if dest.Id == nil || dest.Kind == nil {
		diags.AddError("Malformed destination response", "destination response missing id or kind")
		return
	}

	priorConfig, _ := mapStringFromAttr(ctx, data.Config)
	apiCfg := destinationConfigFromAPI(*dest.Kind, dest.Config)
	preserved := preserveObfuscatedDestinationAttributes(priorConfig, apiCfg)

	// mparticle compat: when user supplied user_identities, the server
	// may also return the legacy user_identity scalar. Drop it from
	// state so it doesn't surface as drift. Mirrors the SDKv2
	// configDiffSuppressFunc on user_identity.
	if *dest.Kind == "mparticle" {
		if _, hasNew := priorConfig["user_identities"]; hasNew {
			delete(preserved, "user_identity")
		}
	}

	cfgVal, d := types.MapValueFrom(ctx, types.StringType, preserved)
	diags.Append(d...)
	data.Config = cfgVal

	if dest.Name != nil {
		data.Name = types.StringValue(*dest.Name)
	}
	data.Kind = types.StringValue(*dest.Kind)
	if dest.On != nil {
		data.On = types.BoolValue(*dest.On)
	} else {
		data.On = types.BoolValue(false)
	}
	data.ID = types.StringValue(strings.Join([]string{projectKey, envKey, *dest.Id}, "/"))

	// SDKv2 source never set TAGS on read; Destinations API doesn't expose
	// tags on the response object in this client. Preserve incoming tags
	// to keep state stable across plans.
	if data.Tags.IsNull() || data.Tags.IsUnknown() {
		empty, _ := setFromStringSlice(ctx, []string{})
		data.Tags = empty
	}
}

// destinationConfigMapToAPI is the framework analogue of
// destinationConfigFromResourceData. Translates snake_case keys in the
// user's terraform config into the camelCase keys the LD API expects,
// validating that every required field is present and that the
// mparticle user_identities JSON parses into a well-formed slice.
func destinationConfigMapToAPI(kind string, userConfig map[string]string) (map[string]interface{}, error) {
	attrs, ok := CONFIG_CONVERSIONS[kind]
	if !ok {
		return nil, fmt.Errorf("%q is not one of the supported destination kinds", kind)
	}
	out := make(map[string]interface{}, len(attrs.required)+len(attrs.optional))
	for k, v := range attrs.required {
		raw, present := userConfig[k]
		if !present {
			return nil, fmt.Errorf("missing required config field %q for destination kind %q", k, kind)
		}
		apiKey := v.(string)
		out[apiKey] = raw
	}
	for k, v := range attrs.optional {
		raw, present := userConfig[k]
		if !present {
			continue
		}
		apiKey := v.(string)
		if kind == "mparticle" && k == "user_identities" {
			var identities []map[string]interface{}
			if err := json.Unmarshal([]byte(raw), &identities); err != nil {
				return nil, fmt.Errorf("config field %q for destination kind %q is not valid: %s", k, kind, err.Error())
			}
			if err := validateMParticleUserIdentities(identities); err != nil {
				return nil, fmt.Errorf("badly-formed mParticle user_identities field: %s", err.Error())
			}
			out[apiKey] = identities
			continue
		}
		out[apiKey] = raw
	}
	return out, nil
}

// destinationConfigFromAPI mirrors destinationConfigToResourceData but
// produces map[string]string (the framework Map<String> shape).
func destinationConfigFromAPI(kind string, apiConfig interface{}) map[string]string {
	coerced, ok := apiConfig.(map[string]interface{})
	if !ok {
		return map[string]string{}
	}
	out := make(map[string]string, len(coerced))
	for k, v := range CONFIG_CONVERSIONS[kind].allParameters() {
		apiKey := v.(string)
		raw, present := coerced[apiKey]
		if !present || raw == nil {
			continue
		}
		if kind == "mparticle" && k == "user_identities" {
			b, err := json.Marshal(raw)
			if err != nil {
				continue
			}
			out[k] = string(b)
			continue
		}
		out[k] = formatConfigValue(raw)
	}
	return out
}

// preserveObfuscatedDestinationAttributes keeps user-supplied secrets in
// state when the API returns obfuscated stand-ins. Mirrors
// preserveObfuscatedConfigAttributes from destination_helper.go.
func preserveObfuscatedDestinationAttributes(priorState, apiCfg map[string]string) map[string]string {
	obfuscated := []string{"api_key", "secret", "write_key", "policy_key"}
	for _, k := range obfuscated {
		if _, server := apiCfg[k]; server {
			if user, ok := priorState[k]; ok {
				apiCfg[k] = user
			}
		}
	}
	return apiCfg
}

// destinationImportIDtoKeys splits "projectKey/envKey/destinationID".
func destinationImportIDtoKeys(importID string) (projKey, envKey, destinationID string, err error) {
	if strings.Count(importID, "/") != 2 {
		return "", "", "", fmt.Errorf("found unexpected destination import id format: %q expected format: 'project_key/env_key/destination_id'", importID)
	}
	parts := strings.SplitN(importID, "/", 3)
	return parts[0], parts[1], parts[2], nil
}

func formatConfigValue(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case bool:
		return strconv.FormatBool(x)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", x)
	}
}
