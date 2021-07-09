package env

import "fmt"

type EnvError struct{ Env string }

func (e *EnvError) Error() string {
	return fmt.Sprintf("Missing required environment variable: %s", e.Env)
}

type EnvConvError struct{ Env string }

func (e *EnvConvError) Error() string {
	return fmt.Sprintf("Failed to perform type conversion on required environment variable: %s", e.Env)
}
