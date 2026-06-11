package launchdarkly

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPluginProviderConfigureRequest(t *testing.T, maxConcurrency int64) provider.ConfigureRequest {
	t.Helper()
	pluginProvider := NewPluginProvider("test")()
	schemaResponse := provider.SchemaResponse{}
	pluginProvider.Schema(context.Background(), provider.SchemaRequest{}, &schemaResponse)

	return provider.ConfigureRequest{
		Config: tfsdk.Config{
			Raw: tftypes.NewValue(tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					API_HOST:        tftypes.String,
					ACCESS_TOKEN:    tftypes.String,
					OAUTH_TOKEN:     tftypes.String,
					HTTP_TIMEOUT:    tftypes.Number,
					MAX_CONCURRENCY: tftypes.Number,
				},
			}, map[string]tftypes.Value{
				API_HOST:        tftypes.NewValue(tftypes.String, "https://test.com"),
				ACCESS_TOKEN:    tftypes.NewValue(tftypes.String, "test-token"),
				HTTP_TIMEOUT:    tftypes.NewValue(tftypes.Number, 0),
				OAUTH_TOKEN:     tftypes.NewValue(tftypes.String, ""),
				MAX_CONCURRENCY: tftypes.NewValue(tftypes.Number, maxConcurrency),
			}),
			Schema: schemaResponse.Schema,
		},
	}
}

func TestPluginProviderParsesApiHostWithoutScheme(t *testing.T) {
	pluginProvider := NewPluginProvider("test")()
	ctx := context.Background()

	configureResp := provider.ConfigureResponse{}
	pluginProvider.Configure(ctx, newPluginProviderConfigureRequest(t, 0), &configureResp)
	require.Len(t, configureResp.Diagnostics, 0)

	assert.Equal(t, configureResp.ResourceData.(*Client).apiHost, "test.com")
}

func TestPluginProviderMaxConcurrency(t *testing.T) {
	t.Run("accepts a value greater than 1", func(t *testing.T) {
		pluginProvider := NewPluginProvider("test")()
		configureResp := provider.ConfigureResponse{}
		pluginProvider.Configure(context.Background(), newPluginProviderConfigureRequest(t, 10), &configureResp)
		require.Len(t, configureResp.Diagnostics, 0)
		assert.NotNil(t, configureResp.ResourceData)
	})

	t.Run("rejects a negative value", func(t *testing.T) {
		pluginProvider := NewPluginProvider("test")()
		configureResp := provider.ConfigureResponse{}
		pluginProvider.Configure(context.Background(), newPluginProviderConfigureRequest(t, -1), &configureResp)
		require.True(t, configureResp.Diagnostics.HasError())
	})
}
