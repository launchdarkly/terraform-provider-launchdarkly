package launchdarkly

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestViewAssociationSettingNeedsPatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		plan     types.Bool
		state    types.Bool
		isCreate bool
		want     bool
	}{
		{
			name:     "create default false does not patch",
			plan:     types.BoolValue(false),
			isCreate: true,
			want:     false,
		},
		{
			name:     "create unset does not patch",
			plan:     types.BoolNull(),
			isCreate: true,
			want:     false,
		},
		{
			name:     "create true patches",
			plan:     types.BoolValue(true),
			isCreate: true,
			want:     true,
		},
		{
			name:  "update unchanged false does not patch",
			plan:  types.BoolValue(false),
			state: types.BoolValue(false),
			want:  false,
		},
		{
			name:  "update unchanged true does not patch",
			plan:  types.BoolValue(true),
			state: types.BoolValue(true),
			want:  false,
		},
		{
			name:  "update false to true patches",
			plan:  types.BoolValue(true),
			state: types.BoolValue(false),
			want:  true,
		},
		{
			name:  "update true to false patches",
			plan:  types.BoolValue(false),
			state: types.BoolValue(true),
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, viewAssociationSettingNeedsPatch(tt.plan, tt.state, tt.isCreate))
		})
	}
}
