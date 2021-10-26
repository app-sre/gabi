package user

import (
	"bufio"
	"os"

	"github.com/app-sre/gabi/pkg/env"
)

type Userenv struct {
	Users []string
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

	return nil
}
