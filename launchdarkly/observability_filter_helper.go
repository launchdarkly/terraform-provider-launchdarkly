package launchdarkly

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

// Observability filter settings are served by beta REST endpoints under
// /api/v2/projects/{projectKey}/observability/. They are not yet in the
// public OpenAPI spec, so api-client-go has no typed methods for them; the
// helpers below hand-roll the requests (the project_helper.go pattern) and
// reuse the client's retrying HTTP client. Swap to the generated client once
// the endpoints are public.

const observabilityAPIVersion = "beta"

// observabilityProductTypes is the REST/Terraform spelling of the products
// that support ingestion filter rules, in display order.
var observabilityProductTypes = []string{SESSIONS, ERRORS, LOGS, TRACES, METRICS}

var (
	observabilityOperations = []string{"include", "exclude"}
	// observabilityRuleKeyPattern mirrors the server-side rule key constraint.
	observabilityRuleKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,255}$`)
)

func observabilityRuleKeyValidator() regexValidator {
	return regexValidator{
		pattern: observabilityRuleKeyPattern,
		desc:    "Must be 1-256 characters of letters, numbers, '.', '-', or '_' and must start with an alphanumeric",
	}
}

// observabilityFilterRule is the REST FilterRule representation.
type observabilityFilterRule struct {
	Key         string          `json:"key"`
	Name        string          `json:"name"`
	ProductType string          `json:"productType"`
	Enabled     bool            `json:"enabled"`
	Query       string          `json:"query"`
	Operation   string          `json:"operation"`
	Rate        float64         `json:"rate"`
	Attributes  []string        `json:"attributes"`
	Version     int64           `json:"version"`
	CreatedAt   json.RawMessage `json:"createdAt,omitempty"`
	UpdatedAt   json.RawMessage `json:"updatedAt,omitempty"`
}

type observabilityFilterRulePost struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	ProductType string   `json:"productType"`
	Enabled     bool     `json:"enabled"`
	Query       string   `json:"query"`
	Operation   string   `json:"operation"`
	Rate        float64  `json:"rate"`
	Attributes  []string `json:"attributes"`
}

type observabilityFilterRulePut struct {
	Name       string   `json:"name"`
	Enabled    bool     `json:"enabled"`
	Query      string   `json:"query"`
	Operation  string   `json:"operation"`
	Rate       float64  `json:"rate"`
	Attributes []string `json:"attributes"`
	Version    *int64   `json:"version,omitempty"`
}

type observabilityFilterRuleCollection struct {
	Items []observabilityFilterRule `json:"items"`
}

// observabilityFilterSettings is the REST FilterSettings representation.
// A nil MinuteRateLimits entry means unlimited. On PUT, a product absent from
// RuleOrder leaves that product's order unchanged.
type observabilityFilterSettings struct {
	Version          *int64              `json:"version,omitempty"`
	MinuteRateLimits map[string]*int64   `json:"minuteRateLimits"`
	RuleOrder        map[string][]string `json:"ruleOrder"`
}

// observabilityAPIError carries the status and the LaunchDarkly error body
// ({"code": ..., "message": ...}) of a failed request.
type observabilityAPIError struct {
	StatusCode int
	Code       string
	Message    string
	Body       string
}

func (e *observabilityAPIError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = e.Body
	}
	if e.Code != "" {
		return fmt.Sprintf("%d %s (%s): %s", e.StatusCode, http.StatusText(e.StatusCode), e.Code, msg)
	}
	return fmt.Sprintf("%d %s: %s", e.StatusCode, http.StatusText(e.StatusCode), msg)
}

// REST error codes (gonfalon's {"code", "message"} body) for these endpoints.
const (
	// observabilityCodeVersionConflict: 409, the object's version moved on
	// since the caller read it.
	observabilityCodeVersionConflict = "optimistic_locking_error"
	// observabilityCodeKeyTaken: 409 on create, the rule key is in use.
	observabilityCodeKeyTaken = "conflict"
)

func (e *observabilityAPIError) isVersionConflict() bool {
	return e.StatusCode == http.StatusConflict && e.Code == observabilityCodeVersionConflict
}

func asObservabilityAPIError(err error) (*observabilityAPIError, bool) {
	var apiErr *observabilityAPIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}

func isObservabilityNotFound(err error) bool {
	apiErr, ok := asObservabilityAPIError(err)
	return ok && apiErr.StatusCode == http.StatusNotFound
}

// addObservabilityError adds a diagnostic for err, with an actionable detail
// for the errors a Terraform user can do something about:
//   - 409 optimistic_locking_error: someone else changed the object after
//     Terraform last read it.
//   - any other 409 (code "conflict"): on create, the rule key is taken;
//     conflictHint says how to resolve that.
//   - 401/403: the credential cannot manage observability settings.
func addObservabilityError(diags diagnosticsSink, summary string, err error, conflictHint string) {
	apiErr, ok := asObservabilityAPIError(err)
	if !ok {
		diags.AddError(summary, err.Error())
		return
	}
	switch {
	case apiErr.isVersionConflict():
		diags.AddError(summary, fmt.Sprintf(
			"The object was changed outside of Terraform (for example in the LaunchDarkly UI) after Terraform last read it, "+
				"so LaunchDarkly rejected the write as a version conflict. Run `terraform plan` to refresh and review the changes, then apply again.\n\n%s",
			err.Error()))
	case apiErr.StatusCode == http.StatusConflict && conflictHint != "":
		diags.AddError(summary, conflictHint+"\n\n"+err.Error())
	case apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden:
		diags.AddError(summary, "LaunchDarkly rejected the credential for observability filter settings. These beta endpoints accept a personal API access token "+
			"whose member can manage the project's observability settings, or an OAuth token with the `observability` scope. Service tokens are not supported.\n\n"+err.Error())
	default:
		diags.AddError(summary, err.Error())
	}
}

// observabilityWriteMu serialises writes per project. Creating or deleting a
// rule rewrites its product's rule order on the server, as does a settings
// PUT, so in-process writes for one project must not interleave.
var observabilityWriteMu sync.Map // project key -> *sync.Mutex

func lockObservabilityProject(projectKey string) func() {
	mu, _ := observabilityWriteMu.LoadOrStore(projectKey, &sync.Mutex{})
	m := mu.(*sync.Mutex)
	m.Lock()
	return m.Unlock
}

func observabilityURL(apiHost string, segments ...string) string {
	host := apiHost
	if host == "" {
		host = DEFAULT_LAUNCHDARKLY_HOST
	}
	base := "https://" + host
	if u, err := url.Parse(host); err == nil && u.Scheme != "" && u.Host != "" {
		base = u.Scheme + "://" + u.Host
	}
	escaped := make([]string, len(segments))
	for i, s := range segments {
		escaped[i] = url.PathEscape(s)
	}
	return base + "/api/v2/" + strings.Join(escaped, "/")
}

// doObservabilityRequest sends one beta request and decodes a 2xx JSON
// response into out (when non-nil). Non-2xx responses become
// *observabilityAPIError.
func doObservabilityRequest(ctx context.Context, client *Client, method, endpoint string, body, out interface{}) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", client.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("LD-API-Version", observabilityAPIVersion)
	req.Header.Set("User-Agent", fmt.Sprintf("launchdarkly-terraform-provider/%s", version))

	var resp *http.Response
	err = client.withConcurrency(ctx, func() error {
		var reqErr error
		resp, reqErr = client.ld.GetConfig().HTTPClient.Do(req)
		return reqErr
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		apiErr := &observabilityAPIError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(respBody))}
		var parsed struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if json.Unmarshal(respBody, &parsed) == nil {
			apiErr.Code = parsed.Code
			apiErr.Message = parsed.Message
		}
		return apiErr
	}
	if out != nil && len(bytes.TrimSpace(respBody)) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("failed to decode %s %s response: %w", method, endpoint, err)
		}
	}
	return nil
}

func observabilityRulesURL(client *Client, projectKey string, ruleKey ...string) string {
	segments := []string{"projects", projectKey, "observability", "filter-rules"}
	return observabilityURL(client.apiHost, append(segments, ruleKey...)...)
}

func observabilitySettingsURL(client *Client, projectKey string) string {
	return observabilityURL(client.apiHost, "projects", projectKey, "observability", "filter-settings")
}

func getObservabilityFilterRule(ctx context.Context, client *Client, projectKey, ruleKey string) (*observabilityFilterRule, error) {
	var rule observabilityFilterRule
	if err := doObservabilityRequest(ctx, client, http.MethodGet, observabilityRulesURL(client, projectKey, ruleKey), nil, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func listObservabilityFilterRules(ctx context.Context, client *Client, projectKey, productType string) ([]observabilityFilterRule, error) {
	endpoint := observabilityRulesURL(client, projectKey)
	if productType != "" {
		endpoint += "?productType=" + url.QueryEscape(productType)
	}
	var coll observabilityFilterRuleCollection
	if err := doObservabilityRequest(ctx, client, http.MethodGet, endpoint, nil, &coll); err != nil {
		return nil, err
	}
	return coll.Items, nil
}

func createObservabilityFilterRule(ctx context.Context, client *Client, projectKey string, post observabilityFilterRulePost) (*observabilityFilterRule, error) {
	unlock := lockObservabilityProject(projectKey)
	defer unlock()
	var rule observabilityFilterRule
	if err := doObservabilityRequest(ctx, client, http.MethodPost, observabilityRulesURL(client, projectKey), post, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func updateObservabilityFilterRule(ctx context.Context, client *Client, projectKey, ruleKey string, put observabilityFilterRulePut) (*observabilityFilterRule, error) {
	unlock := lockObservabilityProject(projectKey)
	defer unlock()
	var rule observabilityFilterRule
	if err := doObservabilityRequest(ctx, client, http.MethodPut, observabilityRulesURL(client, projectKey, ruleKey), put, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

func deleteObservabilityFilterRule(ctx context.Context, client *Client, projectKey, ruleKey string) error {
	unlock := lockObservabilityProject(projectKey)
	defer unlock()
	return doObservabilityRequest(ctx, client, http.MethodDelete, observabilityRulesURL(client, projectKey, ruleKey), nil, nil)
}

func getObservabilityFilterSettings(ctx context.Context, client *Client, projectKey string) (*observabilityFilterSettings, error) {
	var settings observabilityFilterSettings
	if err := doObservabilityRequest(ctx, client, http.MethodGet, observabilitySettingsURL(client, projectKey), nil, &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}

func putObservabilityFilterSettings(ctx context.Context, client *Client, projectKey string, settings observabilityFilterSettings) (*observabilityFilterSettings, error) {
	unlock := lockObservabilityProject(projectKey)
	defer unlock()
	var out observabilityFilterSettings
	if err := doObservabilityRequest(ctx, client, http.MethodPut, observabilitySettingsURL(client, projectKey), settings, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
