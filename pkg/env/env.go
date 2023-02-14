package env

import "fmt"

type Error struct {
	Name string
}

func (e *Error) Error() string {
	return fmt.Sprintf("unable to access environment variable: %s", e.Name)
}

type TypeError struct {
	Name string
}

func (e *TypeError) Error() string {
	return fmt.Sprintf("unable to convert environment variable: %s", e.Name)
}
