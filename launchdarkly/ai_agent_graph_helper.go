package launchdarkly

import (
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v23"
)

// agentGraphBetaVersion is the LD-API-Version the agent graph endpoints
// require. The generated request builders expose a per-request
// .LDAPIVersion(...) setter, and the server rejects the call with
// "lDAPIVersion is required and must be specified" if it is omitted.
const agentGraphBetaVersion = "beta"

// newAIAgentGraphBetaClient returns a beta-configured client for the agent
// graph endpoints. They live on the beta API surface, so we use a beta client
// (which does not set a default LD-API-Version header) and pair it with the
// per-request .LDAPIVersion("beta"); using the standard client would send the
// header twice ("Too many values for parameter LD-API-Version").
func newAIAgentGraphBetaClient(c *Client) (*Client, error) {
	return newBetaClient(c.apiKey, c.apiHost, false, DEFAULT_HTTP_TIMEOUT_S, DEFAULT_MAX_CONCURRENCY)
}

// agentGraphEdgeModel is the Terraform representation of a single edge in an
// agent graph. It is shared by the resource and data source.
type agentGraphEdgeModel struct {
	Key          types.String `tfsdk:"key"`
	SourceConfig types.String `tfsdk:"source_config"`
	TargetConfig types.String `tfsdk:"target_config"`
	Handoff      types.String `tfsdk:"handoff"`
}

// agentGraphEdgeAttrTypes is the attribute type map for a single edge object,
// used when building the types.List value on read.
func agentGraphEdgeAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		KEY:           types.StringType,
		SOURCE_CONFIG: types.StringType,
		TARGET_CONFIG: types.StringType,
		HANDOFF:       types.StringType,
	}
}

func agentGraphEdgeObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: agentGraphEdgeAttrTypes()}
}

// aiAgentGraphIDToKeys splits a composite import ID of the form
// `project_key/graph_key` into its parts.
func aiAgentGraphIDToKeys(id string) (projectKey, graphKey string, err error) {
	parts := splitID(id, 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("import ID must be in the format project_key/graph_key, got: %q", id)
	}
	return parts[0], parts[1], nil
}

// sortedEdgeKeys returns the edge-map keys in a stable order so request
// bodies are deterministic.
func sortedEdgeKeys(edges map[string]agentGraphEdgeModel) []string {
	keys := make([]string, 0, len(edges))
	for k := range edges {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// agentGraphEdgePostsFromModel converts the Terraform edge models (keyed by
// edge key) into the generated client's POST representation used when
// creating a graph. The map key is the edge's authoritative key.
func agentGraphEdgePostsFromModel(edges map[string]agentGraphEdgeModel) ([]ldapi.AgentGraphEdgePost, error) {
	out := make([]ldapi.AgentGraphEdgePost, 0, len(edges))
	for _, k := range sortedEdgeKeys(edges) {
		e := edges[k]
		edge := ldapi.NewAgentGraphEdgePost(k, e.SourceConfig.ValueString(), e.TargetConfig.ValueString())
		handoff, err := edgeHandoffMap(k, e)
		if err != nil {
			return nil, err
		}
		edge.Handoff = handoff
		out = append(out, *edge)
	}
	return out, nil
}

// agentGraphEdgesFromModel converts the Terraform edge models (keyed by edge
// key) into the generated client's representation used when updating a graph
// (PATCH replaces all existing edges).
func agentGraphEdgesFromModel(edges map[string]agentGraphEdgeModel) ([]ldapi.AgentGraphEdge, error) {
	out := make([]ldapi.AgentGraphEdge, 0, len(edges))
	for _, k := range sortedEdgeKeys(edges) {
		e := edges[k]
		edge := ldapi.NewAgentGraphEdge(k, e.SourceConfig.ValueString(), e.TargetConfig.ValueString())
		handoff, err := edgeHandoffMap(k, e)
		if err != nil {
			return nil, err
		}
		edge.Handoff = handoff
		out = append(out, *edge)
	}
	return out, nil
}

func edgeHandoffMap(key string, e agentGraphEdgeModel) (map[string]interface{}, error) {
	if e.Handoff.IsNull() || e.Handoff.IsUnknown() || e.Handoff.ValueString() == "" {
		return nil, nil
	}
	m, err := jsonStringToMap(e.Handoff.ValueString())
	if err != nil {
		return nil, fmt.Errorf("invalid handoff JSON for edge %q: %w", key, err)
	}
	return m, nil
}

// agentGraphEdgeModelsFromAPI converts the API edge representation into the
// Terraform edge models keyed by edge key. handoff serializes back to a JSON
// string, null when empty so an unset handoff round-trips cleanly.
func agentGraphEdgeModelsFromAPI(edges []ldapi.AgentGraphEdge) (map[string]agentGraphEdgeModel, error) {
	out := make(map[string]agentGraphEdgeModel, len(edges))
	for _, e := range edges {
		handoffJSON, err := mapToJsonString(e.Handoff)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize handoff for edge %q: %w", e.GetKey(), err)
		}
		out[e.GetKey()] = agentGraphEdgeModel{
			Key:          types.StringValue(e.GetKey()),
			SourceConfig: types.StringValue(e.GetSourceConfig()),
			TargetConfig: types.StringValue(e.GetTargetConfig()),
			Handoff:      stringValueOrNull(handoffJSON),
		}
	}
	return out, nil
}
