package launchdarkly

import (
	"testing"

	ldapi "github.com/launchdarkly/api-client-go/v17"
)

func TestValidateRuleResourceData(t *testing.T) {
	tests := []struct {
		name    string
		ruleMap map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid rule with variation",
			ruleMap: map[string]interface{}{
				ROLLOUT_WEIGHTS: []interface{}{},
				BUCKET_BY:       "",
				CONTEXT_KIND:    "",
				VARIATION:       1,
			},
			wantErr: false,
		},
		{
			name: "valid rule with rollout weights",
			ruleMap: map[string]interface{}{
				ROLLOUT_WEIGHTS: []interface{}{
					map[string]interface{}{
						"variation": 0,
						"weight":    50000,
					},
				},
				BUCKET_BY:    "email",
				CONTEXT_KIND: "user",
				VARIATION:    0,
			},
			wantErr: false,
		},
		{
			name: "invalid - bucket_by with variation",
			ruleMap: map[string]interface{}{
				ROLLOUT_WEIGHTS: []interface{}{},
				BUCKET_BY:       "email",
				CONTEXT_KIND:    "",
				VARIATION:       1,
			},
			wantErr: true,
			errMsg:  "rules: cannot use bucket_by argument with variation, only with rollout_weights",
		},
		{
			name: "invalid - context_kind with variation",
			ruleMap: map[string]interface{}{
				ROLLOUT_WEIGHTS: []interface{}{},
				BUCKET_BY:       "",
				CONTEXT_KIND:    "organization",
				VARIATION:       1,
			},
			wantErr: true,
			errMsg:  "rules: cannot use context_kind argument with variation, only with rollout_weights",
		},
		{
			name: "valid - context_kind user with variation",
			ruleMap: map[string]interface{}{
				ROLLOUT_WEIGHTS: []interface{}{},
				BUCKET_BY:       "",
				CONTEXT_KIND:    "user",
				VARIATION:       1,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRuleResourceData(tt.ruleMap)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRuleResourceData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("validateRuleResourceData() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestRulesToResourceData(t *testing.T) {
	bucketBy := "email"
	contextKind := "user"
	description := "test description"
	variation := int32(1)
	tests := []struct {
		name    string
		rules   []ldapi.Rule
		want    []map[string]interface{}
		wantErr bool
	}{
		{
			name: "rule with variation",
			rules: []ldapi.Rule{
				{
					Description: &description,
					Variation:   &variation,
					Clauses: []ldapi.Clause{
						{
							Attribute: "email",
							Op:        "contains",
							Values:    []interface{}{"test@test.com"},
						},
					},
				},
			},
			want: []map[string]interface{}{
				{
					DESCRIPTION: description,
					VARIATION:   &variation,
					CLAUSES: []interface{}{
						map[string]interface{}{
							"attribute": "email",
							"op":        "contains",
							"values":    []interface{}{"test@test.com"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "rule with rollout",
			rules: []ldapi.Rule{
				{
					Description: &description,
					Rollout: &ldapi.Rollout{
						BucketBy:    &bucketBy,
						ContextKind: &contextKind,
						Variations: []ldapi.WeightedVariation{
							{
								Variation: 0,
								Weight:    50000,
							},
						},
					},
					Clauses: []ldapi.Clause{
						{
							Attribute: "country",
							Op:        "in",
							Values:    []interface{}{"US", "CA"},
						},
					},
				},
			},
			want: []map[string]interface{}{
				{
					DESCRIPTION:  description,
					BUCKET_BY:    &bucketBy,
					CONTEXT_KIND: &contextKind,
					ROLLOUT_WEIGHTS: []interface{}{
						map[string]interface{}{
							"variation": 0,
							"weight":    50000,
						},
					},
					CLAUSES: []interface{}{
						map[string]interface{}{
							"attribute": "country",
							"op":        "in",
							"values":    []interface{}{"US", "CA"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "empty rules list",
			rules:   []ldapi.Rule{},
			want:    []map[string]interface{}{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rulesToResourceData(tt.rules)
			if (err != nil) != tt.wantErr {
				t.Errorf("rulesToResourceData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			gotSlice, ok := got.([]interface{})
			if !ok {
				t.Errorf("rulesToResourceData() returned type = %T, want []interface{}", got)
				return
			}

			if len(gotSlice) != len(tt.want) {
				t.Errorf("rulesToResourceData() returned %d rules, want %d", len(gotSlice), len(tt.want))
				return
			}

			for i, rule := range gotSlice {
				ruleMap, ok := rule.(map[string]interface{})
				if !ok {
					t.Errorf("rule %d is not a map[string]interface{}", i)
					continue
				}

				// Compare fields that should exist
				if tt.want[i][DESCRIPTION] != nil && ruleMap[DESCRIPTION] != tt.want[i][DESCRIPTION] {
					t.Errorf("rule %d description = %v, want %v", i, ruleMap[DESCRIPTION], tt.want[i][DESCRIPTION])
				}

				if tt.want[i][VARIATION] != nil && ruleMap[VARIATION] != tt.want[i][VARIATION] {
					t.Errorf("rule %d variation = %v, want %v", i, ruleMap[VARIATION], tt.want[i][VARIATION])
				}

				if tt.want[i][BUCKET_BY] != nil && ruleMap[BUCKET_BY] != tt.want[i][BUCKET_BY] {
					t.Errorf("rule %d bucket_by = %v, want %v", i, ruleMap[BUCKET_BY], tt.want[i][BUCKET_BY])
				}

				if tt.want[i][CONTEXT_KIND] != nil && ruleMap[CONTEXT_KIND] != tt.want[i][CONTEXT_KIND] {
					t.Errorf("rule %d context_kind = %v, want %v", i, ruleMap[CONTEXT_KIND], tt.want[i][CONTEXT_KIND])
				}
			}
		})
	}
}
