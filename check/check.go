package check

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/exograd/go-daemon/jsonpointer"
)

type Checker struct {
	Path   jsonpointer.Pointer
	Errors ValidationErrors
}

type Object interface {
	Check(*Checker)
}

type ValidationError struct {
	Path    jsonpointer.Pointer `json:"path"`
	Message string              `json:"message"`
}

type ValidationErrors []*ValidationError

func (err ValidationError) String() string {
	return fmt.Sprintf("ValidationError{%v, %q}", err.Path, err.Message)
}

func (err ValidationError) GoString() string {
	return err.String()
}

func (err ValidationError) Error() string {
	return fmt.Sprintf("%v: %s", err.Path, err.Message)
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

func (c *Checker) Push(pathSegment string) {
	c.Path = append(c.Path, pathSegment)
}

func (c *Checker) Pop() {
	c.Path = c.Path[:len(c.Path)-1]
}

func (c *Checker) Check(pathSegment string, v bool, format string, args ...interface{}) bool {
	if !v {
		path := []string{}
		path = append(path, c.Path...)
		path = append(path, pathSegment)

		err := ValidationError{
			Path:    path,
			Message: fmt.Sprintf(format, args...),
		}

		c.Errors = append(c.Errors, &err)
	}

	return v
}

func (c *Checker) CheckOptionalObject(pathSegment string, value interface{}) bool {
	valueType := reflect.TypeOf(value)
	if valueType.Kind() != reflect.Pointer {
		panic(fmt.Sprintf("value is not a pointer"))
	}

	pointedValueType := valueType.Elem()
	if pointedValueType.Kind() != reflect.Struct {
		panic(fmt.Sprintf("value is not an object pointer"))
	}

	return c.doCheckObject(pathSegment, value)
}

func (c *Checker) CheckObject(pathSegment string, value interface{}) bool {
	valueType := reflect.TypeOf(value)

	switch valueType.Kind() {
	case reflect.Struct:
		return c.doCheckObject(pathSegment, value)

	case reflect.Pointer:
		pointedValueType := valueType.Elem()
		if pointedValueType.Kind() != reflect.Struct {
			panic(fmt.Sprintf("value is not an object pointer"))
		}

		isNil := reflect.ValueOf(value).IsZero()
		if !c.Check(pathSegment, !isNil, "missing value") {
			return false
		}

		return c.CheckOptionalObject(pathSegment, value)

	default:
		panic(fmt.Sprintf("value is neither a pointer nor a structure"))
	}
}

func (c *Checker) doCheckObject(pathSegment string, value interface{}) bool {
	nbErrors := len(c.Errors)

	if obj, ok := value.(Object); ok {
		c.Push(pathSegment)
		defer c.Pop()

		obj.Check(c)
	}

	return len(c.Errors) == nbErrors
}

func (c *Checker) CheckIntMin(pathSegment string, i, min int) bool {
	return c.Check(pathSegment, i >= min,
		"integer %d must be greater or equal to %d", i, min)
}

func (c *Checker) CheckIntMax(pathSegment string, i, max int) bool {
	return c.Check(pathSegment, i <= max,
		"integer %d must be lower or equal to %d", i, max)
}

func (c *Checker) CheckIntMinMax(pathSegment string, i, min, max int) bool {
	if !c.CheckIntMin(pathSegment, i, min) {
		return false
	}

	return c.CheckIntMax(pathSegment, i, max)
}

func (c *Checker) CheckStringLengthMin(pathSegment string, s string, min int) bool {
	return c.Check(pathSegment, len(s) >= min,
		"string length must be greater or equal to %d", min)
}

func (c *Checker) CheckStringLengthMax(pathSegment string, s string, max int) bool {
	return c.Check(pathSegment, len(s) <= max,
		"string length must be lower or equal to %d", max)
}

func (c *Checker) CheckStringLengthMinMax(pathSegment string, s string, min, max int) bool {
	if !c.CheckStringLengthMin(pathSegment, s, min) {
		return false
	}

	return c.CheckStringLengthMax(pathSegment, s, max)
}

func (c *Checker) CheckStringNotEmpty(pathSegment string, s string) bool {
	return c.Check(pathSegment, s != "",
		"string must not be empty")
}
