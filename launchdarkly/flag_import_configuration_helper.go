package launchdarkly

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// betaVersionRoundTripper forces every request through it to carry exactly
// `LD-API-Version: beta`. This is required because the generated client's
// prepareRequest hardcodes `LD-API-Version: 20240415` when an operation does not
// set the header itself, and then *appends* (http.Header.Add) any configured
// default header rather than replacing it. The FlagImportConfigurationsBetaApi
// request builders do not expose a per-request `.LDAPIVersion("beta")` setter,
// so a plain `AddDefaultHeader("LD-API-Version", "beta")` would leave the
// request carrying `["20240415", "beta"]` — and the server reads the first
// value, `20240415`, rejecting the beta-only endpoint. Setting the header at the
// transport layer (http.Header.Set) replaces both values with `beta`.
type betaVersionRoundTripper struct {
	inner http.RoundTripper
}

func (t betaVersionRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("LD-API-Version", "beta")
	return t.inner.RoundTrip(req)
}

func forceBetaAPIVersion(client *ldapi.APIClient) {
	hc := client.GetConfig().HTTPClient
	inner := hc.Transport
	if inner == nil {
		inner = http.DefaultTransport
	}
	hc.Transport = betaVersionRoundTripper{inner: inner}
}

// newFlagImportConfigurationBetaClient returns a beta-configured client for the
// flag-import endpoints. The generated FlagImportConfigurationsBetaApi request
// builders do not expose a per-request `.LDAPIVersion("beta")` setter, so we
// force the `LD-API-Version: beta` header at the transport layer (see
// betaVersionRoundTripper for why a default header is insufficient). Without the
// beta header these endpoints return 400/403/404.
func newFlagImportConfigurationBetaClient(c *Client) (*Client, error) {
	beta, err := newBetaClient(c.apiKey, c.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		return nil, err
	}
	forceBetaAPIVersion(beta.ld)
	forceBetaAPIVersion(beta.ld404Retry)
	return beta, nil
}

// flagImportConfigurationIdToKeys splits a composite flag import configuration
// ID into its parts. The expected format is
// `project_key/integration_key/integration_id`.
func flagImportConfigurationIdToKeys(id string) (projectKey, integrationKey, integrationID string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("found unexpected flag import configuration id format: %q expected format: 'project_key/integration_key/integration_id'", id)
	}
	return parts[0], parts[1], parts[2], nil
}

// flagImportConfigurationID joins the parts of a flag import configuration's
// composite ID.
func flagImportConfigurationID(projectKey, integrationKey, integrationID string) string {
	return strings.Join([]string{projectKey, integrationKey, integrationID}, "/")
}

// configMapFromJSON parses a JSON object string into the map[string]interface{}
// shape the flag-import API's `config` field expects.
func configMapFromJSON(s string) (map[string]interface{}, error) {
	if s == "" {
		return map[string]interface{}{}, nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, fmt.Errorf("config must be a JSON object: %w", err)
	}
	return out, nil
}

// configJSONFromMap marshals the API's `config` map back into a compact JSON
// string for storage in Terraform state. Key ordering diffs are suppressed by
// jsonNormalizePlanModifier on the schema attribute.
func configJSONFromMap(m map[string]interface{}) (string, error) {
	if len(m) == 0 {
		return "{}", nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("failed to encode config: %w", err)
	}
	return string(b), nil
}
