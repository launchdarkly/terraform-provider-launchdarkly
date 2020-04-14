package launchdarkly

import (
	"context"
	"errors"
	"fmt"

	ldapi "github.com/launchdarkly/api-client-go"
)

const (
	AgentVersion = "1.2.1"
	APIVersion   = "20160426"
)

// Client is used by the provider to access the ld API.
type Client struct {
	apiKey  string
	apiHost string
	ld      *ldapi.APIClient
	ctx     context.Context
}

func newClient(token string, apiHost string, oauth bool) (*Client, error) {
	if token == "" {
		return nil, errors.New("token cannot be empty")
	}
	basePath := "https://app.launchdarkly.com/api/v2"
	if apiHost != "" {
		basePath = fmt.Sprintf("%s/api/v2", apiHost)
	}

	cfg := &ldapi.Configuration{
		BasePath:      basePath,
		DefaultHeader: make(map[string]string),
		UserAgent:     fmt.Sprintf("launchdarkly-terraform-provider/%s", AgentVersion),
	}

	cfg.AddDefaultHeader("LD-API-Version", APIVersion)

	ctx := context.WithValue(context.Background(), ldapi.ContextAPIKey, ldapi.APIKey{
		Key: token,
	})
	if oauth {
		ctx = context.WithValue(context.Background(), ldapi.ContextAccessToken, token)
	}

	return &Client{
		apiKey:  token,
		apiHost: apiHost,
		ld:      ldapi.NewAPIClient(cfg),
		ctx:     ctx,
	}, nil
}
