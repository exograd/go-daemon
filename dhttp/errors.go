package dhttp

import "fmt"

type InvalidQueryParameterError struct {
	Name    string
	Message string
}

func NewInvalidQueryParameterError(name, format string, args ...interface{}) *InvalidQueryParameterError {
	return &InvalidQueryParameterError{
		Name:    name,
		Message: fmt.Sprintf(format, args...),
	}
}

func (err InvalidQueryParameterError) Error() string {
	return fmt.Sprintf("invalid query parameter %q: %s", err.Name, err.Message)
}
