package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
)

var (
	_ ephemeral.EphemeralResource              = &SdkKeyEphemeralResource{}
	_ ephemeral.EphemeralResourceWithConfigure = &SdkKeyEphemeralResource{}
)

// SdkKeyEphemeralResource is the ephemeral counterpart of the
// launchdarkly_sdk_key data source. It performs the same lookup but its
// result, including the key value, is never persisted to plan or state.
type SdkKeyEphemeralResource struct {
	client *Client
}

func NewSdkKeyEphemeralResource() ephemeral.EphemeralResource {
	return &SdkKeyEphemeralResource{}
}

func (r *SdkKeyEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sdk_key"
}

func (r *SdkKeyEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Provides a LaunchDarkly SDK key as an ephemeral resource.

~> **Beta:** This ephemeral resource uses a beta API. Beta resources may change or be removed in future versions.

This ephemeral resource retrieves information about a specific SDK key in a project environment, like the ` + "`launchdarkly_sdk_key`" + ` data source, but its values are never persisted to the Terraform plan or state. Use it to pass the key ` + "`value`" + ` to [write-only arguments](https://developer.hashicorp.com/terraform/language/resources/ephemeral#write-only-arguments), provider configuration blocks, or other ephemeral contexts.

This ephemeral resource only reads the SDK key. It never creates, resets, or rotates keys.

-> **Note:** Ephemeral resources require Terraform 1.10 or later. Write-only arguments require Terraform 1.11 or later.`,
		Attributes: map[string]schema.Attribute{
			ID:              schema.StringAttribute{Computed: true, Description: "The unique ID in the format `project_key/environment_key/key`."},
			PROJECT_KEY:     schema.StringAttribute{Required: true, Description: "The project key."},
			ENVIRONMENT_KEY: schema.StringAttribute{Required: true, Description: "The environment key."},
			KEY:             schema.StringAttribute{Required: true, Description: "The user-defined identifying key of the SDK key."},
			KIND:            schema.StringAttribute{Computed: true, Description: "The kind of SDK key. Either `sdk` (server-side) or `mobile`."},
			NAME:            schema.StringAttribute{Computed: true, Description: "The human-readable name of the SDK key."},
			DESCRIPTION:     schema.StringAttribute{Computed: true, Description: "The description of the SDK key."},
			EXPIRY:          schema.Int64Attribute{Computed: true, Description: "The expiration date for the SDK key, expressed as a Unix epoch time in milliseconds, if set."},
			VALUE: schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The actual SDK key value. Use this when configuring your SDK.",
			},
			IS_DEFAULT: schema.BoolAttribute{Computed: true, Description: "Whether this SDK key is the system-defined default for the environment."},
			VERSION:    schema.Int64Attribute{Computed: true, Description: "The auto-incremented version number of the SDK key."},
		},
	}
}

func (r *SdkKeyEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	r.client = configureEphemeralResourceClient(req, resp)
}

func (r *SdkKeyEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured LaunchDarkly client", "The provider has not been configured. Please report this issue to the provider developers.")
		return
	}

	var data SdkKeyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	beta, err := r.client.betaClientFromConfig()
	if err != nil {
		resp.Diagnostics.AddError("Failed to construct beta client", err.Error())
		return
	}

	projectKey := data.ProjectKey.ValueString()
	environmentKey := data.EnvironmentKey.ValueString()
	sdkKeyKey := data.Key.ValueString()

	sdkKey, _, err := getSdkKey(beta, projectKey, environmentKey, sdkKeyKey)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to get SDK key %q in environment %q of project %q: %s", sdkKeyKey, environmentKey, projectKey, handleLdapiErr(err).Error()),
			"",
		)
		return
	}

	sdkKeyDataSourceModelFromAPI(projectKey, environmentKey, sdkKey, &data)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}
