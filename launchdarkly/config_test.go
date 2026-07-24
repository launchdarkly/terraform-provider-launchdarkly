package launchdarkly

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLDClientConfigPreservesExplicitScheme(t *testing.T) {
	t.Parallel()

	cfg := newLDClientConfig("http://127.0.0.1:8080", DEFAULT_HTTP_TIMEOUT_S, APIVersion, standardRetryPolicy)
	assert.Equal(t, "127.0.0.1:8080", cfg.Host)
	assert.Equal(t, "http", cfg.Scheme)
}

func TestNewLDClientConfigWithoutSchemeUsesDefaultScheme(t *testing.T) {
	t.Parallel()

	cfg := newLDClientConfig("127.0.0.1:8080", DEFAULT_HTTP_TIMEOUT_S, APIVersion, standardRetryPolicy)
	assert.Equal(t, "127.0.0.1:8080", cfg.Host)
	assert.Equal(t, "", cfg.Scheme)
}

func TestNewLDClientConfigSetsDefaultAPIVersionHeaderForStandardClient(t *testing.T) {
	t.Parallel()

	cfg := newLDClientConfig("127.0.0.1:8080", DEFAULT_HTTP_TIMEOUT_S, APIVersion, standardRetryPolicy)
	assert.Equal(t, APIVersion, cfg.DefaultHeader["LD-API-Version"])
}

func TestNewLDClientConfigSkipsDefaultAPIVersionHeaderForBetaClient(t *testing.T) {
	t.Parallel()

	cfg := newLDClientConfig("127.0.0.1:8080", DEFAULT_HTTP_TIMEOUT_S, "beta", standardRetryPolicy)
	_, ok := cfg.DefaultHeader["LD-API-Version"]
	assert.False(t, ok)
}

func TestHandleRateLimits(t *testing.T) {
	t.Run("no retries needed", func(t *testing.T) {
		t.Parallel()
		calls := 0

		// create a test server
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		// create a client
		client, err := newClient("token", ts.URL, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
		require.NoError(t, err)

		res, err := client.ld.GetConfig().HTTPClient.Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, res.StatusCode, http.StatusOK)
		assert.Equal(t, calls, 1)
	})

	t.Run("max retries exceeded", func(t *testing.T) {
		t.Parallel()
		calls := 0

		// create a test server
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.Header().Add("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(100*time.Millisecond).UnixNano()/int64(time.Millisecond), 10))
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer ts.Close()

		// create a client
		client, err := newClient("token", ts.URL, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
		require.NoError(t, err)

		res, err := client.ld.GetConfig().HTTPClient.Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, res.StatusCode, http.StatusTooManyRequests)
		assert.Equal(t, calls, MAX_RETRIES+1)
	})

	t.Run("retry resolved with header", func(t *testing.T) {
		t.Parallel()
		calls := 0

		// create a test server
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++

			if calls == 3 {
				w.WriteHeader(http.StatusOK)
				return
			}

			w.Header().Add("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(100*time.Millisecond).UnixNano()/int64(time.Millisecond), 10))
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer ts.Close()

		// create a client
		client, err := newClient("token", ts.URL, false, 20, DEFAULT_MAX_CONCURRENCY)
		require.NoError(t, err)

		res, err := client.ld.GetConfig().HTTPClient.Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, res.StatusCode, http.StatusOK)
		assert.Equal(t, 3, calls)
	})

	t.Run("retry resolved with negative header", func(t *testing.T) {
		t.Parallel()
		calls := 0

		// create a test server
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++

			if calls == 3 {
				w.WriteHeader(http.StatusOK)
				return
			}

			w.Header().Add("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(-100*time.Millisecond).UnixNano()/int64(time.Millisecond), 10))
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer ts.Close()

		// create a client
		client, err := newClient("token", ts.URL, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
		require.NoError(t, err)

		res, err := client.ld.GetConfig().HTTPClient.Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, res.StatusCode, http.StatusOK)
		assert.Equal(t, 3, calls)
	})

	t.Run("retry resolved without header", func(t *testing.T) {
		t.Parallel()
		calls := 0

		// create a test server
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++

			if calls == 3 {
				w.WriteHeader(http.StatusOK)
				return
			}

			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer ts.Close()

		// create a client
		client, err := newClient("token", ts.URL, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
		require.NoError(t, err)

		res, err := client.ld.GetConfig().HTTPClient.Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, res.StatusCode, http.StatusOK)
		assert.Equal(t, 3, calls)
	})
}

func Test404RetryClient(t *testing.T) {
	t.Run("no retries needed", func(t *testing.T) {
		t.Parallel()
		calls := 0

		// create a test server
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		// create a client
		client, err := newClient("token", ts.URL, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
		require.NoError(t, err)

		res, err := client.ld404Retry.GetConfig().HTTPClient.Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, res.StatusCode, http.StatusOK)
		assert.Equal(t, calls, 1)
	})

	t.Run("max retries exceeded", func(t *testing.T) {
		t.Parallel()
		calls := 0

		// create a test server
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.WriteHeader(http.StatusNotFound)
		}))
		defer ts.Close()

		// create a client
		client, err := newClient("token", ts.URL, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
		require.NoError(t, err)

		res, err := client.ld404Retry.GetConfig().HTTPClient.Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, res.StatusCode, http.StatusNotFound)
		assert.Equal(t, calls, MAX_RETRIES+1)
	})

	t.Run("Resource found after a few retries", func(t *testing.T) {
		t.Parallel()
		calls := 0

		// create a test server
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++

			if calls == 3 {
				w.WriteHeader(http.StatusOK)
				return
			}

			w.WriteHeader(http.StatusNotFound)
		}))
		defer ts.Close()

		// create a client
		client, err := newClient("token", ts.URL, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
		require.NoError(t, err)

		res, err := client.ld404Retry.GetConfig().HTTPClient.Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, res.StatusCode, http.StatusOK)
		assert.Equal(t, 3, calls)
	})
}

func TestSemaphoreConcurrencyLimits(t *testing.T) {
	t.Parallel()

	// Track concurrent requests
	var concurrentRequests int32
	var maxConcurrentRequests int32

	// Create a test server that tracks concurrency
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Increment concurrent requests
		current := atomic.AddInt32(&concurrentRequests, 1)
		defer atomic.AddInt32(&concurrentRequests, -1)

		// Track max concurrent requests
		for {
			max := atomic.LoadInt32(&maxConcurrentRequests)
			if current <= max || atomic.CompareAndSwapInt32(&maxConcurrentRequests, max, current) {
				break
			}
		}

		// Simulate some work
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Create client with max concurrency of 3
	maxConcurrency := 2
	client, err := newClient("token", ts.URL, false, DEFAULT_HTTP_TIMEOUT_S, maxConcurrency)
	require.NoError(t, err)

	// Launch 10 simultaneous requests
	numRequests := 10
	var wg sync.WaitGroup
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := client.withConcurrency(client.ctx, func() error {
				_, err := client.ld.GetConfig().HTTPClient.Get(ts.URL)
				return err
			})
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check that no errors occurred
	for err := range errors {
		require.NoError(t, err)
	}

	// Verify that max concurrent requests never exceeded the semaphore limit
	maxConcurrent := atomic.LoadInt32(&maxConcurrentRequests)
	assert.LessOrEqual(t, maxConcurrent, int32(maxConcurrency),
		"Max concurrent requests (%d) exceeded semaphore limit (%d)", maxConcurrent, maxConcurrency)
}

func TestInjectArchivedDefault(t *testing.T) {
	t.Run("single view without archived gets default", func(t *testing.T) {
		view := map[string]interface{}{"key": "v1", "projectKey": "proj", "name": "View 1"}
		injectArchivedDefault(view)
		assert.Equal(t, false, view["archived"])
	})

	t.Run("existing archived is preserved", func(t *testing.T) {
		view := map[string]interface{}{"key": "v1", "projectKey": "proj", "archived": true}
		injectArchivedDefault(view)
		assert.Equal(t, true, view["archived"])
	})

	t.Run("list response injects into each view item", func(t *testing.T) {
		resp := map[string]interface{}{
			"totalCount": 2,
			"items": []interface{}{
				map[string]interface{}{"key": "v1", "projectKey": "proj"},
				map[string]interface{}{"key": "v2", "projectKey": "proj"},
			},
		}
		injectArchivedDefault(resp)
		items := resp["items"].([]interface{})
		for _, item := range items {
			assert.Equal(t, false, item.(map[string]interface{})["archived"])
		}
	})

	t.Run("non-view objects are left untouched", func(t *testing.T) {
		linked := map[string]interface{}{"resourceKey": "flag-1", "resourceType": "flags"}
		injectArchivedDefault(linked)
		_, hasArchived := linked["archived"]
		assert.False(t, hasArchived, "objects without projectKey must not gain an archived field")
	})
}

// TestViewArchivedShimTransport verifies that a view API response missing the
// `archived` field (as returned by LaunchDarkly after REL-14370) is rewritten
// to include `archived: false` so the strict generated model can deserialize
// it, while non-view responses are passed through untouched.
func TestViewArchivedShimTransport(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if strings.Contains(r.URL.Path, "/views") {
			// Mimic the post-REL-14370 API: no `archived` field.
			_, _ = w.Write([]byte(`{"key":"v1","projectKey":"proj","name":"View 1"}`))
			return
		}
		_, _ = w.Write([]byte(`{"key":"other"}`))
	}))
	t.Cleanup(ts.Close)

	client := &http.Client{Transport: &viewArchivedShimTransport{base: http.DefaultTransport}}

	t.Run("view response gains archived", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/api/v2/projects/proj/views/v1")
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		_ = resp.Body.Close()

		var decoded map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &decoded))
		assert.Equal(t, false, decoded["archived"])
	})

	t.Run("non-view response is untouched", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/api/v2/flags/proj/flag-1")
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		_ = resp.Body.Close()

		var decoded map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &decoded))
		_, hasArchived := decoded["archived"]
		assert.False(t, hasArchived)
	})
}
