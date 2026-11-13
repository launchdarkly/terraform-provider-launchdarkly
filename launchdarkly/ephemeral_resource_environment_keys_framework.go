package launchdarkly

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v24"
)

var (
	_ ephemeral.EphemeralResource              = &EnvironmentKeysEphemeralResource{}
	_ ephemeral.EphemeralResourceWithConfigure = &EnvironmentKeysEphemeralResource{}
)

// EnvironmentKeysEphemeralResource reads an environment's SDK key, mobile key,
// and client-side ID without persisting them to Terraform plan or state. It
// only ever issues a GET against the environment: it never resets or rotates
// any key.
type EnvironmentKeysEphemeralResource struct {
	client *Client
}

type EnvironmentKeysEphemeralResourceModel struct {
	ProjectKey   types.String `tfsdk:"project_key"`
	EnvKey       types.String `tfsdk:"env_key"`
	APIKey       types.String `tfsdk:"api_key"`
	MobileKey    types.String `tfsdk:"mobile_key"`
	ClientSideID types.String `tfsdk:"client_side_id"`
}

func NewEnvironmentKeysEphemeralResource() ephemeral.EphemeralResource {
	return &EnvironmentKeysEphemeralResource{}
}

func (r *EnvironmentKeysEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_keys"
}

func (r *EnvironmentKeysEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides the SDK key, mobile key, and client-side ID of a LaunchDarkly environment as an ephemeral resource.\n\n" +
			"Ephemeral resources are opened on every Terraform operation and their values are never persisted to the plan or state. Use this ephemeral resource to pass an environment's keys to [write-only arguments](https://developer.hashicorp.com/terraform/language/resources/ephemeral#write-only-arguments), such as `value_wo` on `aws_ssm_parameter`, provider configuration blocks, or other ephemeral contexts, without storing them in plaintext in your Terraform state.\n\n" +
			"This ephemeral resource only reads the environment. It never resets or rotates its keys.\n\n" +
			"-> **Note:** Ephemeral resources require Terraform 1.10 or later. Write-only arguments require Terraform 1.11 or later.",
		Attributes: map[string]schema.Attribute{
			PROJECT_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The environment's project key.",
			},
			ENV_KEY: schema.StringAttribute{
				Required:    true,
				Description: "The project-unique key for the environment.",
			},
			API_KEY: schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The environment's server-side SDK key.",
			},
			MOBILE_KEY: schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The environment's mobile key.",
			},
			CLIENT_SIDE_ID: schema.StringAttribute{
				Computed:    true,
				Description: "The environment's client-side ID. This value is not secret: client-side SDKs embed it in the code they ship to browsers.",
			},
		},
	}
}

func (r *EnvironmentKeysEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	r.client = configureEphemeralResourceClient(req, resp)
}

func (r *EnvironmentKeysEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured LaunchDarkly client", "The provider has not been configured. Please report this issue to the provider developers.")
		return
	}

	var data EnvironmentKeysEphemeralResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	envKey := data.EnvKey.ValueString()

	var env *ldapi.Environment
	err := r.client.withConcurrency(r.client.ctx, func() error {
		var e error
		env, _, e = r.client.ld.EnvironmentsApi.GetEnvironment(r.client.ctx, projectKey, envKey).Execute()
		return e
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("failed to get environment with key %q for project key: %q: %s", envKey, projectKey, handleLdapiErr(err).Error()),
			"",
		)
		return
	}

	environmentKeysModelFromAPI(env, &data)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}

// environmentKeysModelFromAPI copies the environment's keys onto the
// ephemeral result model.
func environmentKeysModelFromAPI(env *ldapi.Environment, data *EnvironmentKeysEphemeralResourceModel) {
	data.APIKey = types.StringValue(env.ApiKey)
	data.MobileKey = types.StringValue(env.MobileKey)
	data.ClientSideID = types.StringValue(env.Id)
}
