package launchdarkly

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
)

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

// agentGraphEdgePostsFromModel converts the Terraform edge models into the
// generated client's POST representation used when creating a graph.
func agentGraphEdgePostsFromModel(edges []agentGraphEdgeModel) ([]ldapi.AgentGraphEdgePost, error) {
	out := make([]ldapi.AgentGraphEdgePost, 0, len(edges))
	for _, e := range edges {
		edge := ldapi.NewAgentGraphEdgePost(e.Key.ValueString(), e.SourceConfig.ValueString(), e.TargetConfig.ValueString())
		handoff, err := edgeHandoffMap(e)
		if err != nil {
			return nil, err
		}
		edge.Handoff = handoff
		out = append(out, *edge)
	}
	return out, nil
}

// agentGraphEdgesFromModel converts the Terraform edge models into the
// generated client's representation used when updating a graph (PATCH replaces
// all existing edges).
func agentGraphEdgesFromModel(edges []agentGraphEdgeModel) ([]ldapi.AgentGraphEdge, error) {
	out := make([]ldapi.AgentGraphEdge, 0, len(edges))
	for _, e := range edges {
		edge := ldapi.NewAgentGraphEdge(e.Key.ValueString(), e.SourceConfig.ValueString(), e.TargetConfig.ValueString())
		handoff, err := edgeHandoffMap(e)
		if err != nil {
			return nil, err
		}
		edge.Handoff = handoff
		out = append(out, *edge)
	}
	return out, nil
}

func edgeHandoffMap(e agentGraphEdgeModel) (map[string]interface{}, error) {
	if e.Handoff.IsNull() || e.Handoff.IsUnknown() || e.Handoff.ValueString() == "" {
		return nil, nil
	}
	m, err := jsonStringToMap(e.Handoff.ValueString())
	if err != nil {
		return nil, fmt.Errorf("invalid handoff JSON for edge %q: %w", e.Key.ValueString(), err)
	}
	return m, nil
}

// agentGraphEdgeModelsFromAPI converts the API edge representation into the
// Terraform edge models. handoff serializes back to a JSON string, null when
// empty so an unset handoff round-trips cleanly.
func agentGraphEdgeModelsFromAPI(edges []ldapi.AgentGraphEdge) ([]agentGraphEdgeModel, error) {
	out := make([]agentGraphEdgeModel, 0, len(edges))
	for _, e := range edges {
		handoffJSON, err := mapToJsonString(e.Handoff)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize handoff for edge %q: %w", e.GetKey(), err)
		}
		out = append(out, agentGraphEdgeModel{
			Key:          types.StringValue(e.GetKey()),
			SourceConfig: types.StringValue(e.GetSourceConfig()),
			TargetConfig: types.StringValue(e.GetTargetConfig()),
			Handoff:      stringValueOrNull(handoffJSON),
		})
	}
	return out, nil
}
