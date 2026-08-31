package launchdarkly

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Rate limiting is applied per route, so how exposed this resource is to a 429
// is decided by how many requests each operation makes. These tests pin that
// number down: they stand up a fake LaunchDarkly, run an operation against it,
// and assert the exact request count. A change that reintroduces per-member
// fan-out fails here rather than in a customer's onboarding.

// fakeMember returns a member object with every field the generated client
// requires, so responses unmarshal the same way production ones do.
func fakeMember(id, email string) map[string]interface{} {
	return map[string]interface{}{
		"_links":         map[string]interface{}{},
		"_id":            id,
		"role":           "reader",
		"email":          email,
		"_pendingInvite": false,
		"_verified":      true,
		"customRoles":    []string{},
		"mfa":            "disabled",
		"_lastSeen":      0,
		"creationDate":   0,
	}
}

// fakeMembers wraps members in the collection envelope the client requires.
func fakeMembers(items []map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"_links":     map[string]interface{}{},
		"items":      items,
		"totalCount": len(items),
	}
}

// writeJSON responds with the content type the generated client requires: it
// only decodes application/json and reports "undefined response type" otherwise.
func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}

// ctxBackground and nowMillis keep the test bodies terse.
func ctxBackground() context.Context { return context.Background() }

func nowMillis() int64 { return time.Now().UnixNano() / int64(time.Millisecond) }

// apiCallRecorder counts requests by "METHOD /path".
type apiCallRecorder struct {
	mu     sync.Mutex
	calls  []string
	server *httptest.Server
}

func (r *apiCallRecorder) record(method, path string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, method+" "+path)
}

func (r *apiCallRecorder) countMatching(substr string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, c := range r.calls {
		if strings.Contains(c, substr) {
			n++
		}
	}
	return n
}

func (r *apiCallRecorder) total() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

func (r *apiCallRecorder) all() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.calls))
	copy(out, r.calls)
	return out
}

// newFakeLD serves the handful of member and team endpoints this resource uses,
// recording every request. memberIDs assigns a deterministic ID per email.
func newFakeLD(t *testing.T) (*apiCallRecorder, *Client) {
	t.Helper()
	rec := &apiCallRecorder{}
	nextID := 0

	mux := http.NewServeMux()

	// POST /api/v2/members (batch create) and GET /api/v2/members (filtered read).
	mux.HandleFunc("/api/v2/members", func(w http.ResponseWriter, r *http.Request) {
		rec.record(r.Method, "/api/v2/members")
		switch r.Method {
		case http.MethodPost:
			var forms []map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&forms)
			items := make([]map[string]interface{}, 0, len(forms))
			for _, f := range forms {
				nextID++
				items = append(items, fakeMember(fmt.Sprintf("id%04d", nextID), fmt.Sprint(f["email"])))
			}
			writeJSON(w, http.StatusCreated, fakeMembers(items))
		case http.MethodGet:
			// Return one member per id/email in the filter so bulk reads resolve.
			filter := r.URL.Query().Get("filter")
			items := []map[string]interface{}{}
			if idx := strings.Index(filter, ":"); idx >= 0 {
				kind, list := filter[:idx], filter[idx+1:]
				for i, v := range strings.Split(list, "|") {
					if v == "" {
						continue
					}
					id, email := v, fmt.Sprintf("m%d@example.com", i)
					if kind == "email" {
						id, email = fmt.Sprintf("id%04d", i+1), v
					}
					items = append(items, fakeMember(id, email))
				}
			}
			writeJSON(w, http.StatusOK, fakeMembers(items))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// PATCH/DELETE /api/v2/members/{id}
	mux.HandleFunc("/api/v2/members/", func(w http.ResponseWriter, r *http.Request) {
		rec.record(r.Method, "/api/v2/members/{id}")
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/api/v2/members/")
		writeJSON(w, http.StatusOK, fakeMember(id, "m@example.com"))
	})

	// PATCH /api/v2/teams/{key}
	mux.HandleFunc("/api/v2/teams/", func(w http.ResponseWriter, r *http.Request) {
		rec.record(r.Method, "/api/v2/teams/{key}")
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"_links": map[string]interface{}{}, "key": "t", "name": "t",
		})
	})

	rec.server = httptest.NewServer(mux)
	t.Cleanup(rec.server.Close)

	client, err := newClient("token", rec.server.URL, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	return rec, client
}

// batchOf builds n entries sharing one role, the shape a bulk onboarding uses.
func batchOf(n int, role string, teamKeys ...string) map[string]teamMembersEntryModel {
	out := make(map[string]teamMembersEntryModel, n)
	for i := 0; i < n; i++ {
		e := entry(role)
		if len(teamKeys) > 0 {
			set, _ := types.SetValueFrom(ctxBackground(), types.StringType, teamKeys)
			e.TeamKeys = set
		}
		out[fmt.Sprintf("u%02d@example.com", i)] = e
	}
	return out
}

func TestAPICallCount_CreateIsOneRequest(t *testing.T) {
	for _, size := range []int{1, 10, 50} {
		t.Run(fmt.Sprintf("%d_members", size), func(t *testing.T) {
			rec, client := newFakeLD(t)
			r := &TeamMembersResource{client: client}

			resolved, adopted, diags := r.createMemberBatch(ctxBackground(), batchOf(size, "reader"), false)
			require.False(t, diags.HasError(), "diags: %v", diags)
			assert.Len(t, resolved, size)
			assert.Empty(t, adopted)

			posts := rec.countMatching("POST /api/v2/members")
			assert.Equal(t, 1, posts,
				"a batch of %d must be one POST, not one per member; calls=%v", size, rec.all())
		})
	}
}

func TestAPICallCount_ReadIsOneRequest(t *testing.T) {
	rec, client := newFakeLD(t)
	r := &TeamMembersResource{client: client}

	ids := make([]string, 0, 50)
	for i := 1; i <= 50; i++ {
		ids = append(ids, fmt.Sprintf("id%04d", i))
	}
	_, err := r.fetchMembersByID(ids)
	require.NoError(t, err)

	gets := rec.countMatching("GET /api/v2/members")
	perMember := rec.countMatching("/api/v2/members/{id}")
	assert.Equal(t, 1, gets, "reading 50 members must be one filtered GET; calls=%v", rec.all())
	assert.Zero(t, perMember, "read must not fall back to per-member GETs")
}

func TestAPICallCount_DeleteIsPerMember(t *testing.T) {
	// There is no bulk delete endpoint, so this documents the known ceiling:
	// removals cost one request each. Offboarding trickles in practice, and the
	// retry layer backs off on 429 using the server's reset header, but a very
	// large single-apply removal is the one operation that still fans out.
	rec, client := newFakeLD(t)
	r := &TeamMembersResource{client: client}

	ids := make([]string, 0, 10)
	for i := 1; i <= 10; i++ {
		ids = append(ids, fmt.Sprintf("id%04d", i))
	}
	diags := r.deleteMembersByID(ids)
	require.False(t, diags.HasError())

	assert.Equal(t, 10, rec.countMatching("DELETE /api/v2/members/{id}"),
		"expected one delete per member; calls=%v", rec.all())
}

func TestAPICallCount_TeamAssignmentIsGroupedPerTeam(t *testing.T) {
	// 20 members joining the same two teams must cost two team requests, not
	// forty: team membership is grouped by team key.
	rec, client := newFakeLD(t)
	r := &TeamMembersResource{client: client}

	deltas := map[string]*teamMembershipDelta{}
	for i := 0; i < 20; i++ {
		id := fmt.Sprintf("id%04d", i)
		deltas["team-a"] = appendAdd(deltas["team-a"], id)
		deltas["team-b"] = appendAdd(deltas["team-b"], id)
	}
	diags := r.applyTeamMembershipDeltas(deltas)
	require.False(t, diags.HasError())

	assert.Equal(t, 2, rec.countMatching("PATCH /api/v2/teams/{key}"),
		"expected one request per team; calls=%v", rec.all())
}

func appendAdd(d *teamMembershipDelta, id string) *teamMembershipDelta {
	if d == nil {
		d = &teamMembershipDelta{}
	}
	d.add = append(d.add, id)
	return d
}

func TestAPICallCount_AdoptionAddsOneLookup(t *testing.T) {
	// A conflicting create costs: the failed POST, one bulk email lookup, and
	// the retried POST. It must not look members up one at a time.
	rec := &apiCallRecorder{}
	posts := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/members", func(w http.ResponseWriter, r *http.Request) {
		rec.record(r.Method, "/api/v2/members")
		if r.Method == http.MethodGet {
			filter := r.URL.Query().Get("filter")
			items := []map[string]interface{}{}
			for i, v := range strings.Split(strings.TrimPrefix(filter, "email:"), "|") {
				items = append(items, fakeMember(fmt.Sprintf("existing%d", i), v))
			}
			writeJSON(w, http.StatusOK, fakeMembers(items))
			return
		}
		posts++
		if posts == 1 {
			// First attempt: three of the emails already exist.
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{
				"code":    conflictEmailExistsInAccount,
				"message": "some emails already exist",
				"invalid_emails": []string{
					"u00@example.com", "u01@example.com", "u02@example.com",
				},
			})
			return
		}
		var forms []map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&forms)
		items := make([]map[string]interface{}, 0, len(forms))
		for i, f := range forms {
			items = append(items, fakeMember(fmt.Sprintf("new%d", i), fmt.Sprint(f["email"])))
		}
		writeJSON(w, http.StatusCreated, fakeMembers(items))
	})
	rec.server = httptest.NewServer(mux)
	t.Cleanup(rec.server.Close)
	client, err := newClient("token", rec.server.URL, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	r := &TeamMembersResource{client: client}

	resolved, adopted, diags := r.createMemberBatch(ctxBackground(), batchOf(10, "reader"), true)
	require.False(t, diags.HasError(), "diags: %v", diags)
	assert.Len(t, resolved, 10)
	assert.Len(t, adopted, 3)

	assert.Equal(t, 2, rec.countMatching("POST /api/v2/members"), "one failed POST plus one retry")
	assert.Equal(t, 1, rec.countMatching("GET /api/v2/members"), "adoption must use a single bulk email lookup")
	assert.Equal(t, 3, rec.total(), "adoption of any batch size costs three requests; calls=%v", rec.all())
}

func TestAPICallCount_RateLimitedRequestIsRetriedNotFailed(t *testing.T) {
	// The safety net when a 429 does happen: the shared retry client honors the
	// reset header rather than surfacing the error, so a rate-limited batch is
	// delayed instead of failing the apply.
	rec := &apiCallRecorder{}
	attempts := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/members", func(w http.ResponseWriter, r *http.Request) {
		rec.record(r.Method, "/api/v2/members")
		attempts++
		if attempts == 1 {
			// Reset 50ms out, so the retry is prompt but the header path is used.
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", nowMillis()+50))
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		writeJSON(w, http.StatusCreated, fakeMembers([]map[string]interface{}{
			fakeMember("id0001", "u00@example.com"),
		}))
	})
	rec.server = httptest.NewServer(mux)
	t.Cleanup(rec.server.Close)
	client, err := newClient("token", rec.server.URL, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	require.NoError(t, err)
	r := &TeamMembersResource{client: client}

	resolved, _, diags := r.createMemberBatch(ctxBackground(), batchOf(1, "reader"), false)
	assert.False(t, diags.HasError(), "a 429 must be retried, not surfaced: %v", diags)
	assert.Len(t, resolved, 1)
	assert.GreaterOrEqual(t, rec.countMatching("POST /api/v2/members"), 2, "expected at least one retry")
}

func TestAPICallCount_AttributeChangesArePerMember(t *testing.T) {
	// Documents the other known ceiling. Changing roles or custom roles costs
	// one PATCH per member, because the bulk member-patch endpoint is an
	// Enterprise-only feature and using it would need a per-member fallback for
	// everyone else. Onboarding, the operation that motivated this resource, is
	// unaffected: creation is a single request regardless of batch size.
	rec, client := newFakeLD(t)
	r := &TeamMembersResource{client: client}

	state := map[string]teamMembersEntryModel{}
	changed := map[string]teamMembersEntryModel{}
	for i := 0; i < 10; i++ {
		email := fmt.Sprintf("u%02d@example.com", i)
		prior := entryWithID("reader", fmt.Sprintf("id%04d", i+1))
		desired := entryWithID("writer", fmt.Sprintf("id%04d", i+1))
		state[email] = prior
		changed[email] = desired
	}

	diags := r.patchChangedMembers(ctxBackground(), changed, state)
	require.False(t, diags.HasError(), "diags: %v", diags)

	assert.Equal(t, 10, rec.countMatching("PATCH /api/v2/members/{id}"),
		"role changes cost one request per member; calls=%v", rec.all())
	assert.Zero(t, rec.countMatching("PATCH /api/v2/teams/{key}"),
		"an attribute-only change must not touch teams")
}
