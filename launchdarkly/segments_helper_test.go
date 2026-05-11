package launchdarkly

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSegmentPostCreatePatchOps(t *testing.T) {
	segSchema := baseSegmentSchema(segmentSchemaOptions{isDataSource: false})

	testCases := []struct {
		name           string
		raw            map[string]interface{}
		expectedFields []string
		expectedPaths  []string
	}{
		{
			name: "minimal config: no PATCH-only fields set",
			raw: map[string]interface{}{
				DESCRIPTION: "minimal",
				TAGS:        []interface{}{"tf-test"},
				UNBOUNDED:   false,
			},
			expectedFields: nil,
			expectedPaths:  nil,
		},
		{
			name: "rules only",
			raw: map[string]interface{}{
				DESCRIPTION: "with rules",
				RULES: []interface{}{
					map[string]interface{}{
						CLAUSES: []interface{}{
							map[string]interface{}{
								ATTRIBUTE:    "country",
								OP:           "in",
								VALUES:       []interface{}{"US"},
								NEGATE:       false,
								CONTEXT_KIND: "user",
								VALUE_TYPE:   "string",
							},
						},
					},
				},
			},
			expectedFields: []string{RULES},
			expectedPaths:  []string{"/rules"},
		},
		{
			name: "included + included_contexts",
			raw: map[string]interface{}{
				DESCRIPTION: "with included + ctx",
				INCLUDED:    []interface{}{"user-a"},
				INCLUDED_CONTEXTS: []interface{}{
					map[string]interface{}{
						VALUES:       []interface{}{"org-1"},
						CONTEXT_KIND: "organization",
					},
				},
			},
			expectedFields: []string{INCLUDED, INCLUDED_CONTEXTS},
			expectedPaths:  []string{"/included", "/includedContexts"},
		},
		{
			name: "all PATCH-only fields set",
			raw: map[string]interface{}{
				DESCRIPTION: "everything",
				INCLUDED:    []interface{}{"user-a"},
				EXCLUDED:    []interface{}{"user-b"},
				INCLUDED_CONTEXTS: []interface{}{
					map[string]interface{}{
						VALUES:       []interface{}{"org-1"},
						CONTEXT_KIND: "organization",
					},
				},
				EXCLUDED_CONTEXTS: []interface{}{
					map[string]interface{}{
						VALUES:       []interface{}{"org-2"},
						CONTEXT_KIND: "organization",
					},
				},
				RULES: []interface{}{
					map[string]interface{}{
						CLAUSES: []interface{}{
							map[string]interface{}{
								ATTRIBUTE:    "country",
								OP:           "in",
								VALUES:       []interface{}{"US"},
								NEGATE:       false,
								CONTEXT_KIND: "user",
								VALUE_TYPE:   "string",
							},
						},
					},
				},
			},
			expectedFields: []string{INCLUDED, EXCLUDED, INCLUDED_CONTEXTS, EXCLUDED_CONTEXTS, RULES},
			expectedPaths:  []string{"/included", "/excluded", "/includedContexts", "/excludedContexts", "/rules"},
		},
		{
			name: "empty slices are treated as unset",
			raw: map[string]interface{}{
				DESCRIPTION:       "explicit empties",
				INCLUDED:          []interface{}{},
				EXCLUDED:          []interface{}{},
				INCLUDED_CONTEXTS: []interface{}{},
				EXCLUDED_CONTEXTS: []interface{}{},
				RULES:             []interface{}{},
			},
			expectedFields: nil,
			expectedPaths:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := schema.TestResourceDataRaw(t, segSchema, tc.raw)
			ops, fields, err := segmentPostCreatePatchOps(d)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedFields, fields)
			require.Len(t, ops, len(tc.expectedPaths))
			for i, path := range tc.expectedPaths {
				assert.Equal(t, path, ops[i].Path, "op %d path", i)
				assert.Equal(t, "replace", ops[i].Op, "op %d op", i)
			}
		})
	}
}
