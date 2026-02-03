package launchdarkly

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ldapi "github.com/launchdarkly/api-client-go/v17"
	"github.com/stretchr/testify/require"
)

func TestSetViewRequestHeaders(t *testing.T) {
	req, err := http.NewRequest("GET", "http://example.com", nil)
	require.NoError(t, err)

	setViewRequestHeaders(req, "test-api-key")

	require.Equal(t, "test-api-key", req.Header.Get("Authorization"))
	require.Equal(t, "application/json", req.Header.Get("Content-Type"))
	require.Equal(t, "beta", req.Header.Get("LD-API-Version"))
	require.Equal(t, fmt.Sprintf("launchdarkly-terraform-provider/%s", version), req.Header.Get("User-Agent"))
}

func TestViewRequestsIncludeUserAgentHeader(t *testing.T) {
	projectKey := "test-project"
	viewKey := "test-view"
	expectedUA := fmt.Sprintf("launchdarkly-terraform-provider/%s", version)

	headerCh := make(chan string, 1)
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case headerCh <- r.Header.Get("User-Agent"):
		default:
		}

		if r.URL.Path != fmt.Sprintf("/api/v2/projects/%s/views/%s", projectKey, viewKey) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(View{
			Id:         "view-id",
			Key:        viewKey,
			Name:       "Test View",
			ProjectKey: projectKey,
		})
		require.NoError(t, err)
	}))
	t.Cleanup(ts.Close)

	cfg := ldapi.NewConfiguration()
	cfg.Scheme = "https"
	cfg.Host = strings.TrimPrefix(ts.URL, "https://")
	cfg.HTTPClient = ts.Client()

	client := &Client{
		apiKey:  "test-token",
		apiHost: strings.TrimPrefix(ts.URL, "https://"),
		ld:      ldapi.NewAPIClient(cfg),
		ctx: context.WithValue(context.Background(), ldapi.ContextAPIKeys, map[string]ldapi.APIKey{
			"ApiKey": {Key: "test-token"},
		}),
	}

	_, _, err := getViewRaw(client, projectKey, viewKey)
	require.NoError(t, err)

	select {
	case ua := <-headerCh:
		require.Equal(t, expectedUA, ua)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for User-Agent header")
	}
}
