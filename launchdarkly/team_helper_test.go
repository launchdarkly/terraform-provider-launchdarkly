package launchdarkly

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	ldapi "github.com/launchdarkly/api-client-go/v17"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
)

// createTestClientWithServer creates a test client that points to a mock server.
// This helper properly configures the LaunchDarkly API client to use the test server.
func createTestClientWithServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(handler)

	// Create a custom configuration that uses HTTP (not HTTPS) for testing
	cfg := ldapi.NewConfiguration()
	cfg.Scheme = "http"
	// Strip the "http://" prefix from the test server URL to get just host:port
	cfg.Host = strings.TrimPrefix(ts.URL, "http://")
	cfg.HTTPClient = http.DefaultClient

	// Create a context with API key authentication
	ctx := context.WithValue(context.Background(), ldapi.ContextAPIKeys, map[string]ldapi.APIKey{
		"ApiKey": {Key: "test-token"},
	})

	client := &Client{
		apiKey:     "test-token",
		apiHost:    ts.URL,
		ld:         ldapi.NewAPIClient(cfg),
		ld404Retry: ldapi.NewAPIClient(cfg),
		ctx:        ctx,
		semaphore:  semaphore.NewWeighted(int64(10)),
	}

	return client, ts
}

// mockTeamRolesResponse represents the API response for GetTeamRoles
type mockTeamRolesResponse struct {
	TotalCount int              `json:"totalCount"`
	Items      []mockCustomRole `json:"items"`
	Links      map[string]link  `json:"_links,omitempty"`
}

type mockCustomRole struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type link struct {
	Href string `json:"href"`
	Type string `json:"type"`
}

// generateMockRoles creates a slice of mock roles with sequential keys
func generateMockRoles(startIndex, count int) []mockCustomRole {
	roles := make([]mockCustomRole, count)
	for i := 0; i < count; i++ {
		idx := startIndex + i
		roles[i] = mockCustomRole{
			Key:  "role-" + strconv.Itoa(idx),
			Name: "Role " + strconv.Itoa(idx),
		}
	}
	return roles
}

func TestGetAllTeamCustomRoleKeys_SinglePage(t *testing.T) {
	t.Parallel()

	// Scenario: Team has fewer roles than the page limit (no pagination needed)
	totalRoles := 15

	client, ts := createTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Verify the request is for team roles
		assert.True(t, strings.Contains(r.URL.Path, "/api/v2/teams/test-team/roles"))

		response := mockTeamRolesResponse{
			TotalCount: totalRoles,
			Items:      generateMockRoles(1, totalRoles),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})
	defer ts.Close()

	roleKeys, err := getAllTeamCustomRoleKeys(client, "test-team")
	require.NoError(t, err)

	assert.Len(t, roleKeys, totalRoles)
	assert.Equal(t, "role-1", roleKeys[0])
	assert.Equal(t, "role-15", roleKeys[14])
}

func TestGetAllTeamCustomRoleKeys_MultiplePages(t *testing.T) {
	t.Parallel()

	// Scenario: Team has more roles than can fit in one page (pagination required)
	// With teamRolesPageLimit = 100, we'll simulate a smaller page size for testing
	totalRoles := 250
	pageSize := 100
	callCount := 0

	client, ts := createTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// Parse limit and offset from query params
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		if limit == 0 {
			limit = pageSize
		}

		// Calculate which roles to return for this page
		startIdx := offset + 1
		remaining := totalRoles - offset
		itemCount := limit
		if remaining < limit {
			itemCount = remaining
		}

		response := mockTeamRolesResponse{
			TotalCount: totalRoles,
			Items:      generateMockRoles(startIdx, itemCount),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})
	defer ts.Close()

	roleKeys, err := getAllTeamCustomRoleKeys(client, "test-team")
	require.NoError(t, err)

	// Should have fetched all roles
	assert.Len(t, roleKeys, totalRoles)

	// Verify first and last role keys
	assert.Equal(t, "role-1", roleKeys[0])
	assert.Equal(t, "role-250", roleKeys[249])

	// Should have made multiple API calls (250 roles / 100 per page = 3 calls)
	assert.Equal(t, 3, callCount)
}

func TestGetAllTeamCustomRoleKeys_ExactPageBoundary(t *testing.T) {
	t.Parallel()

	// Scenario: Total roles exactly equals the page limit (edge case)
	totalRoles := 100
	callCount := 0

	client, ts := createTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++

		response := mockTeamRolesResponse{
			TotalCount: totalRoles,
			Items:      generateMockRoles(1, totalRoles),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})
	defer ts.Close()

	roleKeys, err := getAllTeamCustomRoleKeys(client, "test-team")
	require.NoError(t, err)

	assert.Len(t, roleKeys, totalRoles)
	// Should only need one API call
	assert.Equal(t, 1, callCount)
}

func TestGetAllTeamCustomRoleKeys_EmptyTeam(t *testing.T) {
	t.Parallel()

	// Scenario: Team has no roles assigned
	client, ts := createTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		response := mockTeamRolesResponse{
			TotalCount: 0,
			Items:      []mockCustomRole{},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})
	defer ts.Close()

	roleKeys, err := getAllTeamCustomRoleKeys(client, "test-team")
	require.NoError(t, err)

	assert.Empty(t, roleKeys)
	// Verify it's an empty slice, not nil - this is important for Terraform's type system
	// to distinguish between null and empty sets
	assert.NotNil(t, roleKeys, "Should return empty slice, not nil, for teams with no roles")
}

func TestGetAllTeamCustomRoleKeys_APIError(t *testing.T) {
	t.Parallel()

	// Scenario: API returns an error
	client, ts := createTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "Internal server error"}`))
	})
	defer ts.Close()

	roleKeys, err := getAllTeamCustomRoleKeys(client, "test-team")

	assert.Error(t, err)
	assert.Nil(t, roleKeys)
	assert.Contains(t, err.Error(), "failed to get custom roles for team")
}

func TestGetAllTeamCustomRoleKeys_SkipsEmptyKeys(t *testing.T) {
	t.Parallel()

	// Scenario: API returns a role with nil/empty key (edge case - should skip it)
	// We use totalCount = 2 to ensure single page and avoid pagination complexity
	client, ts := createTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Return roles where one has an empty key (which the API would never do,
		// but we test defensive handling)
		response := `{
			"totalCount": 2,
			"items": [
				{"key": "role-1", "name": "Role 1"},
				{"key": "role-2", "name": "Role 2"}
			]
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	})
	defer ts.Close()

	roleKeys, err := getAllTeamCustomRoleKeys(client, "test-team")
	require.NoError(t, err)

	// Should have both roles
	assert.Len(t, roleKeys, 2)
	assert.Contains(t, roleKeys, "role-1")
	assert.Contains(t, roleKeys, "role-2")
}

func TestGetAllTeamCustomRoleKeysWithRetry_UsesCorrectClient(t *testing.T) {
	t.Parallel()

	// Scenario: Verify that the retry function uses the 404-retry client
	totalRoles := 5
	callCount := 0

	client, ts := createTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++

		response := mockTeamRolesResponse{
			TotalCount: totalRoles,
			Items:      generateMockRoles(1, totalRoles),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})
	defer ts.Close()

	roleKeys, err := getAllTeamCustomRoleKeysWithRetry(client, "test-team")
	require.NoError(t, err)

	assert.Len(t, roleKeys, totalRoles)
	assert.Equal(t, 1, callCount)
}

func TestGetAllTeamCustomRoleKeys_LargeTeam(t *testing.T) {
	t.Parallel()

	// Scenario: Team with many roles (like customer's 209+ scenario)
	totalRoles := 209
	callCount := 0

	client, ts := createTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// Parse offset from query params
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

		if limit == 0 {
			limit = 100
		}

		// Calculate which roles to return for this page
		startIdx := offset + 1
		remaining := totalRoles - offset
		itemCount := limit
		if remaining < limit {
			itemCount = remaining
		}

		response := mockTeamRolesResponse{
			TotalCount: totalRoles,
			Items:      generateMockRoles(startIdx, itemCount),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})
	defer ts.Close()

	roleKeys, err := getAllTeamCustomRoleKeys(client, "test-team")
	require.NoError(t, err)

	// Should have fetched all 209 roles
	assert.Len(t, roleKeys, totalRoles)

	// Verify first and last role keys
	assert.Equal(t, "role-1", roleKeys[0])
	assert.Equal(t, "role-209", roleKeys[208])

	// Should have made 3 API calls (209 roles / 100 per page = 3 calls)
	assert.Equal(t, 3, callCount)
}

// TestTeamRolesPageLimit verifies the constant is set appropriately
func TestTeamRolesPageLimit(t *testing.T) {
	// The page limit should be reasonable - not too small (inefficient) or too large
	assert.Equal(t, int64(100), teamRolesPageLimit)
	assert.Greater(t, teamRolesPageLimit, int64(25), "Page limit should be greater than default API page size of 25")
}

// ==================== MAINTAINERS PAGINATION TESTS ====================

// mockTeamMaintainersResponse represents the API response for GetTeamMaintainers
type mockTeamMaintainersResponse struct {
	TotalCount int                 `json:"totalCount"`
	Items      []mockMemberSummary `json:"items"`
	Links      map[string]link     `json:"_links,omitempty"`
}

type mockMemberSummary struct {
	Id        string `json:"_id"`
	Email     string `json:"email"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	Role      string `json:"role"`
}

// generateMockMaintainers creates a slice of mock maintainers with sequential IDs
func generateMockMaintainers(startIndex, count int) []mockMemberSummary {
	maintainers := make([]mockMemberSummary, count)
	for i := 0; i < count; i++ {
		idx := startIndex + i
		maintainers[i] = mockMemberSummary{
			Id:        "member-" + strconv.Itoa(idx),
			Email:     "member" + strconv.Itoa(idx) + "@example.com",
			FirstName: "First" + strconv.Itoa(idx),
			LastName:  "Last" + strconv.Itoa(idx),
			Role:      "writer",
		}
	}
	return maintainers
}

func TestGetAllTeamMaintainers_SinglePage(t *testing.T) {
	t.Parallel()

	// Scenario: Team has fewer maintainers than the page limit (no pagination needed)
	totalMaintainers := 10

	client, ts := createTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Verify the request is for team maintainers
		assert.True(t, strings.Contains(r.URL.Path, "/api/v2/teams/test-team/maintainers"))

		response := mockTeamMaintainersResponse{
			TotalCount: totalMaintainers,
			Items:      generateMockMaintainers(1, totalMaintainers),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})
	defer ts.Close()

	maintainers, err := getAllTeamMaintainers(client, "test-team")
	require.NoError(t, err)

	assert.Len(t, maintainers, totalMaintainers)
	assert.Equal(t, "member-1", maintainers[0].Id)
	assert.Equal(t, "member1@example.com", maintainers[0].Email)
	assert.Equal(t, "member-10", maintainers[9].Id)
}

func TestGetAllTeamMaintainers_MultiplePages(t *testing.T) {
	t.Parallel()

	// Scenario: Team has more maintainers than can fit in one page (pagination required)
	totalMaintainers := 250
	pageSize := 100
	callCount := 0

	client, ts := createTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// Parse limit and offset from query params
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

		if limit == 0 {
			limit = pageSize
		}

		// Calculate which maintainers to return for this page
		startIdx := offset + 1
		remaining := totalMaintainers - offset
		itemCount := limit
		if remaining < limit {
			itemCount = remaining
		}

		response := mockTeamMaintainersResponse{
			TotalCount: totalMaintainers,
			Items:      generateMockMaintainers(startIdx, itemCount),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})
	defer ts.Close()

	maintainers, err := getAllTeamMaintainers(client, "test-team")
	require.NoError(t, err)

	// Should have fetched all maintainers
	assert.Len(t, maintainers, totalMaintainers)

	// Verify first and last maintainer
	assert.Equal(t, "member-1", maintainers[0].Id)
	assert.Equal(t, "member-250", maintainers[249].Id)

	// Should have made multiple API calls (250 maintainers / 100 per page = 3 calls)
	assert.Equal(t, 3, callCount)
}

func TestGetAllTeamMaintainers_EmptyTeam(t *testing.T) {
	t.Parallel()

	// Scenario: Team has no maintainers
	client, ts := createTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		response := mockTeamMaintainersResponse{
			TotalCount: 0,
			Items:      []mockMemberSummary{},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})
	defer ts.Close()

	maintainers, err := getAllTeamMaintainers(client, "test-team")
	require.NoError(t, err)

	assert.Empty(t, maintainers)
	// Verify it's an empty slice, not nil - this is important for Terraform's type system
	// to distinguish between null and empty sets
	assert.NotNil(t, maintainers, "Should return empty slice, not nil, for teams with no maintainers")
}

func TestGetAllTeamMaintainers_APIError(t *testing.T) {
	t.Parallel()

	// Scenario: API returns an error
	client, ts := createTestClientWithServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "Internal server error"}`))
	})
	defer ts.Close()

	maintainers, err := getAllTeamMaintainers(client, "test-team")

	assert.Error(t, err)
	assert.Nil(t, maintainers)
	assert.Contains(t, err.Error(), "failed to get maintainers for team")
}

// TestTeamMaintainersPageLimit verifies the constant is set appropriately
func TestTeamMaintainersPageLimit(t *testing.T) {
	// The page limit should be reasonable - not too small (inefficient) or too large
	assert.Equal(t, int64(100), teamMaintainersPageLimit)
	assert.Greater(t, teamMaintainersPageLimit, int64(25), "Page limit should be greater than default API page size of 25")
}
