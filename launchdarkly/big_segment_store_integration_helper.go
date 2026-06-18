package launchdarkly

import (
	"fmt"
	"strings"
)

// Big segment (persistent) store integration keys. These select the backing
// store technology and are the only values the API accepts for the
// `integration_key` path segment.
const (
	BIG_SEGMENT_STORE_INTEGRATION_KEY_DYNAMODB = "dynamodb"
	BIG_SEGMENT_STORE_INTEGRATION_KEY_REDIS    = "redis"
)

// newBigSegmentStoreIntegrationBetaClient returns a beta-configured client for
// the persistent-store-integration endpoints. The generated
// PersistentStoreIntegrationsBetaApi request builders do not expose a
// per-request .LDAPIVersion("beta") setter, so — as with MetricsBetaApi — we
// set the LD-API-Version header as a client default. The header is read from
// the configuration at request-build time, so mutating it here applies to every
// call made through the returned client.
func newBigSegmentStoreIntegrationBetaClient(c *Client) (*Client, error) {
	beta, err := newBetaClient(c.apiKey, c.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		return nil, err
	}
	beta.ld.GetConfig().AddDefaultHeader("LD-API-Version", "beta")
	beta.ld404Retry.GetConfig().AddDefaultHeader("LD-API-Version", "beta")
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
