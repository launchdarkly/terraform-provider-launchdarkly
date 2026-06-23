package launchdarkly

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	ldapi "github.com/launchdarkly/api-client-go/v22"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReleasePolicyIdToKeys(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		id       string
		wantProj string
		wantKey  string
		wantErr  bool
	}{
		{name: "valid", id: "my-project/my-policy", wantProj: "my-project", wantKey: "my-policy"},
		{name: "missing separator", id: "my-policy", wantErr: true},
		{name: "too many separators", id: "a/b/c", wantErr: true},
		{name: "empty project", id: "/my-policy", wantErr: true},
		{name: "empty policy", id: "my-project/", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			proj, key, err := releasePolicyIdToKeys(tc.id)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantProj, proj)
			assert.Equal(t, tc.wantKey, key)
		})
	}
}

func TestReleasePolicyStagesRoundTrip(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	stages := []ldapi.ReleasePolicyStage{
		{Allocation: 10, DurationMillis: 3600000},
		{Allocation: 50, DurationMillis: 7200000},
	}

	list, diags := releasePolicyStagesToList(stages)
	require.False(t, diags.HasError())
	require.False(t, list.IsNull())

	got, diags := releasePolicyStagesToAPI(ctx, list)
	require.False(t, diags.HasError())
	assert.Equal(t, stages, got)

	// An empty slice maps to a null list (omitted attribute).
	emptyList, diags := releasePolicyStagesToList(nil)
	require.False(t, diags.HasError())
	assert.True(t, emptyList.IsNull())
}

func TestReleasePolicyScopeToAPI_nullObjectIsNil(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	scope, diags := releasePolicyScopeToAPI(ctx, types.ObjectNull(releasePolicyScopeAttrTypes))
	require.False(t, diags.HasError())
	assert.Nil(t, scope)
}

func TestGuardedReleaseConfigToObject_nilIsNull(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	obj, diags := guardedReleaseConfigToObject(ctx, nil, types.ObjectNull(guardedReleaseConfigAttrTypes))
	require.False(t, diags.HasError())
	assert.True(t, obj.IsNull())
}
