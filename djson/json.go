package djson

import "fmt"

type InvalidValueError struct {
	Value interface{}
}

func (err *InvalidValueError) Error() string {
	return fmt.Sprintf("%#v (%T) is not a valid json value",
		err.Value, err.Value)
}

type Value interface{}

func IsNumber(v Value) bool {
	_, ok := v.(float64)
	return ok
}

func IsString(v Value) bool {
	_, ok := v.(string)
	return ok
}

func IsBoolean(v Value) bool {
	_, ok := v.(bool)
	return ok
}

func IsArray(v Value) bool {
	_, ok := v.([]Value)
	return ok
}

func IsObject(v Value) bool {
	_, ok := v.(map[string]Value)
	return ok
}

func AsNumber(v Value) float64 {
	return v.(float64)
}

func AsString(v Value) string {
	return v.(string)
}

func AsBoolean(v Value) bool {
	return v.(bool)
}

func AsArray(v Value) []Value {
	return v.([]Value)
}

func AsObject(v Value) map[string]Value {
	return v.(map[string]Value)
}

func Equal(v1, v2 Value) bool {
	switch {
	case IsNumber(v1) && IsNumber(v2):
		return AsNumber(v1) == AsNumber(v2)

	case IsString(v1) && IsString(v2):
		return AsString(v1) == AsString(v2)

	case IsBoolean(v1) && IsBoolean(v2):
		return AsBoolean(v1) == AsBoolean(v2)

	case IsArray(v1) && IsArray(v2):
		a1 := AsArray(v1)
		a2 := AsArray(v2)

		if len(a1) != len(a2) {
			return false
		}

		for i := 0; i < len(a1); i++ {
			if !Equal(a1[i], a2[i]) {
				return false
			}
		}

		return true

	case IsObject(v1) && IsObject(v2):
		obj1 := AsObject(v1)
		obj2 := AsObject(v2)

		for key, value1 := range obj1 {
			value2, found := obj2[key]
			if !found || !Equal(value1, value2) {
				return false
			}
		}

		for key, value2 := range obj2 {
			value1, found := obj1[key]
			if !found || !Equal(value1, value2) {
				return false
			}
		}

		return true
	}

	return false
}

func ObjectKeys(v Value) []string {
	obj := AsObject(v)

	keys := make([]string, len(obj))

	i := 0
	for key := range obj {
		keys[i] = key
		i++
	}

	return keys
}

func ObjectValues(v Value) []Value {
	obj := AsObject(v)

	values := make([]Value, len(obj))

	i := 0
	for _, value := range obj {
		values[i] = value
		i++
	}

	return values
}
