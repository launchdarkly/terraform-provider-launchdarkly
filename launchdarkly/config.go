package launchdarkly

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	ldapi "github.com/launchdarkly/api-client-go/v7"
)

// The version string gets updated at build time using -ldflags
var version = "unreleased"

const (
	APIVersion = "20191212"
)

// Client is used by the provider to access the ld API.
type Client struct {
	apiKey         string
	apiHost        string
	ld             *ldapi.APIClient
	ctx            context.Context
	fallbackClient *http.Client
}

func newClient(token string, apiHost string, oauth bool) (*Client, error) {
	if token == "" {
		return nil, errors.New("token cannot be empty")
	}

	cfg := ldapi.NewConfiguration()
	cfg.Host = apiHost
	cfg.DefaultHeader = make(map[string]string)
	cfg.UserAgent = fmt.Sprintf("launchdarkly-terraform-provider/%s", version)

	cfg.AddDefaultHeader("LD-API-Version", APIVersion)

	ctx := context.WithValue(context.Background(), ldapi.ContextAPIKeys, map[string]ldapi.APIKey{
		"ApiKey": {
			Key: token,
		}})
	if oauth {
		ctx = context.WithValue(context.Background(), ldapi.ContextAccessToken, token)
	}

	// TODO: remove this once we get the go client reset endpoint fixed
	fallbackClient := http.Client{
		Timeout: time.Duration(5 * time.Second),
	}

	return &Client{
		apiKey:         token,
		apiHost:        apiHost,
		ld:             ldapi.NewAPIClient(cfg),
		ctx:            ctx,
		fallbackClient: &fallbackClient,
	}, nil
}
