package launchdarkly

import (
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	ldapi "github.com/launchdarkly/api-client-go"
	"github.com/stretchr/testify/require"
)

func TestVariationsFromResourceData(t *testing.T) {
	resourceData := schema.TestResourceDataRaw(t,
		map[string]*schema.Schema{variations: variationsSchema()},
		map[string]interface{}{variations: []map[string]interface{}{
			{
				name:        "nameValue",
				description: "descValue",
				value:       "a string value",
			},
			{
				name:        "nameValue2",
				description: "descValue2",
				value:       "another string value",
			},
		}},
	)

	expectedVariations := []ldapi.Variation{
		{"nameValue", "descValue", ptr("a string value")},
		{"nameValue2", "descValue2", ptr("another string value")},
	}

	actualVariations := variationsFromResourceData(resourceData)

	require.Len(t, actualVariations, 2)
	require.ElementsMatch(t, expectedVariations, actualVariations)
}
