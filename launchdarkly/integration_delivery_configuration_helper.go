package launchdarkly

import (
	"fmt"
	"strings"
)

// newIntegrationDeliveryConfigurationBetaClient returns a beta-configured client
// suitable for the integration delivery configuration endpoints. Like the
// MetricsBetaApi builders, the IntegrationDeliveryConfigurationsBetaApi request
// builders do not expose a per-request .LDAPIVersion("beta") setter, so we set
// the LD-API-Version header as a client default instead. The header is read from
// the configuration at request-build time, so mutating it here takes effect for
// every call made through the returned client.
func newIntegrationDeliveryConfigurationBetaClient(c *Client) (*Client, error) {
	beta, err := newBetaClient(c.apiKey, c.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		return nil, err
	}
	beta.ld.GetConfig().AddDefaultHeader("LD-API-Version", "beta")
	beta.ld404Retry.GetConfig().AddDefaultHeader("LD-API-Version", "beta")
	return beta, nil
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
