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

func TestPluginProviderParsesApiHostWithoutScheme(t *testing.T) {
	pluginProvider := NewPluginProvider("test")()
	schemaResponse := provider.SchemaResponse{}
	ctx := context.Background()
	pluginProvider.Schema(ctx, provider.SchemaRequest{}, &schemaResponse)

	configureRequest := provider.ConfigureRequest{
		Config: tfsdk.Config{
			Raw: tftypes.NewValue(tftypes.Object{
				AttributeTypes: map[string]tftypes.Type{
					API_HOST:     tftypes.String,
					ACCESS_TOKEN: tftypes.String,
					OAUTH_TOKEN:  tftypes.String,
					HTTP_TIMEOUT: tftypes.Number,
				},
			}, map[string]tftypes.Value{
				API_HOST:     tftypes.NewValue(tftypes.String, "https://test.com"),
				ACCESS_TOKEN: tftypes.NewValue(tftypes.String, "test-token"),
				HTTP_TIMEOUT: tftypes.NewValue(tftypes.Number, 0),
				OAUTH_TOKEN:  tftypes.NewValue(tftypes.String, ""),
			}),
			Schema: schemaResponse.Schema,
		},
	}

	configureResp := provider.ConfigureResponse{}
	pluginProvider.Configure(ctx, configureRequest, &configureResp)
	require.Len(t, configureResp.Diagnostics, 0)

	assert.Equal(t, configureResp.ResourceData.(*Client).apiHost, "test.com")
}
