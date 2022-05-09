package check

import (
	"fmt"
	"reflect"
)

type Checker struct {
	Path   []string
	Errors []*ValidationError
}

type Object interface {
	Check(*Checker) bool
}

type ValidationError struct {
	Path    []string
	Message string
}

func (err ValidationError) String() string {
	return fmt.Sprintf("ValidationError{%v, %q}", err.Path, err.Message)
}

func (err ValidationError) GoString() string {
	return err.String()
}

func (err ValidationError) Error() string {
	return fmt.Sprintf("%v: %s", err.Path, err.Message) // TODO json path
}

func NewChecker() *Checker {
	return &Checker{}
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

	if obj, ok := value.(Object); ok {
		c.Push(pathSegment)
		defer c.Pop()

		return obj.Check(c)
	}

	return true
}

func (c *Checker) CheckObject(pathSegment string, value interface{}) bool {
	valueType := reflect.TypeOf(value)

	switch valueType.Kind() {
	case reflect.Struct:
		if obj, ok := value.(Object); ok {
			c.Push(pathSegment)
			defer c.Pop()

			return obj.Check(c)
		}
		return true

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

func (c *Checker) CheckIntMin(pathSegment string, i, min int) bool {
	return c.Check(pathSegment, i >= min,
		"value %d must be greater or equal to %d", i, min)
}

func (c *Checker) CheckIntMax(pathSegment string, i, max int) bool {
	return c.Check(pathSegment, i <= max,
		"value %d must be lower or equal to %d", i, max)
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
