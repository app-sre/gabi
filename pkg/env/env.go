package env

import "fmt"

type EnvError struct {
	Name string
}

func (e *EnvError) Error() string {
	return fmt.Sprintf("unable to access environment variable: %s", e.Name)
}

type EnvTypeError struct {
	Name string
}

func (e *EnvTypeError) Error() string {
	return fmt.Sprintf("unable to convert environment variable: %s", e.Name)
}
