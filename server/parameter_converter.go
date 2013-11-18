package server

import (
	"fmt"
	"github.com/rwl/go-endpoints/endpoints"
	"strconv"
	"strings"
)

var booleanValues map[string]bool = map[string]bool{
	"1": true, "true": true, "0": false, "false": false,
}

// Helper that converts parameter values to the type expected by the SPI.
//
// Parameter values that appear in the URL and the query string are usually
// converted to native types before being passed to the SPI. This code handles
// that conversion and some validation.

// Checks if the parameter value is valid if an enum.
//
// This is called by the transformParameterValue function and shouldn't be
// called directly.
//
// This verifies that the value of an enum parameter is valid.
//
// Takes the name of the parameter (Which is either just a variable name or
// the name with the index appended. For example "var" or "var[2]".), the
// value to be used as enum for the parameter and a spec containing
// information specific to the field in question (This is retrieved from
// request.Parameters in the method config.
//
// Returns an enumRejectionError if the given value is not among the accepted
// enum values in the field parameter.
func checkEnum(parameterName string, value string, parameterConfig *endpoints.ApiRequestParamSpec) error {
	//	if parameterConfig == nil || parameterConfig.Enum == nil || len(parameterConfig.Enum) == 0 {
	//		return nil
	//	}

	enumValues := make([]string, 0)
	for _, enum := range parameterConfig.Enum {
		if enum.BackendVal != "" {
			enumValues = append(enumValues, enum.BackendVal)
		}
	}

	for _, ev := range enumValues {
		if value == ev {
			return nil
		}
	}
	return newEnumRejectionError(parameterName, value, enumValues)
}

// Checks if a boolean value is valid.
//
// This is called by the transformParameterValue function and shouldn't be
// called directly.
//
// This checks that the string value passed in can be converted to a valid
// boolean value.
//
// Takes the name of the parameter (Which is either just a variable name or
// the name with the index appended. For example "var" or "var[2]".), the
// value passed in for the parameter and a spec containing
// information specific to the field in question (This is retrieved from
// request.Parameters in the method config.
//
// Returns an basicTypeParameterError if the given value is not a valid
// boolean value.
func checkBoolean(parameterName string, value string, parameterConfig *endpoints.ApiRequestParamSpec) error {
	if parameterConfig.Type != "boolean" {
		return nil
	}

	_, ok := booleanValues[strings.ToLower(value)]
	if !ok {
		return newBasicTypeParameterError(parameterName, value, "boolean")
	}
	return nil
}

// Convert a string to a boolean value the same way the server does.
//
// This is called by the transform_parameter_value function and shouldn't be
// called directly.
//
// Returns true or false, based on whether the value in the string would be
// interpreted as true or false by the server. In the case of an invalid
// entry, this returns false.
func convertBoolean(value string) (interface{}, error) {
	b, ok := booleanValues[strings.ToLower(value)]
	if ok {
		return b, nil
	}
	return false, nil
}

func convertInt(value string) (interface{}, error) {
	i, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return value, err
	}
	return int(i), nil
}

func convertUnsignedInt(value string) (interface{}, error) {
	i, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return value, err
	}
	return uint(i), nil
}

func convertFloat32(value string) (interface{}, error) {
	f, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return value, err
	}
	return float32(f), nil
}

func convertFloat64(value string) (interface{}, error) {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return value, err
	}
	return float64(f), nil
}

// Map to convert parameters from strings to their desired back-end format.
// Anything not listed here will remain a string. Note that the server
// keeps int64 and uint64 as strings when passed to the SPI.
// This maps a type name from the .api method configuration to a type with
// a validation function, a conversion function and a descriptive type name.
// The descriptive type name is only used in conversion error messages, and
// the names here are chosen to match the error messages from the server.
// Note that the 'enum' entry is special cased. Enums have 'type': 'string',
// so we have special case code to recognize them and use the 'enum' map
// entry.
var paramConversions map[string]*paramConverter = map[string]*paramConverter{
	"boolean": &paramConverter{checkBoolean, convertBoolean, "boolean"},
	"int32":   &paramConverter{nil, convertInt, "integer"},
	"uint32":  &paramConverter{nil, convertUnsignedInt, "integer"},
	"float":   &paramConverter{nil, convertFloat32, "float"},
	"double":  &paramConverter{nil, convertFloat64, "double"},
	"enum":    &paramConverter{checkEnum, nil, ""},
}

type paramConverter struct {
	validator func(string, string, *endpoints.ApiRequestParamSpec) error
	converter func(string) (interface{}, error)
	typeName  string
}

// Get information needed to convert the given parameter to its SPI type.
//
// Returns a struct with functions/information needed to validate and convert
// the given parameter from a string to the type expected by the SPI.
func getParameterConversionEntry(parameterConfig *endpoints.ApiRequestParamSpec) *paramConverter {
	entry, ok := paramConversions[parameterConfig.Type]

	// Special handling for enum parameters.  An enum's type is 'string', so
	// we need to detect them by the presence of an 'enum' property in their
	// configuration.
	if !ok && parameterConfig.Enum != nil {
		entry = paramConversions["enum"]
	}

	return entry
}

// Recursively calls transformParameterValue on the values in the array.
//
// Note that '[index-of-value]' is appended to the parameter name for
// error reporting purposes.

// Validates and transforms parameters to the type expected by the SPI.
//
// If the value is an array this will recursively call transformParameterValue
// on the values in the array. Otherwise, it checks all parameter rules for the
// the current value and converts its type from a string to whatever format
// the SPI expects.
//
// In the array case, '[index-of-value]' is appended to the parameter name for
// error reporting purposes.
//
// Takes the name of the parameter (Which is either just a variable name or
// the name with the index appended. For example "var" or "var[2]".), the
// value or array of values to be validated, transformed, and passed along to
// the SPI and a spec containing information specific to the field in question
// (This is retrieved from request.Parameters in the method config.
//
// Returns the converted parameter value or an error if conversion failed. Not
// all types are converted, so this may be the same string that's passed in.
func transformParameterValue(parameterName string, value interface{}, parameterConfig *endpoints.ApiRequestParamSpec) (interface{}, error) {
	if arrVal, ok := value.([]interface{}); ok {
		// We're only expecting to handle path and query string parameters here.
		// The way path and query string parameters are passed in, they'll likely
		// only be single values or singly-nested lists (no lists nested within
		// lists). But even if there are nested lists, we'd want to preserve that
		// structure. These recursive calls should preserve it and convert all
		// parameter values. See the documentation for information about the
		// parameter renaming done here.
		converted := make([]interface{}, 0, len(arrVal))
		for index, element := range arrVal {
			paramName := fmt.Sprintf("%s[%d]", parameterName, index)
			c, err := transformParameterValue(paramName, element, parameterConfig)
			if err != nil {
				return value, err
			}
			converted = append(converted, c)
		}
		return converted, nil
	}
	if arrVal, ok := value.([]string); ok {
		converted := make([]interface{}, 0, len(arrVal))
		for index, element := range arrVal {
			paramName := fmt.Sprintf("%s[%d]", parameterName, index)
			c, err := transformParameterValue(paramName, element, parameterConfig)
			if err != nil {
				return value, err
			}
			converted = append(converted, c)
		}
		return converted, nil
	}
	if strVal, ok := value.(string); ok {
		// Validate and convert the parameter value.
		entry := getParameterConversionEntry(parameterConfig)
		if entry != nil {
			if entry.validator != nil {
				err := entry.validator(parameterName, strVal, parameterConfig)
				if err != nil {
					return value, err
				}
			}
			if entry.converter != nil {
				converted, err := entry.converter(strVal)
				if err != nil {
					return nil, newBasicTypeParameterError(parameterName,
						strVal, entry.typeName)
				}
				return converted, nil
			}
		}
		return strVal, nil
	} else {
		return value, fmt.Errorf("Parameter value '%#v' of unexpected type", value)
	}
}
