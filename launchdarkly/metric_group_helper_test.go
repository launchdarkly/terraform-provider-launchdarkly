package launchdarkly

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v23"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricGroupIdToKeys(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		id         string
		wantProj   string
		wantKey    string
		wantErrStr bool
	}{
		{name: "valid", id: "my-project/my-group", wantProj: "my-project", wantKey: "my-group"},
		{name: "missing separator", id: "my-group", wantErrStr: true},
		{name: "too many separators", id: "a/b/c", wantErrStr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			proj, key, err := metricGroupIdToKeys(tc.id)
			if tc.wantErrStr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantProj, proj)
			assert.Equal(t, tc.wantKey, key)
		})
	}
}

func TestMetricGroupInputsFromModels(t *testing.T) {
	t.Parallel()
	in := []metricGroupMetricModel{
		{Key: types.StringValue("step-one"), NameInGroup: types.StringValue("First step")},
		{Key: types.StringValue("step-two"), NameInGroup: types.StringNull()},
	}
	got := metricGroupInputsFromModels(in)
	require.Len(t, got, 2)
	// Order must be preserved — funnel ordering is significant.
	assert.Equal(t, "step-one", got[0].Key)
	assert.Equal(t, "First step", got[0].NameInGroup)
	assert.Equal(t, "step-two", got[1].Key)
	assert.Equal(t, "", got[1].NameInGroup)
}

func TestMetricGroupMetricsToList(t *testing.T) {
	t.Parallel()
	nameInGroup := "First step"
	metrics := []ldapi.MetricInGroupRep{
		{Key: "step-one", NameInGroup: &nameInGroup},
		{Key: "step-two"},
	}
	list, err := metricGroupMetricsToList(metrics)
	require.NoError(t, err)
	require.False(t, list.IsNull())

	var out []metricGroupMetricModel
	diags := list.ElementsAs(context.Background(), &out, false)
	require.False(t, diags.HasError())
	require.Len(t, out, 2)
	assert.Equal(t, "step-one", out[0].Key.ValueString())
	assert.Equal(t, "First step", out[0].NameInGroup.ValueString())
	assert.Equal(t, "step-two", out[1].Key.ValueString())
	// A standard-group metric (no name_in_group) round-trips as null.
	assert.True(t, out[1].NameInGroup.IsNull())
}
