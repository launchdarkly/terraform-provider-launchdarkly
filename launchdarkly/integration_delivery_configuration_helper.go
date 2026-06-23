package launchdarkly

import (
	"fmt"
	"strings"
)

// newIntegrationDeliveryConfigurationBetaClient returns a beta-configured client
// suitable for the integration delivery configuration endpoints. Like the
// flag-import client, it forces the `LD-API-Version: beta` header at the
// transport layer via the shared forceBetaAPIVersion helper: the
// IntegrationDeliveryConfigurationsBetaApi request builders do not expose a
// per-request `.LDAPIVersion("beta")` setter, and a configured default header is
// appended after the generated client's stable version rather than replacing it
// (see betaVersionRoundTripper in flag_import_configuration_helper.go for the
// full rationale). Without the beta header these endpoints return 400/403/404.
func newIntegrationDeliveryConfigurationBetaClient(c *Client) (*Client, error) {
	beta, err := newBetaClient(c.apiKey, c.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		return nil, err
	}
	forceBetaAPIVersion(beta.ld)
	forceBetaAPIVersion(beta.ld404Retry)
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
