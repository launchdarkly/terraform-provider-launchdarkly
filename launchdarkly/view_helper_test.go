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

	ldapi "github.com/launchdarkly/api-client-go/v24"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
)

// newViewTestClient returns a Client pointed at ts.
func newViewTestClient(t *testing.T, ts *httptest.Server) *Client {
	t.Helper()

	cfg := ldapi.NewConfiguration()
	cfg.Scheme = "https"
	cfg.Host = strings.TrimPrefix(ts.URL, "https://")
	cfg.HTTPClient = ts.Client()

	return &Client{
		apiKey:    "test-token",
		apiHost:   strings.TrimPrefix(ts.URL, "https://"),
		ld:        ldapi.NewAPIClient(cfg),
		semaphore: semaphore.NewWeighted(1),
		ctx: context.WithValue(context.Background(), ldapi.ContextAPIKeys, map[string]ldapi.APIKey{
			"ApiKey": {Key: "test-token"},
		}),
	}
}

func TestValidateViewKeysExist(t *testing.T) {
	projectKey := "test-project"
	existing := map[string]bool{"payments-team": true, "frontend-team": true}

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		prefix := fmt.Sprintf("/api/v2/projects/%s/views/", projectKey)
		key := strings.TrimPrefix(r.URL.Path, prefix)
		if !strings.HasPrefix(r.URL.Path, prefix) || !existing[key] {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "view-id", "accountId": "account-id", "_affectsSdkPayload": false,
			"projectId": "project-id", "projectKey": projectKey, "key": key,
			"name": key, "description": "", "version": 1, "tags": []string{},
			"createdAt": 0, "updatedAt": 0, "deleted": false,
		}))
	}))
	t.Cleanup(ts.Close)

	client := newViewTestClient(t, ts)

	t.Run("all keys resolve", func(t *testing.T) {
		require.NoError(t, validateViewKeysExist(client, projectKey, "flag", []string{"payments-team", "frontend-team"}))
	})

	t.Run("no keys", func(t *testing.T) {
		require.NoError(t, validateViewKeysExist(client, projectKey, "flag", nil))
	})

	t.Run("missing key errors with guidance", func(t *testing.T) {
		err := validateViewKeysExist(client, projectKey, "flag", []string{"payments-team", "paymentz-team"})
		require.Error(t, err)
		// The offending key, not just the first one checked.
		require.Contains(t, err.Error(), `"paymentz-team"`)
		require.Contains(t, err.Error(), "view does not exist")
		// Points at the fix rather than only reporting the symptom.
		require.Contains(t, err.Error(), "launchdarkly_view.my_view.key")
		require.Contains(t, err.Error(), "lowercase")
	})

	// viewKeyValidator rejects uppercase keys at plan time, so this input can
	// no longer arrive from a plan. The helper still must not lowercase it on
	// the caller's behalf: it looks up exactly what it was handed, so a key
	// reaching it by any other path (import, drift reconcile) fails loudly
	// instead of silently resolving to a different view.
	t.Run("keys are looked up verbatim, not lowercased", func(t *testing.T) {
		err := validateViewKeysExist(client, projectKey, "segment", []string{"Payments-Team"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "segment")
		require.Contains(t, err.Error(), `"Payments-Team"`)
	})
}

func TestBetaClientFromConfigInheritsProviderSettings(t *testing.T) {
	t.Run("inherits configured values", func(t *testing.T) {
		client, err := newClient("test-token", "app.launchdarkly.com", false, 37, 5)
		require.NoError(t, err)

		beta, err := client.betaClientFromConfig()
		require.NoError(t, err)
		require.Equal(t, 37, beta.httpTimeout)
		require.Equal(t, 5, beta.maxConcurrency)
		require.Equal(t, 37*time.Second, beta.ld.GetConfig().HTTPClient.Timeout)
		// Beta clients must not send a default LD-API-Version header.
		require.NotContains(t, beta.ld.GetConfig().DefaultHeader, "LD-API-Version")
	})

	t.Run("falls back to defaults when unset", func(t *testing.T) {
		// Simulates a Client built without going through baseNewClient.
		client := &Client{apiKey: "test-token", apiHost: "app.launchdarkly.com"}

		beta, err := client.betaClientFromConfig()
		require.NoError(t, err)
		require.Equal(t, DEFAULT_HTTP_TIMEOUT_S, beta.httpTimeout)
		require.Equal(t, DEFAULT_MAX_CONCURRENCY, beta.maxConcurrency)
	})
}

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
