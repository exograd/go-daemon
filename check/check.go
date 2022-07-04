package check

import (
	"bytes"
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strconv"

	"github.com/exograd/go-daemon/djson"
)

type Checker struct {
	Pointer djson.Pointer
	Errors  ValidationErrors
}

type Object interface {
	Check(*Checker)
}

type ValidationError struct {
	Pointer djson.Pointer `json:"pointer"`
	Code    string        `json:"code"`
	Message string        `json:"message"`
}

type ValidationErrors []*ValidationError

func (err ValidationError) String() string {
	return fmt.Sprintf("ValidationError{%v, %q, %q}",
		err.Pointer, err.Code, err.Message)
}

func (err ValidationError) GoString() string {
	return err.String()
}

func (err ValidationError) Error() string {
	return fmt.Sprintf("%v: %s: %s", err.Pointer, err.Code, err.Message)
}

func (errs ValidationErrors) Error() string {
	var buf bytes.Buffer
	for _, err := range errs {
		buf.WriteString(err.Error())
		buf.WriteByte('\n')
	}
	return buf.String()
}

func NewChecker() *Checker {
	return &Checker{}
}

func (c *Checker) Error() error {
	if len(c.Errors) == 0 {
		return nil
	}

	return c.Errors
}

func (c *Checker) Push(token interface{}) {
	c.Pointer = pointerAppend(c.Pointer, token)
}

func (c *Checker) Pop() {
	c.Pointer = c.Pointer[:len(c.Pointer)-1]
}

func (c *Checker) WithChild(token interface{}, fn func()) {
	c.Push(token)
	defer c.Pop()

	fn()
}

func (c *Checker) AddError(token interface{}, code, format string, args ...interface{}) {
	var pointer djson.Pointer
	pointer = append(pointer, c.Pointer...)
	pointer = pointerAppend(pointer, token)

	err := ValidationError{
		Pointer: pointer,
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}

	c.Errors = append(c.Errors, &err)
}

func (c *Checker) Check(token interface{}, v bool, code, format string, args ...interface{}) bool {
	if !v {
		c.AddError(token, code, format, args...)
	}

	return v
}

func (c *Checker) CheckIntMin(token interface{}, i, min int) bool {
	return c.Check(token, i >= min, "integer_too_small",
		"integer %d must be greater or equal to %d", i, min)
}

func (c *Checker) CheckIntMax(token interface{}, i, max int) bool {
	return c.Check(token, i <= max, "integer_too_large",
		"integer %d must be lower or equal to %d", i, max)
}

func (c *Checker) CheckIntMinMax(token interface{}, i, min, max int) bool {
	if !c.CheckIntMin(token, i, min) {
		return false
	}

	return c.CheckIntMax(token, i, max)
}

func (c *Checker) CheckFloatMin(token interface{}, i, min float64) bool {
	return c.Check(token, i >= min, "float_too_small",
		"float %f must be greater or equal to %f", i, min)
}

func (c *Checker) CheckFloatMax(token interface{}, i, max float64) bool {
	return c.Check(token, i <= max, "float_too_large",
		"float %f must be lower or equal to %f", i, max)
}

func (c *Checker) CheckFloatMinMax(token interface{}, i, min, max float64) bool {
	if !c.CheckFloatMin(token, i, min) {
		return false
	}

	return c.CheckFloatMax(token, i, max)
}

func (c *Checker) CheckStringLengthMin(token interface{}, s string, min int) bool {
	return c.Check(token, len(s) >= min, "string_too_small",
		"string length must be greater or equal to %d", min)
}

func (c *Checker) CheckStringLengthMax(token interface{}, s string, max int) bool {
	return c.Check(token, len(s) <= max, "string_too_large",
		"string length must be lower or equal to %d", max)
}

func (c *Checker) CheckStringLengthMinMax(token interface{}, s string, min, max int) bool {
	if !c.CheckStringLengthMin(token, s, min) {
		return false
	}

	return c.CheckStringLengthMax(token, s, max)
}

func (c *Checker) CheckStringNotEmpty(token interface{}, s string) bool {
	return c.Check(token, s != "", "empty_string",
		"string must not be empty")
}

func (c *Checker) CheckStringValue(token interface{}, value interface{}, values interface{}) bool {
	valueType := reflect.TypeOf(value)
	if valueType.Kind() != reflect.String {
		panicf("value %#v (%T) is not a string", value, value)
	}

	s := reflect.ValueOf(value).String()

	valuesType := reflect.TypeOf(values)
	if valuesType.Kind() != reflect.Slice {
		panicf("values %#v (%T) are not a slice", values, values)
	}
	if valuesType.Elem().Kind() != reflect.String {
		panicf("values %#v (%T) are not a slice of strings", values, values)
	}

	valuesValue := reflect.ValueOf(values)

	found := false
	for i := 0; i < valuesValue.Len(); i++ {
		s2 := valuesValue.Index(i).String()
		if s == s2 {
			found = true
		}
	}

	var buf bytes.Buffer

	buf.WriteString("value must be one of the following strings: ")

	for i := 0; i < valuesValue.Len(); i++ {
		if i > 0 {
			buf.WriteString(", ")
		}

		s2 := valuesValue.Index(i).String()
		buf.WriteString(s2)
	}

	if !found {
		c.AddError(token, "invalid_value", "%s", buf.String())
	}

	return found
}

func (c *Checker) CheckStringMatch(token interface{}, s string, re *regexp.Regexp) bool {
	return c.CheckStringMatch2(token, s, re, "invalid_string_format",
		"string must match the following regular expression: %s",
		re.String())
}

func (c *Checker) CheckStringMatch2(token interface{}, s string, re *regexp.Regexp, code, format string, args ...interface{}) bool {
	if !re.MatchString(s) {
		c.AddError(token, code, format, args...)
		return false
	}

	return true
}

func (c *Checker) CheckStringURI(token interface{}, s string) bool {
	// The url.Parse function considers that the empty string is a valid URL.
	// It is not.

	if s == "" {
		c.AddError(token, "empty_uri", "string must be a valid uri")
		return false
	} else if _, err := url.Parse(s); err != nil {
		c.AddError(token, "invalid_uri_format", "string must be a valid uri")
		return false
	}

	return true
}

func (c *Checker) CheckArrayLengthMin(token interface{}, value interface{}, min int) bool {
	var length int

	checkArray(value, &length)

	return c.Check(token, length >= min, "array_too_small",
		"array must contain %d or more elements", min)
}

func (c *Checker) CheckArrayLengthMax(token interface{}, value interface{}, max int) bool {
	var length int

	checkArray(value, &length)

	return c.Check(token, length <= max, "array_too_large",
		"array must contain %d or less elements", max)
}

func (c *Checker) CheckArrayLengthMinMax(token interface{}, value interface{}, min, max int) bool {
	if !c.CheckArrayLengthMin(token, value, min) {
		return false
	}

	return c.CheckArrayLengthMax(token, value, max)
}

func (c *Checker) CheckArrayNotEmpty(token interface{}, value interface{}) bool {
	var length int

	checkArray(value, &length)

	return c.Check(token, length > 0, "empty_array", "array must not be empty")
}

func checkArray(value interface{}, plen *int) {
	valueType := reflect.TypeOf(value)

	switch valueType.Kind() {
	case reflect.Slice:
		*plen = reflect.ValueOf(value).Len()

	case reflect.Array:
		*plen = valueType.Len()

	default:
		panicf("value is not a slice or array")
	}
}

func (c *Checker) CheckOptionalObject(token interface{}, value interface{}) bool {
	var isNil bool
	checkObject(value, &isNil)

	if isNil {
		return true
	}

	return c.doCheckObject(token, value)
}

func (c *Checker) CheckObject(token interface{}, value interface{}) bool {
	var isNil bool
	checkObject(value, &isNil)

	if !c.Check(token, !isNil, "missing_value", "missing value") {
		return false
	}

	return c.doCheckObject(token, value)
}

func (c *Checker) doCheckObject(token interface{}, value interface{}) bool {
	nbErrors := len(c.Errors)

	obj, ok := value.(Object)
	if !ok {
		panicf("value %#v (%T) does not implement Object", value, value)
	}

	c.WithChild(token, func() {
		obj.Check(c)
	})

	return len(c.Errors) == nbErrors
}

func (c *Checker) CheckObjectArray(token interface{}, value interface{}) bool {
	valueType := reflect.TypeOf(value)
	kind := valueType.Kind()

	if kind != reflect.Array && kind != reflect.Slice {
		panicf("value %#v (%T) is not an array or slice", value, value)
	}

	ok := true

	c.WithChild(token, func() {
		values := reflect.ValueOf(value)

		for i := 0; i < values.Len(); i++ {
			child := values.Index(i).Interface()
			childOk := c.CheckObject(strconv.Itoa(i), child)
			ok = ok && childOk
		}
	})

	return ok
}

func (c *Checker) CheckObjectMap(token interface{}, value interface{}) bool {
	valueType := reflect.TypeOf(value)
	if valueType.Kind() != reflect.Map {
		panicf("value %#v (%T) is not a map", value, value)
	}

	ok := true

	c.WithChild(token, func() {
		values := reflect.ValueOf(value)

		iter := values.MapRange()
		for iter.Next() {
			key := iter.Key()
			if key.Kind() != reflect.String {
				panicf("value %#v (%T) is a map whose keys are not strings",
					value, value)
			}
			keyString := key.Interface().(string)

			value := iter.Value().Interface()

			valueOk := c.CheckObject(keyString, value)
			ok = ok && valueOk
		}
	})

	return ok
}

func checkObject(value interface{}, pnil *bool) {
	valueType := reflect.TypeOf(value)
	if valueType == nil {
		*pnil = true
		return
	}

	if valueType.Kind() != reflect.Pointer {
		panicf("value %#v (%T) is not a pointer", value, value)
	}

	pointedValueType := valueType.Elem()
	if pointedValueType.Kind() != reflect.Struct {
		panicf("value %#v (%T) is not an object pointer", value, value)
	}

	*pnil = reflect.ValueOf(value).IsZero()
}

func pointerAppend(p djson.Pointer, token interface{}) djson.Pointer {
	switch v := token.(type) {
	case string:
		return append(p, v)

	case int:
		return append(p, strconv.Itoa(v))

	case djson.Pointer:
		return append(p, v...)
	}

	panicf("invalid token %#v (%T)", token, token)
	return nil // the Go compiler cannot infer that panic() never returns...
}

func panicf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}
