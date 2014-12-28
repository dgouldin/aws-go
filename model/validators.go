package model

import (
	"fmt"
	"github.com/stripe/aws-go/aws"
	"reflect"
	"regexp"
	"strings"
)

type Validator interface {
	Validate() error
}

type ValidationErrors map[string][]*ValidationError

func (e ValidationErrors) Error() string {
	errorStrs := []string{"The following fields falied validation:"}
	for fieldName, errors := range e {
		if len(errors) == 0 {
			continue
		}
		str := fmt.Sprintf(" - %s\n", fieldName)
		for _, err := range errors {
			str += fmt.Sprintf("   - %s", err.Error())
		}
		errorStrs = append(errorStrs, str)
	}
	return strings.Join(errorStrs, "\n")
}

type ValidationError struct {
	Type    string
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

func empty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return true
		}
	case reflect.Slice, reflect.Map:
		if v.Len() == 0 {
			return true
		}
	default:
		if !v.IsValid() {
			return true
		}
	}
	return false
}

func getStructField(v interface{}, name string) reflect.Value {
	value := reflect.ValueOf(v)
	switch value.Kind() {
	case reflect.Ptr:
		return getStructField(value.Elem().Interface(), name)
	case reflect.Struct:
		value = value.FieldByName(name)
	default:
		panic("Validations require a struct")
	}

	if value == reflect.ValueOf(nil) {
		panic(fmt.Sprintf("Field '%s' not found in type %s", name, reflect.TypeOf(v)))
	}

	return value
}

func ValidateRequired(v interface{}, name string) *ValidationError {
	fv := getStructField(v, name)
	if empty(fv) {
		return &ValidationError{
			Type:    "required",
			Message: "This field is required",
		}
	}
	return nil
}

func ValidateEnum(v interface{}, name string, enum []string) *ValidationError {
	fv := getStructField(v, name)
	if fv.IsNil() {
		return nil // If not required and not supplied, it's valid
	}

	value := *fv.Interface().(aws.StringValue)
	for _, e := range enum {
		if value == e {
			return nil
		}
	}
	return &ValidationError{
		Type:    "enum",
		Message: fmt.Sprintf("'%s' is not a valid value for this enum", value),
	}
}

func ValidateMin(v interface{}, name string, min int) *ValidationError {
	fv := getStructField(v, name)
	switch value := fv.Interface().(type) {
	case int:
		if value < min {
			return &ValidationError{
				Type:    "min",
				Message: fmt.Sprintf("This field must be at least %d", min),
			}
		}
	default:
		if fv.IsNil() || fv.Len() < min {
			return &ValidationError{
				Type:    "min",
				Message: fmt.Sprintf("This field length must be at least %d", min),
			}
		}
	}
	return nil
}

func ValidateMax(v interface{}, name string, max int) *ValidationError {
	fv := getStructField(v, name)
	switch fv.Kind() {
	case reflect.Int:
		if fv.Int() > int64(max) {
			return &ValidationError{
				Type:    "max",
				Message: fmt.Sprintf("This field must not be greater than %d", max),
			}
		}
	default:
		if fv.IsNil() {
			return nil // if not required and not supplied, it's valid
		}
		if fv.Len() > max {
			return &ValidationError{
				Type:    "max",
				Message: fmt.Sprintf("This field length must not be greater than %d", max),
			}
		}
	}
	return nil
}

func ValidatePattern(v interface{}, name string, pattern string) *ValidationError {
	fv := getStructField(v, name)
	if fv.IsNil() {
		return nil // If not required and not supplied, it's valid
	}

	value := *fv.Interface().(aws.StringValue)
	match, _ := regexp.MatchString(pattern, value)
	if !match {
		return &ValidationError{
			Type:    "pattern",
			Message: "This field does not match the specified pattern",
		}
	}
	return nil
}
