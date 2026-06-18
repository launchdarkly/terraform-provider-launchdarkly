package launchdarkly

import (
	"fmt"
	"net/http"
	"strings"

	ldapi "github.com/launchdarkly/api-client-go/v22"
)

// Big segment (persistent) store integration keys. These select the backing
// store technology and are the only values the API accepts for the
// `integration_key` path segment.
const (
	BIG_SEGMENT_STORE_INTEGRATION_KEY_DYNAMODB = "dynamodb"
	BIG_SEGMENT_STORE_INTEGRATION_KEY_REDIS    = "redis"
)

// betaAPIVersionRoundTripper forces LD-API-Version: beta on every outgoing
// request. A client-default header is insufficient here: the generated client's
// prepareRequest unconditionally injects the stable "20240415" version when the
// header is not already present in the per-request header params, and only
// *then* appends configured default headers. That leaves two LD-API-Version
// values on the wire ("20240415", "beta") and LaunchDarkly honours the first,
// rejecting the call as a non-beta request. Setting the header in a round
// tripper (which replaces rather than appends) is the reliable fix for beta
// endpoints — like PersistentStoreIntegrationsBetaApi — whose generated request
// builders do not expose a per-request .LDAPIVersion("beta") setter.
type betaAPIVersionRoundTripper struct {
	wrapped http.RoundTripper
}

func (t betaAPIVersionRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Set("LD-API-Version", "beta")
	return t.wrapped.RoundTrip(clone)
}

// newBigSegmentStoreIntegrationBetaClient returns a beta-configured client for
// the persistent-store-integration endpoints, with the LD-API-Version: beta
// header enforced via a round tripper (see betaAPIVersionRoundTripper for why a
// client-default header does not work for this generated client).
func newBigSegmentStoreIntegrationBetaClient(c *Client) (*Client, error) {
	beta, err := newBetaClient(c.apiKey, c.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		return nil, err
	}
	for _, api := range []*ldapi.APIClient{beta.ld, beta.ld404Retry} {
		hc := api.GetConfig().HTTPClient
		hc.Transport = betaAPIVersionRoundTripper{wrapped: hc.Transport}
	}
	return beta, nil
}

// bigSegmentStoreIntegrationID builds the composite Terraform ID from the four
// identifiers the API uses to address a single integration.
func bigSegmentStoreIntegrationID(projectKey, environmentKey, integrationKey, integrationID string) string {
	return strings.Join([]string{projectKey, environmentKey, integrationKey, integrationID}, "/")
}

// bigSegmentStoreIntegrationIDToKeys splits a composite integration ID into its
// parts. The expected format is
// `project_key/environment_key/integration_key/integration_id`.
func bigSegmentStoreIntegrationIDToKeys(id string) (projectKey, environmentKey, integrationKey, integrationID string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 4 {
		return "", "", "", "", fmt.Errorf("found unexpected big segment store integration id format: %q expected format: 'project_key/environment_key/integration_key/integration_id'", id)
	}
	return parts[0], parts[1], parts[2], parts[3], nil
}
