package launchdarkly

// A minimal test-only "echo" provider built on terraform-plugin-framework.
//
// Ephemeral values never reach plan or state, so acceptance tests assert on
// them by passing them into this provider's `data` configuration and reading
// them back from the `data` attribute of an `echo` resource. This mirrors
// terraform-plugin-testing's echoprovider package, which cannot be used with
// the pinned terraform-plugin-testing version because its server does not
// implement the terraform-plugin-go interface that this provider's framework
// version requires.

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type testEchoProvider struct{}

type testEchoModel struct {
	Data types.Dynamic `tfsdk:"data"`
}

func newTestEchoProvider() provider.Provider { return &testEchoProvider{} }

func (p *testEchoProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "echo"
}

func (p *testEchoProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerschema.Schema{
		Attributes: map[string]providerschema.Attribute{
			"data": providerschema.DynamicAttribute{Optional: true},
		},
	}
}

func (p *testEchoProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg testEchoModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	resp.ResourceData = cfg.Data
}

func (p *testEchoProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

func (p *testEchoProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{func() resource.Resource { return &testEchoResource{} }}
}

type testEchoResource struct {
	data types.Dynamic
}

func (r *testEchoResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName
}

func (r *testEchoResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Attributes: map[string]resourceschema.Attribute{
			"data": resourceschema.DynamicAttribute{Computed: true},
		},
	}
}

func (r *testEchoResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if d, ok := req.ProviderData.(types.Dynamic); ok {
		r.data = d
	}
}

func (r *testEchoResource) Create(ctx context.Context, _ resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, &testEchoModel{Data: r.data})...)
}

func (r *testEchoResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
	// Keep the echoed data from the last apply.
}

func (r *testEchoResource) Update(ctx context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, &testEchoModel{Data: r.data})...)
}

func (r *testEchoResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}
