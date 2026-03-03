package config

import (
	"errors"
	"fmt"
)

var errNonPositiveTimeout = errors.New("must be a positive integer")

// InvalidEnvError is returned when an environment variable contains an invalid value.
type InvalidEnvError struct {
	Err   error
	Name  string
	Value string
}

func (e *InvalidEnvError) Error() string {
	return fmt.Sprintf("config: invalid env %s=%q: %v", e.Name, e.Value, e.Err)
}

func (e *InvalidEnvError) Unwrap() error {
	return e.Err
}
