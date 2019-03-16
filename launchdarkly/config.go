package launchdarkly

import (
	"context"
	"github.com/launchdarkly/api-client-go"
)

// Client is used by the provider to access the ld API.
type Client struct {
	apiKey string
	ld     *ldapi.APIClient
	ctx    context.Context
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		ld:     ldapi.NewAPIClient(ldapi.NewConfiguration()),
		ctx: context.WithValue(context.Background(), ldapi.ContextAPIKey, ldapi.APIKey{
			Key: apiKey,
		}),
	}
}
