package launchdarkly

import (
	"net/http"
	"net/http/httptest"
	"strconv"
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
		client, err := newClient("token", ts.URL, false)
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
		client, err := newClient("token", ts.URL, false)
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
		client, err := newClient("token", ts.URL, false)
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
		client, err := newClient("token", ts.URL, false)
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
		client, err := newClient("token", ts.URL, false)
		require.NoError(t, err)

		res, err := client.ld.GetConfig().HTTPClient.Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, res.StatusCode, http.StatusOK)
		assert.Equal(t, 3, calls)
	})
}

func TestHandleConflicts(t *testing.T) {
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
		client, err := newClient("token", ts.URL, false)
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
			w.WriteHeader(http.StatusConflict)
		}))
		defer ts.Close()

		// create a client
		client, err := newClient("token", ts.URL, false)
		require.NoError(t, err)

		res, err := client.ld.GetConfig().HTTPClient.Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, res.StatusCode, http.StatusConflict)
		assert.Equal(t, calls, MAX_RETRIES+1)
	})

	t.Run("conflict resolved", func(t *testing.T) {
		t.Parallel()
		calls := 0

		// create a test server
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++

			if calls == 3 {
				w.WriteHeader(http.StatusOK)
				return
			}

			w.WriteHeader(http.StatusConflict)
		}))
		defer ts.Close()

		// create a client
		client, err := newClient("token", ts.URL, false)
		require.NoError(t, err)

		res, err := client.ld.GetConfig().HTTPClient.Get(ts.URL)
		require.NoError(t, err)
		assert.Equal(t, res.StatusCode, http.StatusOK)
		assert.Equal(t, 3, calls)
	})
}
