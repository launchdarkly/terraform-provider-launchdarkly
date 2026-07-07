package launchdarkly

import (
	"context"
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
	ldapi "github.com/launchdarkly/api-client-go/v23"
	strcase "github.com/stoewer/go-strcase"
)

var (
	_ resource.Resource                = &AuditLogSubscriptionResource{}
	_ resource.ResourceWithImportState = &AuditLogSubscriptionResource{}
)

type AuditLogSubscriptionResource struct {
	client *Client
}

type AuditLogSubscriptionResourceModel struct {
	ID             types.String `tfsdk:"id"`
	IntegrationKey types.String `tfsdk:"integration_key"`
	Name           types.String `tfsdk:"name"`
	Config         types.Map    `tfsdk:"config"`
	Statements     types.List   `tfsdk:"statements"`
	On             types.Bool   `tfsdk:"on"`
	Tags           types.Set    `tfsdk:"tags"`
}

func NewAuditLogSubscriptionResource() resource.Resource {
	return &AuditLogSubscriptionResource{}
}

func (r *AuditLogSubscriptionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_audit_log_subscription"
}

func (r *AuditLogSubscriptionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a LaunchDarkly audit log subscription resource.\n\nThis resource allows you to create and manage LaunchDarkly audit log subscriptions.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			INTEGRATION_KEY: schema.StringAttribute{
				Required:      true,
				Description:   fmt.Sprintf("The integration key. Supported integration keys are %s. A change in this field will force the destruction of the existing resource and the creation of a new one.", oxfordCommaJoin(getValidIntegrationKeys())),
				Validators:    []validator.String{oneOfValidator{allowed: getValidIntegrationKeys()}},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			NAME: schema.StringAttribute{
				Required:    true,
				Description: "A human-friendly name for your audit log subscription viewable from within the LaunchDarkly Integrations page.",
			},
			CONFIG: schema.MapAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "The set of configuration fields corresponding to the value defined for `integration_key`. Refer to the `formVariables` field in the corresponding `integrations/<integration_key>/manifest.json` file in [this repo](https://github.com/launchdarkly/integration-framework/tree/master/integrations) for a full list of fields for the integration you wish to configure. **IMPORTANT**: Please note that Terraform will only accept these in snake case, regardless of the case shown in the manifest.",
			},
			ON: schema.BoolAttribute{
				Required:    true,
				Description: "Whether or not you want your subscription enabled, i.e. to actively send events.",
			},
			TAGS: schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Tags associated with your resource.",
			},
			STATEMENTS: frameworkPolicyStatementsResourceAttribute(true, "The resources to which you wish to subscribe.", ""),
		},
	}
}

func (r *AuditLogSubscriptionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureResourceClient(req, resp)
}

func (r *AuditLogSubscriptionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AuditLogSubscriptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	integrationKey := plan.IntegrationKey.ValueString()
	name := plan.Name.ValueString()
	on := plan.On.ValueBool()

	tags, d := stringSliceFromSet(ctx, plan.Tags)
	resp.Diagnostics.Append(d...)

	rawConfig, d := mapStringFromAttr(ctx, plan.Config)
	resp.Diagnostics.Append(d...)

	statements, d := frameworkPolicyStatementsFromList(ctx, plan.Statements)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiConfig, err := convertSubscriptionConfigToAPI(integrationKey, rawConfig)
	if err != nil {
		// One-line form so ExpectError regex matches against summary
		// only — the detail field is rendered separately by the CLI.
		resp.Diagnostics.AddError(fmt.Sprintf("failed to create %s integration with name %s: %s", integrationKey, name, err.Error()), "")
		return
	}

	body := ldapi.SubscriptionPost{
		Name:       name,
		On:         &on,
		Tags:       tags,
		Config:     apiConfig,
		Statements: statements,
	}

	var sub *ldapi.Integration
	err = r.client.withConcurrency(r.client.ctx, func() error {
		var e error
		sub, _, e = r.client.ld.IntegrationAuditLogSubscriptionsApi.CreateSubscription(r.client.ctx, integrationKey).SubscriptionPost(body).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("failed to create %s integration with name %s", integrationKey, name), err)
		return
	}
	if sub.Id == nil {
		resp.Diagnostics.AddError("Missing subscription ID", "LaunchDarkly returned a subscription without an ID")
		return
	}

	plan.ID = types.StringValue(*sub.Id)
	r.readIntoModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AuditLogSubscriptionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AuditLogSubscriptionResourceModel
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

func (r *AuditLogSubscriptionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AuditLogSubscriptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	integrationKey := plan.IntegrationKey.ValueString()
	name := plan.Name.ValueString()
	on := plan.On.ValueBool()
	id := plan.ID.ValueString()

	tags, d := stringSliceFromSet(ctx, plan.Tags)
	resp.Diagnostics.Append(d...)

	rawConfig, d := mapStringFromAttr(ctx, plan.Config)
	resp.Diagnostics.Append(d...)

	statements, d := frameworkPolicyStatementsFromList(ctx, plan.Statements)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiConfig, err := convertSubscriptionConfigToAPI(integrationKey, rawConfig)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("failed to update %s integration %q", integrationKey, id), err.Error())
		return
	}

	patch := []ldapi.PatchOperation{
		patchReplace("/name", &name),
		patchReplace("/tags", &tags),
		patchReplace("/config", &apiConfig),
		patchReplace("/on", &on),
		patchReplace("/statements", &statements),
	}

	err = r.client.withConcurrency(r.client.ctx, func() error {
		_, _, e := r.client.ld.IntegrationAuditLogSubscriptionsApi.UpdateSubscription(r.client.ctx, integrationKey, id).PatchOperation(patch).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("failed to update %q integration with name %q and ID %q", integrationKey, name, id), err)
		return
	}

	r.readIntoModel(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AuditLogSubscriptionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AuditLogSubscriptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.withConcurrency(r.client.ctx, func() error {
		_, e := r.client.ld.IntegrationAuditLogSubscriptionsApi.DeleteSubscription(r.client.ctx, data.IntegrationKey.ValueString(), data.ID.ValueString()).Execute()
		return e
	})
	if err != nil {
		addLdapiError(&resp.Diagnostics, fmt.Sprintf("failed to delete integration with ID %q", data.ID.ValueString()), err)
	}
}

// ImportState expects "integrationKey/integrationID".
func (r *AuditLogSubscriptionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("found unexpected id format for import: %q. expected format: 'integrationKey/integration_id'", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(INTEGRATION_KEY), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *AuditLogSubscriptionResource) readIntoModel(ctx context.Context, data *AuditLogSubscriptionResourceModel, diags *diag.Diagnostics) {
	integrationKey := data.IntegrationKey.ValueString()
	id := data.ID.ValueString()

	var sub *ldapi.Integration
	var res *http.Response
	var err error
	err = r.client.withConcurrency(r.client.ctx, func() error {
		sub, res, err = r.client.ld.IntegrationAuditLogSubscriptionsApi.GetSubscriptionByID(r.client.ctx, integrationKey, id).Execute()
		return err
	})
	if err != nil {
		if isStatusNotFound(res) {
			data.ID = types.StringNull()
			return
		}
		diags.AddError(fmt.Sprintf("failed to get integration with ID %q", id), handleLdapiErr(err).Error())
		return
	}

	if sub.Id != nil {
		data.ID = types.StringValue(*sub.Id)
	}
	if sub.Name != nil {
		data.Name = types.StringValue(*sub.Name)
	}
	if sub.On != nil {
		data.On = types.BoolValue(*sub.On)
	} else {
		data.On = types.BoolValue(false)
	}

	// Reconstruct config map: API returns camelCase / kebab-case keys;
	// terraform schema uses snake_case. Preserve secrets from prior state
	// so a re-plan doesn't surface drift on obfuscated server responses.
	priorConfig, _ := mapStringFromAttr(ctx, data.Config)
	updated, cerr := convertSubscriptionConfigFromAPI(integrationKey, sub.Config, priorConfig)
	if cerr != nil {
		diags.AddError(fmt.Sprintf("failed to convert config for integration %q", id), cerr.Error())
		return
	}
	configVal, d := types.MapValueFrom(ctx, types.StringType, updated)
	diags.Append(d...)
	data.Config = configVal

	stmtList, d := frameworkPolicyStatementsValue(ctx, sub.Statements)
	diags.Append(d...)
	data.Statements = stmtList

	tagsSet, d := setFromStringSlice(ctx, sub.Tags)
	diags.Append(d...)
	data.Tags = tagsSet
}

// convertSubscriptionConfigToAPI translates user-facing snake_case keys
// into each integration's manifest casing (camelCase for most,
// kebab-case for the integrations listed in KEBAB_CASE_INTEGRATIONS,
// with the datadog hostUrl->hostURL override preserved).
func convertSubscriptionConfigToAPI(integrationKey string, userConfig map[string]string) (map[string]interface{}, error) {
	configMap := getSubscriptionConfigurationMap()
	configFormat, ok := configMap[integrationKey]
	if !ok {
		return nil, fmt.Errorf("%s is not a valid integration_key for audit log subscriptions", integrationKey)
	}

	for k := range userConfig {
		key := getConfigFieldKey(integrationKey, k)
		if integrationKey == "datadog" && key == "hostUrl" {
			key = "hostURL"
		}
		if _, ok := configFormat[key]; !ok {
			return nil, fmt.Errorf("config variable %s not valid for integration type %s", k, integrationKey)
		}
	}

	converted := make(map[string]interface{}, len(userConfig))
	for k, v := range configFormat {
		key := strcase.SnakeCase(k)
		rawValue, ok := userConfig[key]
		if !ok {
			if !v.IsOptional {
				return nil, fmt.Errorf("config variable %s must be set", key)
			}
			continue
		}
		switch v.Type {
		case "string", "uri":
			converted[k] = rawValue
		case "boolean":
			b, err := strconv.ParseBool(rawValue)
			if err != nil {
				return nil, fmt.Errorf("config value %s for %v must be of type bool", rawValue, k)
			}
			converted[k] = b
		case "enum":
			if !stringInSlice(rawValue, v.AllowedValues) {
				return nil, fmt.Errorf("config value %s for %v must be one of the following approved string values: %v", rawValue, k, v.AllowedValues)
			}
			converted[k] = rawValue
		default:
			converted[k] = rawValue
		}
	}
	return converted, nil
}

// convertSubscriptionConfigFromAPI normalizes API output back into the
// user-facing snake_case map. priorState carries the previously-set
// values; secret fields are passed through unchanged because the API
// obfuscates them on read, and absent (non-user-set) optional fields
// are dropped to avoid surfacing API defaults as drift.
func convertSubscriptionConfigFromAPI(integrationKey string, apiConfig map[string]interface{}, priorState map[string]string) (map[string]string, error) {
	configMap := getSubscriptionConfigurationMap()
	configFormat, ok := configMap[integrationKey]
	if !ok {
		return nil, fmt.Errorf("%s is not a currently supported integration_key for audit log subscriptions", integrationKey)
	}
	out := make(map[string]string, len(apiConfig))
	for k, v := range apiConfig {
		key := strcase.SnakeCase(k)
		if _, setByUser := priorState[key]; !setByUser {
			continue
		}
		if configFormat[k].IsSecret {
			out[key] = priorState[key]
			continue
		}
		if b, isBool := v.(bool); isBool {
			out[key] = strconv.FormatBool(b)
			continue
		}
		out[key] = fmt.Sprintf("%v", v)
	}
	return out, nil
}
