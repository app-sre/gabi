package user

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUserEnv(t *testing.T) {
	t.Parallel()

	actual := NewUserEnv()

	require.NotNil(t, actual)
	assert.IsType(t, &Env{}, actual)
}

func TestPopulate(t *testing.T) {
	cases := []struct {
		description string
		given       func() string
		expected    *Env
		error       bool
		want        string
	}{
		{
			"not using configuration file",
			func() string {
				t.Setenv("EXPIRATION_DATE", "2023-01-01")
				t.Setenv("AUTHORIZED_USERS", "test")
				return ""
			},
			&Env{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Users: []string{"test"}},
			false,
			``,
		},
		{
			"not using configuration file with no environment variables set",
			func() string {
				return ""
			},
			&Env{},
			true,
			`unable to access environment variable: EXPIRATION_DATE`,
		},
		{
			"not using configuration file with invalid expiration date",
			func() string {
				t.Setenv("EXPIRATION_DATE", "test")
				return ""
			},
			&Env{},
			true,
			`unable to parse expiration date`,
		},
		{
			"not using configuration file with expiration date and no users set",
			func() string {
				t.Setenv("EXPIRATION_DATE", "2023-01-01")
				return ""
			},
			&Env{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
			false,
			``,
		},
		{
			"not using configuration file with expiration date and empty users set",
			func() string {
				t.Setenv("EXPIRATION_DATE", "2023-01-01")
				t.Setenv("AUTHORIZED_USERS", "")
				return ""
			},
			&Env{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
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
				t.Setenv("CONFIG_FILE_PATH", file.Name())
				return file.Name()
			},
			&Env{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
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
				t.Setenv("CONFIG_FILE_PATH", file.Name())
				return file.Name()
			},
			&Env{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Users: []string{"test"}},
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
				t.Setenv("CONFIG_FILE_PATH", file.Name())
				t.Setenv("AUTHORIZED_USERS", "test2")
				return file.Name()
			},
			&Env{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Users: []string{"test2"}},
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
				t.Setenv("USERS_FILE_PATH", file.Name())
				return file.Name()
			},
			&Env{Users: []string{"test"}},
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
				t.Setenv("USERS_FILE_PATH", file.Name())
				return file.Name()
			},
			&Env{},
			false,
			``,
		},
		{
			"invalid configuration file",
			func() string {
				t.Setenv("CONFIG_FILE_PATH", "test")
				return ""
			},
			&Env{},
			true,
			`unable to read users file: open test`,
		},
		{
			"invalid legacy users file",
			func() string {
				t.Setenv("USERS_FILE_PATH", "test")
				return ""
			},
			&Env{},
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
				t.Setenv("CONFIG_FILE_PATH", file.Name())
				return file.Name()
			},
			&Env{},
			true,
			`unable to unmarshal users file`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			file := tc.given()
			t.Cleanup(func() {
				os.Clearenv()
				os.Remove(file)
			})

			actual := &Env{}
			err := actual.Populate()

			if tc.error {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.want)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestIsDeprecated(t *testing.T) {
	t.Parallel()

	cases := []struct {
		description string
		given       Env
		expected    bool
	}{
		{
			"not deprecated with expiration date set",
			Env{Expiration: time.Now()},
			false,
		},
		{
			"deprecated with default expiration date value set",
			Env{Expiration: time.Time{}},
			true,
		},
		{
			"deprecated with nothing set",
			Env{},
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
	t.Parallel()

	cases := []struct {
		description string
		given       Env
		expected    bool
	}{
		{
			"before expiration date",
			Env{Expiration: time.Now().AddDate(0, 0, 1)},
			false,
		},
		{
			"past expiration date",
			Env{Expiration: time.Now().AddDate(0, 0, -1)},
			true,
		},
		{
			"invalid expiration date",
			Env{},
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
	t.Parallel()

	cases := []struct {
		description string
		given       Env
		expected    string
	}{
		{
			"users and expiration date",
			Env{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Users: []string{"test"}},
			`{"users":["test"],"expiration":"2023-01-01"}`,
		},
		{
			"no users and no expiration date",
			Env{},
			`{"users":[],"expiration":"0001-01-01"}`,
		},
		{
			"empty users and empty expiration date",
			Env{Users: []string{}, Expiration: time.Time{}},
			`{"users":[],"expiration":"0001-01-01"}`,
		},
		{
			"empty users and no expiration date",
			Env{Users: []string{}},
			`{"users":[],"expiration":"0001-01-01"}`,
		},
		{
			"users and no expiration date",
			Env{Users: []string{"test"}},
			`{"users":["test"],"expiration":"0001-01-01"}`,
		},
		{
			"no users and expiration date",
			Env{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
			`{"users":[],"expiration":"2023-01-01"}`,
		},
		{
			"empty users and expiration date",
			Env{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Users: []string{}},
			`{"users":[],"expiration":"2023-01-01"}`,
		},
		{
			"users and empty expiration date",
			Env{Users: []string{"test"}, Expiration: time.Time{}},
			`{"users":["test"],"expiration":"0001-01-01"}`,
		},
		{
			"no users and empty expiration date",
			Env{Expiration: time.Time{}},
			`{"users":[],"expiration":"0001-01-01"}`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			results, err := tc.given.MarshalJSON()

			require.NoError(t, err)
			assert.Equal(t, tc.expected, string(results))
		})
	}
}

func TestUnmarshalJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		description string
		given       string
		expected    Env
		error       bool
		want        string
	}{
		{
			"valid JSON with users and expiration date",
			`{"users":["test"],"expiration":"2023-01-01"}`,
			Env{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Users: []string{"test"}},
			false,
			``,
		},
		{
			"valid JSON with users and expiration date set to null",
			`{"users":null,"expiration":null}`,
			Env{},
			true,
			`unable to parse expiration date`,
		},
		{
			"valid JSON with users and expiration set to empty values",
			`{"users":[],"expiration":""}`,
			Env{},
			true,
			`unable to parse expiration date`,
		},
		{
			"valid JSON with users set to null without expiration date",
			`{"users":null}`,
			Env{},
			true,
			`unable to find expiration date`,
		},
		{
			"valid JSON with users set to empty value without expiration date",
			`{"users":[]}`,
			Env{},
			true,
			`unable to find expiration date`,
		},
		{
			"valid JSON with user set to invalid value",
			`{"users":["test", -1], "expiration": "2023-01-01"}`,
			Env{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), Users: []string{"test"}},
			true,
			`unable to parse user`,
		},
		{
			"valid JSON without users with expiration date",
			`{"expiration":"2023-01-01"}`,
			Env{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
			true,
			`unable to find users list`,
		},
		{
			"valid JSON without users with expiration date set to null",
			`{"expiration":null}`,
			Env{},
			true,
			`unable to parse expiration date`,
		},
		{
			"valid JSON with users set to null with expiration date",
			`{"users":null,"expiration":"2023-01-01"}`,
			Env{Expiration: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
			true,
			`unable to parse users list`,
		},
		{
			"invalid empty string with nothing set",
			``,
			Env{},
			true,
			`unable to unmarshal user file`,
		},
		{
			"invalid empty JSON object with nothing set",
			`{}`,
			Env{},
			true,
			`unable to find expiration date`,
		},
		{
			"invalid JSON",
			`{"users:[], "expiration": "2023-01-01"}`,
			Env{},
			true,
			`unable to unmarshal user file`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			results := Env{}
			err := results.UnmarshalJSON([]byte(tc.given))

			if tc.error {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.want)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.expected, results)
		})
	}
}
