package launchdarkly

import (
	"fmt"
	"net/http"
	"strings"
)

// betaAPIVersionRoundTripper forces the LD-API-Version header to "beta" on every
// outgoing request. It exists because the IntegrationDeliveryConfigurationsBetaApi
// request builders do not expose a per-request .LDAPIVersion("beta") setter, and
// the vendored client's prepareRequest unconditionally inserts the stable
// LD-API-Version ("20240415") whenever the per-request header params omit it (see
// the CUSTOM block in api-client-go/v22 client.go). A client default header set via
// AddDefaultHeader is *appended* after that stable value (Header.Add, not Set), so
// the request would carry LD-API-Version: ["20240415", "beta"] and the API reads
// the first value, rejecting the call as non-beta. Setting the header here with
// http.Header.Set collapses the pair down to the single "beta" value the endpoint
// requires.
type betaAPIVersionRoundTripper struct {
	next http.RoundTripper
}

func (t betaAPIVersionRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("LD-API-Version", "beta")
	return t.next.RoundTrip(req)
}

// newIntegrationDeliveryConfigurationBetaClient returns a beta-configured client
// suitable for the integration delivery configuration endpoints, wrapping each
// underlying HTTP client so that every request carries LD-API-Version: beta.
func newIntegrationDeliveryConfigurationBetaClient(c *Client) (*Client, error) {
	beta, err := newBetaClient(c.apiKey, c.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		return nil, err
	}
	forceBetaAPIVersion(beta.ld.GetConfig().HTTPClient)
	forceBetaAPIVersion(beta.ld404Retry.GetConfig().HTTPClient)
	return beta, nil
}

// forceBetaAPIVersion wraps the transport of the given HTTP client so that it
// always sends LD-API-Version: beta.
func forceBetaAPIVersion(httpClient *http.Client) {
	if httpClient == nil {
		return
	}
	base := httpClient.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	httpClient.Transport = betaAPIVersionRoundTripper{next: base}
}

// integrationDeliveryConfigurationID builds the composite Terraform ID for a
// delivery configuration from its parent keys and the server-assigned config ID.
// Format: project_key/env_key/integration_key/config_id.
func integrationDeliveryConfigurationID(projectKey, envKey, integrationKey, configID string) string {
	return strings.Join([]string{projectKey, envKey, integrationKey, configID}, "/")
}

// integrationDeliveryConfigurationIDToKeys splits a composite delivery
// configuration ID into its components. The expected format is
// project_key/env_key/integration_key/config_id.
func integrationDeliveryConfigurationIDToKeys(id string) (projectKey, envKey, integrationKey, configID string, err error) {
	if strings.Count(id, "/") != 3 {
		return "", "", "", "", fmt.Errorf("found unexpected integration delivery configuration id format: %q expected format: 'project_key/env_key/integration_key/config_id'", id)
	}
	parts := strings.SplitN(id, "/", 4)
	return parts[0], parts[1], parts[2], parts[3], nil
}
