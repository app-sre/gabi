package user

import (
	"bufio"
	"os"
	"time"

	"github.com/app-sre/gabi/pkg/env"
)

type Userenv struct {
	Users []string
	Expiration time.Time
}

func (usere *Userenv) Populate() error {
	path, found := os.LookupEnv("USERS_FILE_PATH")
	if !(found) {
		return &env.EnvError{Env: "USERS_FILE_PATH"}
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// optionally, resize scanner's capacity for lines over 64K, see next example
	for scanner.Scan() {
		usere.Users = append(usere.Users, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	expiration, found := os.LookupEnv("INSTANCE_EXPIRATION")
	if !(found) {
		return &env.EnvError{Env: "INSTANCE_EXPIRATION"}
	}
	if parsed_expiration, err := time.Parse("2006-01-02", expiration); err != nil {
		return err
	} else {
		usere.Expiration = parsed_expiration
	}

	return nil
}
