package launchdarkly

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderRegistersEphemeralResources(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	p, ok := NewPluginProvider("test")().(provider.ProviderWithEphemeralResources)
	require.True(t, ok, "provider must implement ProviderWithEphemeralResources")

	names := map[string]bool{}
	for _, factory := range p.EphemeralResources(ctx) {
		r := factory()
		var meta ephemeral.MetadataResponse
		r.Metadata(ctx, ephemeral.MetadataRequest{ProviderTypeName: "launchdarkly"}, &meta)
		names[meta.TypeName] = true

		var schemaResp ephemeral.SchemaResponse
		r.Schema(ctx, ephemeral.SchemaRequest{}, &schemaResp)
		require.False(t, schemaResp.Diagnostics.HasError(), "%s: %v", meta.TypeName, schemaResp.Diagnostics)
		require.False(t, schemaResp.Schema.ValidateImplementation(ctx).HasError(), meta.TypeName)

		_, ok := r.(ephemeral.EphemeralResourceWithConfigure)
		assert.True(t, ok, "%s must implement EphemeralResourceWithConfigure", meta.TypeName)
	}
	assert.True(t, names["launchdarkly_environment_keys"])
	assert.True(t, names["launchdarkly_sdk_key"])
}

func TestEphemeralResourceSecretsAreSensitive(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var envResp ephemeral.SchemaResponse
	NewEnvironmentKeysEphemeralResource().Schema(ctx, ephemeral.SchemaRequest{}, &envResp)
	assert.True(t, envResp.Schema.Attributes[API_KEY].IsSensitive())
	assert.True(t, envResp.Schema.Attributes[MOBILE_KEY].IsSensitive())

	var sdkResp ephemeral.SchemaResponse
	NewSdkKeyEphemeralResource().Schema(ctx, ephemeral.SchemaRequest{}, &sdkResp)
	assert.True(t, sdkResp.Schema.Attributes[VALUE].IsSensitive())
}

func TestProviderConfigureSetsEphemeralResourceData(t *testing.T) {
	pluginProvider := NewPluginProvider("test")()
	resp := provider.ConfigureResponse{}
	pluginProvider.Configure(context.Background(), newPluginProviderConfigureRequest(t, 0), &resp)
	require.Len(t, resp.Diagnostics, 0)
	_, ok := resp.EphemeralResourceData.(*Client)
	assert.True(t, ok, "ephemeral resources must receive the configured client")
}
