package user

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewUserEnv(t *testing.T) {
	actual := NewUserEnv()

	assert.NotNil(t, actual)
	assert.IsType(t, &UserEnv{}, actual)
}

func TestPopulate(t *testing.T) {
	cases := []struct {
		description string
		given       func() string
		clean       func()
		expected    *UserEnv
		error       bool
		message     string
	}{
		{
			"not using configuration file",
			func() string {
				os.Setenv("EXPIRATION_DATE", "2023-01-01")
				os.Setenv("AUTHORIZED_USERS", "test")
				return ""
			},
			os.Clearenv,
			&UserEnv{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Users: []string{"test"}},
			false,
			``,
		},
		{
			"not using configuration file with no environment variables set",
			func() string {
				// No-op.
				return ""
			},
			os.Clearenv,
			&UserEnv{},
			true,
			`unable to access environment variable: EXPIRATION_DATE`,
		},
		{
			"not using configuration file with invalid expiration date",
			func() string {
				os.Setenv("EXPIRATION_DATE", "test")
				return ""
			},
			os.Clearenv,
			&UserEnv{},
			true,
			`unable to parse expiration date`,
		},
		{
			"not using configuration file with expiration date and no users set",
			func() string {
				os.Setenv("EXPIRATION_DATE", "2023-01-01")
				return ""
			},
			os.Clearenv,
			&UserEnv{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
			false,
			``,
		},
		{
			"not using configuration file with expiration date and empty users set",
			func() string {
				os.Setenv("EXPIRATION_DATE", "2023-01-01")
				os.Setenv("AUTHORIZED_USERS", "")
				return ""
			},
			os.Clearenv,
			&UserEnv{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
			false,
			``,
		},
		{
			"using configuration file with expiration date and empty users set",
			func() string {
				file, err := os.CreateTemp("", "user-")
				if err != nil {
					t.Fatal(err)
				}
				_, err = file.WriteString(`{"expiration":"2023-01-01", "users":[]}`)
				if err != nil {
					t.Fatal(err)
				}
				os.Setenv("CONFIG_FILE_PATH", file.Name())
				return file.Name()
			},
			os.Clearenv,
			&UserEnv{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
			false,
			``,
		},
		{
			"using configuration file with expiration date and users set",
			func() string {
				file, err := os.CreateTemp("", "user-")
				if err != nil {
					t.Fatal(err)
				}
				_, err = file.WriteString(`{"expiration":"2023-01-01", "users":["test"]}`)
				if err != nil {
					t.Fatal(err)
				}
				os.Setenv("CONFIG_FILE_PATH", file.Name())
				return file.Name()
			},
			os.Clearenv,
			&UserEnv{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Users: []string{"test"}},
			false,
			``,
		},
		{
			"using configuration file and environment variables with expiration date and users set",
			func() string {
				file, err := os.CreateTemp("", "user-")
				if err != nil {
					t.Fatal(err)
				}
				_, err = file.WriteString(`{"expiration":"2023-01-01", "users":["test"]}`)
				if err != nil {
					t.Fatal(err)
				}
				os.Setenv("CONFIG_FILE_PATH", file.Name())
				os.Setenv("AUTHORIZED_USERS", "test2")
				return file.Name()
			},
			os.Clearenv,
			&UserEnv{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Users: []string{"test2"}},
			false,
			``,
		},
		{
			"legacy users file support",
			func() string {
				file, err := os.CreateTemp("", "user-")
				if err != nil {
					t.Fatal(err)
				}
				_, err = file.WriteString(`test`)
				if err != nil {
					t.Fatal(err)
				}
				os.Setenv("USERS_FILE_PATH", file.Name())
				return file.Name()
			},
			os.Clearenv,
			&UserEnv{Users: []string{"test"}},
			false,
			``,
		},
		{
			"empty legacy users file support",
			func() string {
				file, err := os.CreateTemp("", "user-")
				if err != nil {
					t.Fatal(err)
				}
				os.Setenv("USERS_FILE_PATH", file.Name())
				return file.Name()
			},
			os.Clearenv,
			&UserEnv{},
			false,
			``,
		},
		{
			"invalid configuration file",
			func() string {
				os.Setenv("CONFIG_FILE_PATH", "test")
				return ""
			},
			os.Clearenv,
			&UserEnv{},
			true,
			`unable to read users file: open test`,
		},
		{
			"invalid legacy users file",
			func() string {
				os.Setenv("USERS_FILE_PATH", "test")
				return ""
			},
			os.Clearenv,
			&UserEnv{},
			true,
			`unable to read users file`,
		},
		{
			"invalid configuration file JSON content",
			func() string {
				file, err := os.CreateTemp("", "user-")
				if err != nil {
					t.Fatal(err)
				}
				_, err = file.WriteString(`{"expiration":"2023-01-01", "users:["test"]}`)
				if err != nil {
					t.Fatal(err)
				}
				os.Setenv("CONFIG_FILE_PATH", file.Name())
				return file.Name()
			},
			os.Clearenv,
			&UserEnv{},
			true,
			`unable to unmarshal users file`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			tc.clean()
			defer os.Remove(tc.given())

			actual := &UserEnv{}
			err := actual.Populate()

			if tc.error {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), tc.message)
			} else {
				assert.Nil(t, err)
			}

			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestIsDeprecated(t *testing.T) {
	cases := []struct {
		description string
		given       UserEnv
		expected    bool
	}{
		{
			"not deprecated with expiration date set",
			UserEnv{Expiration: time.Now()},
			false,
		},
		{
			"deprecated with default expiration date value set",
			UserEnv{Expiration: time.Time{}},
			true,
		},
		{
			"deprecated with nothing set",
			UserEnv{},
			true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			actual := tc.given.IsDeprecated()

			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestIsExpired(t *testing.T) {
	cases := []struct {
		description string
		given       UserEnv
		expected    bool
	}{
		{
			"before expiration date",
			UserEnv{Expiration: time.Now().AddDate(0, 0, 1)},
			false,
		},
		{
			"past expiration date",
			UserEnv{Expiration: time.Now().AddDate(0, 0, -1)},
			true,
		},
		{
			"invalid expiration date",
			UserEnv{},
			true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			actual := tc.given.IsExpired()

			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestMarshalJSON(t *testing.T) {
	cases := []struct {
		description string
		given       UserEnv
		expected    string
	}{
		{
			"users and expiration date",
			UserEnv{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Users: []string{"test"}},
			`{"users":["test"],"expiration":"2023-01-01"}`,
		},
		{
			"no users and no expiration date",
			UserEnv{},
			`{"users":[],"expiration":"0001-01-01"}`,
		},
		{
			"empty users and empty expiration date",
			UserEnv{Users: []string{}, Expiration: time.Time{}},
			`{"users":[],"expiration":"0001-01-01"}`,
		},
		{
			"empty users and no expiration date",
			UserEnv{Users: []string{}},
			`{"users":[],"expiration":"0001-01-01"}`,
		},
		{
			"users and no expiration date",
			UserEnv{Users: []string{"test"}},
			`{"users":["test"],"expiration":"0001-01-01"}`,
		},
		{
			"no users and expiration date",
			UserEnv{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
			`{"users":[],"expiration":"2023-01-01"}`,
		},
		{
			"empty users and expiration date",
			UserEnv{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Users: []string{}},
			`{"users":[],"expiration":"2023-01-01"}`,
		},
		{
			"users and empty expiration date",
			UserEnv{Users: []string{"test"}, Expiration: time.Time{}},
			`{"users":["test"],"expiration":"0001-01-01"}`,
		},
		{
			"no users and empty expiration date",
			UserEnv{Expiration: time.Time{}},
			`{"users":[],"expiration":"0001-01-01"}`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			results, err := tc.given.MarshalJSON()

			assert.Nil(t, err)
			assert.Equal(t, tc.expected, string(results))
		})
	}
}

func TestUnmarshalJSON(t *testing.T) {
	cases := []struct {
		description string
		given       string
		expected    UserEnv
		error       bool
		message     string
	}{
		{
			"valid JSON with users and expiration date",
			`{"users":["test"],"expiration":"2023-01-01"}`,
			UserEnv{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Users: []string{"test"}},
			false,
			``,
		},
		{
			"valid JSON with users and expiration date set to null",
			`{"users":null,"expiration":null}`,
			UserEnv{},
			true,
			`unable to parse expiration date`,
		},
		{
			"valid JSON with users and expiration set to empty values",
			`{"users":[],"expiration":""}`,
			UserEnv{},
			true,
			`unable to parse expiration date`,
		},
		{
			"valid JSON with users set to null without expiration date",
			`{"users":null}`,
			UserEnv{},
			true,
			`unable to find expiration date`,
		},
		{
			"valid JSON with users set to empty value without expiration date",
			`{"users":[]}`,
			UserEnv{},
			true,
			`unable to find expiration date`,
		},
		{
			"valid JSON with user set to invalid value",
			`{"users":["test", -1], "expiration": "2023-01-01"}`,
			UserEnv{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Users: []string{"test"}},
			true,
			`unable to parse user`,
		},
		{
			"valid JSON without users with expiration date",
			`{"expiration":"2023-01-01"}`,
			UserEnv{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
			true,
			`unable to find users list`,
		},
		{
			"valid JSON without users with expiration date set to null",
			`{"expiration":null}`,
			UserEnv{},
			true,
			`unable to parse expiration date`,
		},
		{
			"valid JSON with users set to null with expiration date",
			`{"users":null,"expiration":"2023-01-01"}`,
			UserEnv{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
			true,
			`unable to parse users list`,
		},
		{
			"invalid empty string with nothing set",
			``,
			UserEnv{},
			true,
			`unable to unmarshal user file`,
		},
		{
			"invalid empty JSON object with nothing set",
			`{}`,
			UserEnv{},
			true,
			`unable to find expiration date`,
		},
		{
			"invalid JSON",
			`{"users:[], "expiration": "2023-01-01"}`,
			UserEnv{},
			true,
			`unable to unmarshal user file`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			results := UserEnv{}
			err := results.UnmarshalJSON([]byte(tc.given))

			if tc.error {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), tc.message)
			} else {
				assert.Nil(t, err)
			}

			assert.Equal(t, tc.expected, results)
		})
	}
}
