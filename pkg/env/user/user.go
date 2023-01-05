package user

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/app-sre/gabi/pkg/env"
)

const ExpiryDateLayout = "2006-01-02"

type UserEnv struct {
	Expiration time.Time `json:"expiration"`
	Users      []string  `json:"users"`
}

func NewUserEnv() *UserEnv {
	return &UserEnv{}
}

func (u *UserEnv) Populate() error {
	if path := os.Getenv("USERS_FILE_PATH"); path != "" {
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("unable to read users file: %w", err)
		}
		defer func() { _ = file.Close() }()

		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			if s := strings.Trim(scanner.Text(), " "); s != "" {
				u.Users = append(u.Users, s)
			}
		}

		return nil
	}

	if path := os.Getenv("CONFIG_FILE_PATH"); path != "" {
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("unable to read users file: %w", err)
		}

		if err := json.Unmarshal(content, &u); err != nil {
			return fmt.Errorf("unable to unmarshal users file: %w", err)
		}
	}

	if expiration := os.Getenv("EXPIRATION_DATE"); expiration != "" {
		t, err := time.Parse(ExpiryDateLayout, expiration)
		if err != nil {
			return fmt.Errorf("unable to parse expiration date: %w", err)
		}
		u.Expiration = t
	}
	if u.Expiration == (time.Time{}) {
		return &env.EnvError{Name: "EXPIRATION_DATE"}
	}

	if users := os.Getenv("AUTHORIZED_USERS"); users != "" {
		ss := strings.Split(users, ",")
		copy := make([]string, len(ss))

		i := 0
		for _, entry := range ss {
			if s := strings.Trim(entry, " "); s != "" {
				copy[i] = s
				i++
			}
		}
		u.Users = copy[0:i]
	}

	return nil
}

func (u *UserEnv) IsDeprecated() bool {
	return u.Expiration == (time.Time{})
}

func (u *UserEnv) IsExpired() bool {
	return u.Expiration.Before(time.Now())
}

func (u *UserEnv) MarshalJSON() ([]byte, error) {
	type alias UserEnv

	aux := &struct {
		*alias
		Expiration string `json:"expiration"`
	}{
		alias:      (*alias)(u),
		Expiration: u.Expiration.Format(ExpiryDateLayout),
	}
	aux.Users = append([]string{}, u.Users...)

	json, err := json.Marshal(aux)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal user data: %w", err)
	}

	return json, nil
}

func (u *UserEnv) UnmarshalJSON(b []byte) error {
	var raw map[string]any

	err := json.Unmarshal(b, &raw)
	if err != nil {
		return fmt.Errorf("unable to unmarshal user file: %w", err)
	}

	if _, found := raw["expiration"]; !found {
		return errors.New("unable to find expiration date")
	}
	s, ok := raw["expiration"].(string)
	if !ok {
		return fmt.Errorf("unable to parse expiration date: %v", raw["expiration"])
	}

	expiration, err := time.Parse(ExpiryDateLayout, s)
	if err != nil {
		return fmt.Errorf("unable to parse expiration date: %w", err)
	}
	u.Expiration = expiration

	if _, found := raw["users"]; !found {
		return errors.New("unable to find users list")
	}
	users, ok := raw["users"].([]any)
	if !ok {
		return fmt.Errorf("unable to parse users list: %v", raw["users"])
	}

	var copy []string

	for _, v := range users {
		user, ok := v.(string)
		if !ok {
			return fmt.Errorf("unable to parse user: %v", v)
		}
		copy = append(copy, strings.Trim(user, " "))
	}
	u.Users = copy

	return nil
}
