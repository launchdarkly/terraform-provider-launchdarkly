package launchdarkly

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v23"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAIAgentGraphIDToKeys(t *testing.T) {
	t.Run("valid composite ID", func(t *testing.T) {
		projectKey, graphKey, err := aiAgentGraphIDToKeys("my-project/my-graph")
		require.NoError(t, err)
		assert.Equal(t, "my-project", projectKey)
		assert.Equal(t, "my-graph", graphKey)
	})

	t.Run("missing separator", func(t *testing.T) {
		_, _, err := aiAgentGraphIDToKeys("my-graph")
		assert.Error(t, err)
	})

	t.Run("too many parts", func(t *testing.T) {
		_, _, err := aiAgentGraphIDToKeys("a/b/c")
		assert.Error(t, err)
	})
}

func TestAgentGraphEdgePostsFromModel(t *testing.T) {
	edges := map[string]agentGraphEdgeModel{
		"edge-1": {
			Key:          types.StringValue("edge-1"),
			SourceConfig: types.StringValue("config-a"),
			TargetConfig: types.StringValue("config-b"),
			Handoff:      types.StringValue(`{"reason":"escalate"}`),
		},
		"edge-2": {
			// key omitted in config: the map key is authoritative.
			SourceConfig: types.StringValue("config-b"),
			TargetConfig: types.StringValue("config-c"),
			Handoff:      types.StringNull(),
		},
	}

	posts, err := agentGraphEdgePostsFromModel(edges)
	require.NoError(t, err)
	require.Len(t, posts, 2)

	assert.Equal(t, "edge-1", posts[0].GetKey())
	assert.Equal(t, "config-a", posts[0].GetSourceConfig())
	assert.Equal(t, "config-b", posts[0].GetTargetConfig())
	assert.Equal(t, "escalate", posts[0].Handoff["reason"])

	assert.Equal(t, "edge-2", posts[1].GetKey())
	assert.Nil(t, posts[1].Handoff)
}

func TestAgentGraphEdgePostsFromModel_InvalidHandoff(t *testing.T) {
	edges := map[string]agentGraphEdgeModel{
		"edge-1": {
			SourceConfig: types.StringValue("config-a"),
			TargetConfig: types.StringValue("config-b"),
			Handoff:      types.StringValue("not-json"),
		},
	}
	_, err := agentGraphEdgePostsFromModel(edges)
	assert.Error(t, err)
}

func TestAgentGraphEdgesFromModel(t *testing.T) {
	edges := map[string]agentGraphEdgeModel{
		"edge-1": {
			SourceConfig: types.StringValue("config-a"),
			TargetConfig: types.StringValue("config-b"),
			Handoff:      types.StringValue(`{"reason":"escalate"}`),
		},
	}
	out, err := agentGraphEdgesFromModel(edges)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "edge-1", out[0].GetKey())
	assert.Equal(t, "escalate", out[0].Handoff["reason"])
}

func TestAgentGraphEdgeModelsFromAPI(t *testing.T) {
	apiEdges := []ldapi.AgentGraphEdge{
		{
			Key:          "edge-1",
			SourceConfig: "config-a",
			TargetConfig: "config-b",
			Handoff:      map[string]interface{}{"reason": "escalate"},
		},
		{
			Key:          "edge-2",
			SourceConfig: "config-b",
			TargetConfig: "config-c",
		},
	}

	models, err := agentGraphEdgeModelsFromAPI(apiEdges)
	require.NoError(t, err)
	require.Len(t, models, 2)

	assert.Equal(t, "edge-1", models["edge-1"].Key.ValueString())
	assert.Equal(t, "config-a", models["edge-1"].SourceConfig.ValueString())
	assert.Equal(t, "config-b", models["edge-1"].TargetConfig.ValueString())
	assert.JSONEq(t, `{"reason":"escalate"}`, models["edge-1"].Handoff.ValueString())

	// An edge with no handoff should round-trip to a null string.
	assert.True(t, models["edge-2"].Handoff.IsNull())
}

func TestAgentGraphEdgeModelsFromAPI_Empty(t *testing.T) {
	models, err := agentGraphEdgeModelsFromAPI(nil)
	require.NoError(t, err)
	assert.Empty(t, models)
}
