package pfsense

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/sjafferali/pfsense-api-goclient/pfsenseapi"
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

func parseValue(i interface{}) interface{} {
	if i == nil {
		return nil
	}

	value := reflect.ValueOf(i)
	kind := value.Kind()

	switch kind {
	case reflect.Chan, reflect.Map:
		if value.IsNil() || value.Len() == 0 {
			return nil
		}

		return i
	case reflect.Slice, reflect.Array:
		if value.IsNil() || value.Len() == 0 {
			return nil
		}

		return i
	case reflect.Func, reflect.Interface:
		if value.IsNil() {
			return nil
		}

		return i
	case reflect.String:
		if value.Len() == 0 {
			return nil
		}

		return i
	case reflect.Pointer, reflect.UnsafePointer:
		if value.IsNil() {
			return nil
		}

		if parseValue(value.Elem().Interface()) == nil {
			return nil
		}

		return i
	default:
		oji, ok := i.(pfsenseapi.OptionalJSONInt)

		if ok {
			if oji.Value == nil {
				return nil
			} else {
				return *oji.Value
			}
		}

		return i
	}
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
