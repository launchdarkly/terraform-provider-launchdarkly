package launchdarkly

import (
	"context"

	ldapi "github.com/launchdarkly/api-client-go"
)

// Config is used to configure the creation of a LaunchDarkly client.
type Config struct {
	APIKey string
}

// Client is used by the provider to access the LaunchDarkly API.
type Client struct {
	LaunchDarkly *ldapi.APIClient
	Ctx          context.Context
}

// New returns a configured LaunchDarkly client.
func (c *Config) New() interface{} {
	ctx := context.WithValue(context.Background(), ldapi.ContextAPIKey, ldapi.APIKey{
		Key: c.APIKey,
	})

	return &Client{
		LaunchDarkly: ldapi.NewAPIClient(ldapi.NewConfiguration()),
		Ctx:          ctx,
	}
}
