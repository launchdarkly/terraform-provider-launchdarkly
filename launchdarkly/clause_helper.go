package launchdarkly

// clause_helper.go retains the value-coercion + type-inference helpers
// that the framework clauses code in clauses_framework.go uses. The
// SDKv2 schema + ResourceData converters were deleted with the SDKv2
// segment / feature_flag / feature_flag_environment resources.

import (
	"fmt"
	"reflect"
	"strconv"
)

const (
	BOOL_CLAUSE_VALUE   = "boolean"
	STRING_CLAUSE_VALUE = "string"
	NUMBER_CLAUSE_VALUE = "number"
)

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
