package launchdarkly

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateFeatureFlagWithViewKeysSetsUserAgentHeader(t *testing.T) {
	t.Parallel()

	var gotUserAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserAgent = r.Header.Get("User-Agent")
		require.Equal(t, "/api/v2/flags/test-project", r.URL.Path)
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client, err := newClient("token", server.URL, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	err = createFeatureFlagWithViewKeys(client, "test-project", FeatureFlagBodyWithViewKeys{
		Name: "test-flag",
		Key:  "test-flag",
	})
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("launchdarkly-terraform-provider/%s", version), gotUserAgent)
}

func TestCreateSegmentWithViewKeysSetsUserAgentHeader(t *testing.T) {
	t.Parallel()

	var gotUserAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserAgent = r.Header.Get("User-Agent")
		require.Equal(t, "/api/v2/segments/test-project/test-env", r.URL.Path)
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client, err := newClient("token", server.URL, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	err = createSegmentWithViewKeys(client, "test-project", "test-env", SegmentBodyWithViewKeys{
		Name: "test-segment",
		Key:  "test-segment",
	})
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("launchdarkly-terraform-provider/%s", version), gotUserAgent)
}

func TestGetProjectViewSettingsSetsUserAgentHeader(t *testing.T) {
	t.Parallel()

	var gotUserAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserAgent = r.Header.Get("User-Agent")
		require.Equal(t, "/api/v2/projects/test-project", r.URL.Path)
		_, _ = io.WriteString(w, `{"requireViewAssociationForNewFlags":true,"requireViewAssociationForNewSegments":false}`)
	}))
	defer server.Close()

	client, err := newClient("token", server.URL, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	_, err = getProjectViewSettings(client, "test-project")
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("launchdarkly-terraform-provider/%s", version), gotUserAgent)
}

func TestPatchProjectViewSettingsSetsUserAgentHeader(t *testing.T) {
	t.Parallel()

	var gotUserAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserAgent = r.Header.Get("User-Agent")
		require.Equal(t, http.MethodPatch, r.Method)
		require.Equal(t, "/api/v2/projects/test-project", r.URL.Path)
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := newClient("token", server.URL, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)

	err = patchProjectViewSettings(client, "test-project", true, false, true, false)
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("launchdarkly-terraform-provider/%s", version), gotUserAgent)
}
