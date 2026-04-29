package state

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DriftDetector detects differences between state and actual infrastructure
type DriftDetector struct{}

// NewDriftDetector creates a new DriftDetector instance
func NewDriftDetector() *DriftDetector {
	return &DriftDetector{}
}

// DetectDrift compares state with API response and returns differences
// Returns a list of field names that have drifted
func (dd *DriftDetector) DetectDrift(stateValue, apiValue interface{}) ([]string, error) {
	if stateValue == nil || apiValue == nil {
		return nil, fmt.Errorf("stateValue and apiValue cannot be nil")
	}

	var drifts []string

	stateVal := reflect.ValueOf(stateValue)
	apiVal := reflect.ValueOf(apiValue)

	// Handle pointers
	if stateVal.Kind() == reflect.Ptr {
		stateVal = stateVal.Elem()
	}
	if apiVal.Kind() == reflect.Ptr {
		apiVal = apiVal.Elem()
	}

	// Ensure both are structs
	if stateVal.Kind() != reflect.Struct || apiVal.Kind() != reflect.Struct {
		return nil, fmt.Errorf("both stateValue and apiValue must be structs")
	}

	// Compare each field
	for i := 0; i < stateVal.NumField(); i++ {
		fieldName := stateVal.Type().Field(i).Name
		stateField := stateVal.Field(i)
		apiField := apiVal.FieldByName(fieldName)

		// Skip if field doesn't exist in API response
		if !apiField.IsValid() {
			continue
		}

		// Skip if types don't match
		if stateField.Type() != apiField.Type() {
			continue
		}

		// Compare values based on type
		if hasDrift := dd.compareValues(stateField, apiField); hasDrift {
			drifts = append(drifts, fieldName)
		}
	}

	return drifts, nil
}

// compareValues compares two reflect.Value instances and returns true if they differ
func (dd *DriftDetector) compareValues(stateField, apiField reflect.Value) bool {
	// Handle Terraform types specially
	switch stateField.Interface().(type) {
	case types.String:
		stateStr := stateField.Interface().(types.String)
		apiStr := apiField.Interface().(types.String)
		return !stateStr.Equal(apiStr)
	case types.Int64:
		stateInt := stateField.Interface().(types.Int64)
		apiInt := apiField.Interface().(types.Int64)
		return !stateInt.Equal(apiInt)
	case types.Bool:
		stateBool := stateField.Interface().(types.Bool)
		apiBool := apiField.Interface().(types.Bool)
		return !stateBool.Equal(apiBool)
	case types.Float64:
		stateFloat := stateField.Interface().(types.Float64)
		apiFloat := apiField.Interface().(types.Float64)
		return !stateFloat.Equal(apiFloat)
	case types.Object:
		stateObj := stateField.Interface().(types.Object)
		apiObj := apiField.Interface().(types.Object)
		return !stateObj.Equal(apiObj)
	case types.List:
		stateList := stateField.Interface().(types.List)
		apiList := apiField.Interface().(types.List)
		return !stateList.Equal(apiList)
	case types.Set:
		stateSet := stateField.Interface().(types.Set)
		apiSet := apiField.Interface().(types.Set)
		return !stateSet.Equal(apiSet)
	case types.Map:
		stateMap := stateField.Interface().(types.Map)
		apiMap := apiField.Interface().(types.Map)
		return !stateMap.Equal(apiMap)
	default:
		// For other types, use reflect.DeepEqual
		return !reflect.DeepEqual(stateField.Interface(), apiField.Interface())
	}
}
