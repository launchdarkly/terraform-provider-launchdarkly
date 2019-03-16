package launchdarkly

import (
	"context"
	"errors"

	ldapi "github.com/launchdarkly/api-client-go"
)

// Client is used by the provider to access the ld API.
type Client struct {
	apiKey string
	ld     *ldapi.APIClient
	ctx    context.Context
}

func newClient(apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, errors.New("apiKey cannot be empty")
	}

	return &Client{
		apiKey: apiKey,
		ld:     ldapi.NewAPIClient(ldapi.NewConfiguration()),
		ctx: context.WithValue(context.Background(), ldapi.ContextAPIKey, ldapi.APIKey{
			Key: apiKey,
		}),
	}, nil
}
