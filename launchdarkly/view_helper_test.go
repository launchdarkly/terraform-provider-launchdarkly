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

	ldapi "github.com/launchdarkly/api-client-go/v23"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
)

func TestViewRequestsIncludeUserAgentHeader(t *testing.T) {
	projectKey := "test-project"
	viewKey := "test-view"
	expectedUA := fmt.Sprintf("launchdarkly-terraform-provider/%s", version)

	userAgentCh := make(chan string, 1)
	apiVersionValuesCh := make(chan []string, 1)
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case userAgentCh <- r.Header.Get("User-Agent"):
		default:
		}
		select {
		case apiVersionValuesCh <- append([]string(nil), r.Header.Values("LD-API-Version")...):
		default:
		}

		if r.URL.Path != fmt.Sprintf("/api/v2/projects/%s/views/%s", projectKey, viewKey) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(map[string]interface{}{
			"id":                 "view-id",
			"accountId":          "account-id",
			"_affectsSdkPayload": false,
			"projectId":          "project-id",
			"projectKey":         projectKey,
			"key":                viewKey,
			"name":               "Test View",
			"description":        "",
			"version":            1,
			"tags":               []string{},
			"createdAt":          0,
			"updatedAt":          0,
			"archived":           false,
			"deleted":            false,
		})
		require.NoError(t, err)
	}))
	t.Cleanup(ts.Close)

	cfg := ldapi.NewConfiguration()
	cfg.Scheme = "https"
	cfg.Host = strings.TrimPrefix(ts.URL, "https://")
	cfg.UserAgent = expectedUA
	cfg.HTTPClient = ts.Client()

	client := &Client{
		apiKey:    "test-token",
		apiHost:   strings.TrimPrefix(ts.URL, "https://"),
		ld:        ldapi.NewAPIClient(cfg),
		semaphore: semaphore.NewWeighted(1),
		ctx: context.WithValue(context.Background(), ldapi.ContextAPIKeys, map[string]ldapi.APIKey{
			"ApiKey": {Key: "test-token"},
		}),
	}

	_, _, err := getViewRaw(client, projectKey, viewKey)
	require.NoError(t, err)

	select {
	case ua := <-userAgentCh:
		require.Equal(t, expectedUA, ua)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for User-Agent header")
	}
	select {
	case apiVersionValues := <-apiVersionValuesCh:
		require.Equal(t, []string{"beta"}, apiVersionValues)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for LD-API-Version header")
	}
}
