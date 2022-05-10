package check

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"

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

func (c *Checker) CheckOptionalObject(token string, value interface{}) bool {
	valueType := reflect.TypeOf(value)
	if valueType.Kind() != reflect.Pointer {
		panic(fmt.Sprintf("value is not a pointer"))
	}

	pointedValueType := valueType.Elem()
	if pointedValueType.Kind() != reflect.Struct {
		panic(fmt.Sprintf("value is not an object pointer"))
	}

	return c.doCheckObject(token, value)
}

func (c *Checker) CheckObject(token string, value interface{}) bool {
	valueType := reflect.TypeOf(value)

	switch valueType.Kind() {
	case reflect.Struct:
		return c.doCheckObject(token, value)

	case reflect.Pointer:
		pointedValueType := valueType.Elem()
		if pointedValueType.Kind() != reflect.Struct {
			panic(fmt.Sprintf("value is not an object pointer"))
		}

		isNil := reflect.ValueOf(value).IsZero()
		if !c.Check(token, !isNil, "missing value") {
			return false
		}

		return c.CheckOptionalObject(token, value)

	default:
		panic(fmt.Sprintf("value is neither a pointer nor a structure"))
	}
}

func (c *Checker) doCheckObject(token string, value interface{}) bool {
	nbErrors := len(c.Errors)

	if obj, ok := value.(Object); ok {
		c.Push(token)
		defer c.Pop()

		obj.Check(c)
	}

	return len(c.Errors) == nbErrors
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

func (c *Checker) CheckStringValue(token string, s string, values []string) bool {
	found := false
	for _, v := range values {
		if v == s {
			found = true
		}
	}

	var buf bytes.Buffer

	buf.WriteString("value must be one of the following strings: ")

	for i, v := range values {
		if i > 0 {
			buf.WriteString(", ")
		}

		buf.WriteString(v)
	}

	if !found {
		c.AddError(token, "%s", buf.String())
	}

	return found
}

func (c *Checker) CheckStringMatch(token string, s string, re *regexp.Regexp) bool {
	if !re.MatchString(s) {
		c.AddError(token,
			"string must match the following regular expression: %s",
			re.String())

		return false
	}

	return true
}
