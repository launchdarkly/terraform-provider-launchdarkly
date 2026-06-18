package launchdarkly

import (
	"encoding/json"
	"fmt"
	"strings"
)

// newFlagImportConfigurationBetaClient returns a beta-configured client for the
// flag-import endpoints. Like the metric-groups builder, the generated
// FlagImportConfigurationsBetaApi request builders do not expose a per-request
// .LDAPIVersion("beta") setter, so we set the LD-API-Version header as a client
// default. Without the beta header these endpoints return 400/404.
func newFlagImportConfigurationBetaClient(c *Client) (*Client, error) {
	beta, err := newBetaClient(c.apiKey, c.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
	if err != nil {
		return nil, err
	}
	beta.ld.GetConfig().AddDefaultHeader("LD-API-Version", "beta")
	beta.ld404Retry.GetConfig().AddDefaultHeader("LD-API-Version", "beta")
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
