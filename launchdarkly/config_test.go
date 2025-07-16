package launchdarkly

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
