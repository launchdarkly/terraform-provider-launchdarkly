package launchdarkly

import (
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleNoConflict(t *testing.T) {
	t.Run("no retries needed", func(t *testing.T) {
		t.Parallel()
		calls := 0
		_, res, err := handleNoConflict(func() (interface{}, *http.Response, error) {
			calls++
			return nil, &http.Response{StatusCode: http.StatusOK}, nil
		})
		require.NoError(t, err)
		assert.Equal(t, 1, calls)
		assert.Equal(t, res.StatusCode, http.StatusOK)
	})
	t.Run("max retries exceeded", func(t *testing.T) {
		t.Parallel()
		calls := 0
		_, res, err := handleNoConflict(func() (interface{}, *http.Response, error) {
			calls++
			return nil, &http.Response{StatusCode: http.StatusConflict}, errors.New("Conflict")
		})
		require.Error(t, err)
		assert.Equal(t, 6, calls)
		assert.Equal(t, res.StatusCode, http.StatusConflict)
	})
	t.Run("conflict resolved", func(t *testing.T) {
		t.Parallel()
		calls := 0
		_, res, err := handleNoConflict(func() (interface{}, *http.Response, error) {
			calls++
			if calls == 3 {
				return nil, &http.Response{StatusCode: http.StatusOK}, nil
			}
			return nil, &http.Response{StatusCode: http.StatusConflict}, nil
		})
		require.NoError(t, err)
		assert.Equal(t, 3, calls)
		assert.Equal(t, res.StatusCode, http.StatusOK)
	})
}

func TestHandleRateLimit(t *testing.T) {
	t.Run("no retries needed", func(t *testing.T) {
		t.Parallel()
		calls := 0
		_, res, err := handleRateLimit(func() (interface{}, *http.Response, error) {
			calls++
			return nil, &http.Response{StatusCode: http.StatusOK}, nil
		})
		require.NoError(t, err)
		assert.Equal(t, 1, calls)
		assert.Equal(t, res.StatusCode, http.StatusOK)
	})
	t.Run("max retries exceeded", func(t *testing.T) {
		t.Parallel()
		calls := 0
		_, res, err := handleRateLimit(func() (interface{}, *http.Response, error) {
			calls++
			res := &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{}}
			res.Header.Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(100*time.Millisecond).UnixNano()/int64(time.Millisecond), 10))
			return nil, res, errors.New("Rate limit exceeded")
		})
		require.Error(t, err)
		assert.Equal(t, MAX_429_RETRIES+1, calls)
		assert.Equal(t, res.StatusCode, http.StatusTooManyRequests)
	})
	t.Run("retry resolved with header", func(t *testing.T) {
		t.Parallel()
		calls := 0
		_, res, err := handleRateLimit(func() (interface{}, *http.Response, error) {
			calls++
			if calls == 3 {
				return nil, &http.Response{StatusCode: http.StatusOK}, nil
			}
			res := &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{}}
			res.Header.Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(100*time.Millisecond).UnixNano()/int64(time.Millisecond), 10))
			return nil, res, errors.New("Rate limit exceeded")
		})
		require.NoError(t, err)
		assert.Equal(t, 3, calls)
		assert.Equal(t, res.StatusCode, http.StatusOK)
	})
	t.Run("retry resolved with negative header", func(t *testing.T) {
		t.Parallel()
		calls := 0
		_, res, err := handleRateLimit(func() (interface{}, *http.Response, error) {
			calls++
			if calls == 3 {
				return nil, &http.Response{StatusCode: http.StatusOK}, nil
			}
			res := &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{}}
			res.Header.Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(-100*time.Millisecond).UnixNano()/int64(time.Millisecond), 10))
			return nil, res, errors.New("Rate limit exceeded")
		})
		require.NoError(t, err)
		assert.Equal(t, 3, calls)
		assert.Equal(t, res.StatusCode, http.StatusOK)
	})
	t.Run("retry resolved without header", func(t *testing.T) {
		t.Parallel()
		calls := 0
		_, res, err := handleRateLimit(func() (interface{}, *http.Response, error) {
			calls++
			if calls == 3 {
				return nil, &http.Response{StatusCode: http.StatusOK}, nil
			}
			res := &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{}}
			return nil, res, errors.New("Rate limit exceeded")
		})
		require.NoError(t, err)
		assert.Equal(t, 3, calls)
		assert.Equal(t, res.StatusCode, http.StatusOK)
	})
}
