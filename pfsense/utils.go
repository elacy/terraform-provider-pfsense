package pfsense

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

func isNilInterface(i interface{}) bool {
	// Check if the interface itself is nil
	if i == nil {
		return true
	}

	// Use reflection to check for nil pointers of unknown types
	value := reflect.ValueOf(i)
	kind := value.Kind()
	return kind >= reflect.Chan && kind <= reflect.Slice && value.IsNil()
}

func splitIntoArray(value string, separator string) []string {
	if value == "" {
		return []string{}
	}

	return strings.Split(value, separator)
}

func interfaceToStringArray(value interface{}) ([]string, error) {
	if value == nil {
		return nil, nil
	}

	output, ok := value.([]interface{})

	if !ok {
		return nil, fmt.Errorf("Value is %v not a []interface{}", output)
	}

	result := make([]string, len(output))

	for i, item := range output {
		result[i], ok = item.(string)

		if !ok {
			return nil, fmt.Errorf("Value %v contains non string values", output)
		}
	}

	return result, nil
}

func regexValidator(regularExpression string) *regexp.Regexp {
	r, err := regexp.Compile(regularExpression)

	if err != nil {
		panic(fmt.Sprintf("Unable to compile regex %s, had the following error: %v", regularExpression, err))
	}

	return r
}
