package launchdarkly

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	ldapi "github.com/launchdarkly/api-client-go/v15"
)

const (
	BOOL_CLAUSE_VALUE   = "boolean"
	STRING_CLAUSE_VALUE = "string"
	NUMBER_CLAUSE_VALUE = "number"
)

func clauseSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		Description: "List of nested blocks specifying the logical clauses to evaluate",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				ATTRIBUTE: {
					Type:        schema.TypeString,
					Required:    true,
					Description: "The user attribute to operate on",
				},
				OP: {
					Type:             schema.TypeString,
					Required:         true,
					Description:      "The operator associated with the rule clause. Available options are `in`, `endsWith`, `startsWith`, `matches`, `contains`, `lessThan`, `lessThanOrEqual`, `greaterThanOrEqual`, `before`, `after`, `segmentMatch`, `semVerEqual`, `semVerLessThan`, and `semVerGreaterThan`. Read LaunchDarkly's [Operators](https://docs.launchdarkly.com/sdk/concepts/flag-evaluation-rules#operators) documentation for more information.",
					ValidateDiagFunc: validateOp(),
				},
				VALUES: {
					Type: schema.TypeList,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
					Required:    true,
					Description: "The list of values associated with the rule clause.",
				},
				VALUE_TYPE: {
					Type:             schema.TypeString,
					Default:          STRING_CLAUSE_VALUE,
					Optional:         true,
					Description:      "The type for each of the clause's values. Available types are `boolean`, `string`, and `number`. If omitted, `value_type` defaults to `string`.",
					DiffSuppressFunc: diffSuppressFunc,
					ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(
						[]string{
							BOOL_CLAUSE_VALUE,
							STRING_CLAUSE_VALUE,
							NUMBER_CLAUSE_VALUE,
						},
						false,
					)),
				},
				NEGATE: {
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
					Description: "Whether to negate the rule clause.",
				},
				CONTEXT_KIND: {
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "user",
					Description: "The context kind associated with this rule clause. This argument is only valid if `rollout_weights` is also specified. If omitted, defaults to `user`.",
				},
			},
		},
	}
}

func diffSuppressFunc(_, old, new string, d *schema.ResourceData) bool {
	return (old == "" && new == STRING_CLAUSE_VALUE && d.Get("value_type") == nil)
}

func clauseFromResourceData(val interface{}) (ldapi.Clause, error) {
	clauseMap := val.(map[string]interface{})
	c := ldapi.Clause{
		Attribute: clauseMap[ATTRIBUTE].(string),
		Op:        clauseMap[OP].(string),
		Negate:    clauseMap[NEGATE].(bool),
	}
	if contextKind, ok := clauseMap[CONTEXT_KIND].(string); ok {
		if contextKind != "" {
			c.ContextKind = &contextKind
		}
	}
	valueType := clauseMap[VALUE_TYPE].(string)
	if valueType == "" {
		valueType = STRING_CLAUSE_VALUE
	}
	values, err := clauseValuesFromResourceData(clauseMap[VALUES].([]interface{}), valueType)
	if err != nil {
		return c, err
	}
	c.Values = values
	return c, nil
}

func clauseValuesFromResourceData(schemaValues []interface{}, valueType string) ([]interface{}, error) {
	typedValues := make([]interface{}, 0, len(schemaValues))
	for _, schemaValue := range schemaValues {
		strValue, ok := schemaValue.(string)
		if !ok {
			return nil, fmt.Errorf("invalid clause value: %v", schemaValue)
		}
		v, err := clauseValueFromResourceData(strValue, valueType)
		if err != nil {
			return nil, err
		}
		typedValues = append(typedValues, v)
	}
	return typedValues, nil
}

func clauseValueFromResourceData(strValue string, valueType string) (interface{}, error) {
	switch valueType {
	case STRING_CLAUSE_VALUE:
		return strValue, nil
	case BOOL_CLAUSE_VALUE:
		return convertBoolStringToBool(strValue)
	case NUMBER_CLAUSE_VALUE:
		return convertNumberStringToFloat(strValue)
	}
	return nil, fmt.Errorf("invalid clause value type %q", valueType)
}

func convertBoolStringToBool(boolStr string) (bool, error) {
	switch boolStr {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	return false, fmt.Errorf("invalid boolean string %q", boolStr)
}

func convertNumberStringToFloat(numStr string) (float64, error) {
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return num, fmt.Errorf("invalid number string %q", numStr)
	}
	return num, nil
}

func clausesToResourceData(clauses []ldapi.Clause) (interface{}, error) {
	transformed := make([]interface{}, len(clauses))

	for i, c := range clauses {
		var err error
		var valueType string
		strValues := make([]interface{}, 0, len(c.Values))
		for _, v := range c.Values {
			valueType, err = inferClauseValueTypeFromValue(v)
			if err != nil {
				return transformed, err
			}
			strValues = append(strValues, stringifyValue(v))
		}
		contextKind := "user" // should default to user
		if c.ContextKind != nil {
			contextKind = *c.ContextKind
		}
		transformed[i] = map[string]interface{}{
			ATTRIBUTE:    c.Attribute,
			OP:           c.Op,
			VALUES:       strValues,
			VALUE_TYPE:   valueType,
			NEGATE:       c.Negate,
			CONTEXT_KIND: contextKind,
		}
	}
	return transformed, nil
}

func inferClauseValueTypeFromValue(value interface{}) (string, error) {
	switch value.(type) {
	case bool:
		return BOOL_CLAUSE_VALUE, nil
	case string:
		return STRING_CLAUSE_VALUE, nil
	case float64:
		return NUMBER_CLAUSE_VALUE, nil
	}
	return "", fmt.Errorf("unknown clause value type: %q", reflect.TypeOf(value))
}
