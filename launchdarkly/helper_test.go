package launchdarkly

import (
	"errors"
	"net/http"
	"testing"

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
