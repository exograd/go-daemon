package check

import (
	"bytes"
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strconv"

	"github.com/exograd/go-daemon/jsonpointer"
)

type Checker struct {
	Pointer jsonpointer.Pointer
	Errors  ValidationErrors
}

type Object interface {
	Check(*Checker)
}

type ValidationError struct {
	Pointer jsonpointer.Pointer `json:"pointer"`
	Message string              `json:"message"`
}

type ValidationErrors []*ValidationError

func (err ValidationError) String() string {
	return fmt.Sprintf("ValidationError{%v, %q}", err.Pointer, err.Message)
}

func (err ValidationError) GoString() string {
	return err.String()
}

func (err ValidationError) Error() string {
	return fmt.Sprintf("%v: %s", err.Pointer, err.Message)
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

func (c *Checker) Push(token string) {
	c.Pointer = append(c.Pointer, token)
}

func (c *Checker) Pop() {
	c.Pointer = c.Pointer[:len(c.Pointer)-1]
}

func (c *Checker) WithChild(tokenOrIndex interface{}, fn func()) {
	var token string
	switch v := tokenOrIndex.(type) {
	case string:
		token = v
	case int:
		token = strconv.Itoa(v)
	}

	c.Push(token)
	defer c.Pop()

	fn()
}

func (c *Checker) AddError(token string, format string, args ...interface{}) {
	var pointer jsonpointer.Pointer
	pointer = append(pointer, c.Pointer...)
	pointer.Append(token)

	err := ValidationError{
		Pointer: pointer,
		Message: fmt.Sprintf(format, args...),
	}

	c.Errors = append(c.Errors, &err)
}

func (c *Checker) Check(token string, v bool, format string, args ...interface{}) bool {
	if !v {
		c.AddError(token, format, args...)
	}

	return v
}

func (c *Checker) CheckIntMin(token string, i, min int) bool {
	return c.Check(token, i >= min,
		"integer %d must be greater or equal to %d", i, min)
}

func (c *Checker) CheckIntMax(token string, i, max int) bool {
	return c.Check(token, i <= max,
		"integer %d must be lower or equal to %d", i, max)
}

func (c *Checker) CheckIntMinMax(token string, i, min, max int) bool {
	if !c.CheckIntMin(token, i, min) {
		return false
	}

	return c.CheckIntMax(token, i, max)
}

func (c *Checker) CheckStringLengthMin(token string, s string, min int) bool {
	return c.Check(token, len(s) >= min,
		"string length must be greater or equal to %d", min)
}

func (c *Checker) CheckStringLengthMax(token string, s string, max int) bool {
	return c.Check(token, len(s) <= max,
		"string length must be lower or equal to %d", max)
}

func (c *Checker) CheckStringLengthMinMax(token string, s string, min, max int) bool {
	if !c.CheckStringLengthMin(token, s, min) {
		return false
	}

	return c.CheckStringLengthMax(token, s, max)
}

func (c *Checker) CheckStringNotEmpty(token string, s string) bool {
	return c.Check(token, s != "",
		"string must not be empty")
}

func (c *Checker) CheckStringValue(token string, value interface{}, values interface{}) bool {
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
		c.AddError(token, "%s", buf.String())
	}

	return found
}

func (c *Checker) CheckStringMatch(token string, s string, re *regexp.Regexp) bool {
	return c.CheckStringMatch2(token, s, re,
		"string must match the following regular expression: %s",
		re.String())
}

func (c *Checker) CheckStringMatch2(token string, s string, re *regexp.Regexp, format string, args ...interface{}) bool {
	if !re.MatchString(s) {
		c.AddError(token, format, args...)
		return false
	}

	return true
}

func (c *Checker) CheckStringURI(token string, s string) bool {
	// The url.Parse function considers that the empty string is a valid URL.
	// It is not.

	if s == "" {
		c.AddError(token, "string must be a valid uri")
		return false
	} else if _, err := url.Parse(s); err != nil {
		c.AddError(token, "string must be a valid uri")
		return false
	}

	return true
}

func (c *Checker) CheckArrayLengthMin(token string, value interface{}, min int) bool {
	var length int

	checkArray(value, &length)

	return c.Check(token, length >= min,
		"array must contain %d or more elements", min)
}

func (c *Checker) CheckArrayLengthMax(token string, value interface{}, max int) bool {
	var length int

	checkArray(value, &length)

	return c.Check(token, length <= max,
		"array must contain %d or less elements", max)
}

func (c *Checker) CheckArrayLengthMinMax(token string, value interface{}, min, max int) bool {
	if !c.CheckArrayLengthMin(token, value, min) {
		return false
	}

	return c.CheckArrayLengthMax(token, value, max)
}

func (c *Checker) CheckArrayNotEmpty(token string, value interface{}) bool {
	var length int

	checkArray(value, &length)

	return c.Check(token, length > 0, "array must not be empty")
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

func (c *Checker) CheckOptionalObject(token string, value interface{}) bool {
	var isNil bool
	checkObject(value, &isNil)

	if isNil {
		return true
	}

	return c.doCheckObject(token, value)
}

func (c *Checker) CheckObject(token string, value interface{}) bool {
	var isNil bool
	checkObject(value, &isNil)

	if !c.Check(token, !isNil, "missing value") {
		return false
	}

	return c.doCheckObject(token, value)
}

func (c *Checker) doCheckObject(token string, value interface{}) bool {
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

func (c *Checker) CheckObjectArray(token string, value interface{}) bool {
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

func (c *Checker) CheckObjectMap(token string, value interface{}) bool {
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

func panicf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}
