package db

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDBEnv(t *testing.T) {
	actual := NewDBEnv()

	assert.NotNil(t, actual)
	assert.IsType(t, &DBEnv{}, actual)
}

func TestPopulate(t *testing.T) {
	cases := []struct {
		description string
		given       func()
		clean       func()
		expected    *DBEnv
		error       bool
		message     string
	}{
		{
			"all environment variables set",
			func() {
				os.Setenv("DB_DRIVER", "pgx")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_PORT", "1234")
				os.Setenv("DB_USER", "test")
				os.Setenv("DB_PASS", "test123")
				os.Setenv("DB_NAME", "test")
				os.Setenv("DB_WRITE", "false")
			},
			os.Clearenv,
			&DBEnv{
				Driver:     "pgx",
				Host:       "test",
				Port:       1234,
				Username:   "test",
				Password:   "test123",
				Name:       "test",
				AllowWrite: false,
			},
			false,
			``,
		},
		{
			"missing required environment variables",
			func() {
				// No-op.
			},
			os.Clearenv,
			&DBEnv{},
			true,
			`unable to access environment variable: DB_DRIVER`,
		},
		{
			"required environment variable DB_DRIVER with empty value set",
			func() {
				os.Setenv("DB_DRIVER", "")
			},
			os.Clearenv,
			&DBEnv{},
			true,
			`unable to access environment variable: DB_DRIVER`,
		},
		{
			"missing required DB_HOST environment variable",
			func() {
				os.Setenv("DB_DRIVER", "pgx")
			},
			os.Clearenv,
			&DBEnv{Driver: "pgx"},
			true,
			`unable to access environment variable: DB_HOST`,
		},
		{
			"missing required DB_USER environment variable",
			func() {
				os.Setenv("DB_DRIVER", "pgx")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_PORT", "1234")
			},
			os.Clearenv,
			&DBEnv{Driver: "pgx", Host: "test", Port: 1234},
			true,
			`unable to access environment variable: DB_USER`,
		},
		{
			"missing required DB_PASS environment variable",
			func() {
				os.Setenv("DB_DRIVER", "pgx")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_PORT", "1234")
				os.Setenv("DB_USER", "test")
			},
			os.Clearenv,
			&DBEnv{Driver: "pgx", Host: "test", Port: 1234, Username: "test"},
			true,
			`unable to access environment variable: DB_PASS`,
		},
		{
			"missing required DB_NAME environment variable",
			func() {
				os.Setenv("DB_DRIVER", "pgx")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_PORT", "1234")
				os.Setenv("DB_USER", "test")
				os.Setenv("DB_PASS", "test123")
			},
			os.Clearenv,
			&DBEnv{Driver: "pgx", Host: "test", Port: 1234, Username: "test", Password: "test123"},
			true,
			`unable to access environment variable: DB_NAME`,
		},
		{
			"only required environment variables set",
			func() {
				os.Setenv("DB_DRIVER", "pgx")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_USER", "test")
				os.Setenv("DB_PASS", "test123")
				os.Setenv("DB_NAME", "test")
			},
			os.Clearenv,
			&DBEnv{
				Driver:     "pgx",
				Host:       "test",
				Port:       5432,
				Username:   "test",
				Password:   "test123",
				Name:       "test",
				AllowWrite: false,
			},
			false,
			``,
		},
		{
			"environment variable with alternative driver name set",
			func() {
				os.Setenv("DB_DRIVER", "postgres")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_USER", "test")
				os.Setenv("DB_PASS", "test123")
				os.Setenv("DB_NAME", "test")
			},
			os.Clearenv,
			&DBEnv{
				Driver:     "postgres",
				Host:       "test",
				Port:       5432,
				Username:   "test",
				Password:   "test123",
				Name:       "test",
				AllowWrite: false,
			},
			false,
			``,
		},
		{
			"environment variable with invalid driver name set",
			func() {
				os.Setenv("DB_DRIVER", "test")
			},
			os.Clearenv,
			&DBEnv{Driver: "test", Host: "", Port: 0, Username: "", Password: "", Name: "", AllowWrite: false},
			true,
			`unable to use driver type: test`,
		},
		{
			"environment variable with database write enabled",
			func() {
				os.Setenv("DB_DRIVER", "pgx")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_USER", "test")
				os.Setenv("DB_PASS", "test123")
				os.Setenv("DB_NAME", "test")
				os.Setenv("DB_WRITE", "true")
			},
			os.Clearenv,
			&DBEnv{
				Driver:     "pgx",
				Host:       "test",
				Port:       5432,
				Username:   "test",
				Password:   "test123",
				Name:       "test",
				AllowWrite: true,
			},
			false,
			``,
		},
		{
			"environment variable with invalid database port set",
			func() {
				os.Setenv("DB_DRIVER", "pgx")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_PORT", "test")
			},
			os.Clearenv,
			&DBEnv{Driver: "pgx", Host: "test", Port: 5432, Username: "", Password: "", Name: "", AllowWrite: false},
			true,
			`unable to convert environment variable: DB_PORT`,
		},
		{
			"environment variable with invalid database write controls",
			func() {
				os.Setenv("DB_DRIVER", "pgx")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_USER", "test")
				os.Setenv("DB_PASS", "test123")
				os.Setenv("DB_NAME", "test")
				os.Setenv("DB_WRITE", "-1")
			},
			os.Clearenv,
			&DBEnv{Driver: "pgx", Host: "test", Port: 5432, Username: "test", Password: "test123", Name: "test", AllowWrite: false},
			true,
			`unable to convert environment variable: DB_WRITE`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			tc.clean()

			tc.given()
			actual := &DBEnv{}
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

func TestConnectionDSN(t *testing.T) {
	cases := []struct {
		description string
		given       func()
		clean       func()
		expected    string
	}{
		{
			"connection string for PostgreSQL",
			func() {
				os.Setenv("DB_DRIVER", "pgx")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_PORT", "1234")
				os.Setenv("DB_USER", "test")
				os.Setenv("DB_PASS", "test123")
				os.Setenv("DB_NAME", "test")
			},
			os.Clearenv,
			`postgres://test:test123@test:1234/test`,
		},
		{
			"connection string for PostgreSQL with password using reserved characters",
			func() {
				os.Setenv("DB_DRIVER", "pgx")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_PORT", "1234")
				os.Setenv("DB_USER", "test")
				os.Setenv("DB_PASS", "t#e%s$t&!123")
				os.Setenv("DB_NAME", "test")
			},
			os.Clearenv,
			`postgres://test:t%23e%25s$t&%21123@test:1234/test`,
		},
		{
			"connection string for MySQL",
			func() {
				os.Setenv("DB_DRIVER", "mysql")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_PORT", "1234")
				os.Setenv("DB_USER", "test")
				os.Setenv("DB_PASS", "test123")
				os.Setenv("DB_NAME", "test")
			},
			os.Clearenv,
			`test:test123@tcp(test:1234)/test`,
		},
		{
			"connection string for MySQL with password using reserved characters",
			func() {
				os.Setenv("DB_DRIVER", "mysql")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_PORT", "1234")
				os.Setenv("DB_USER", "test")
				os.Setenv("DB_PASS", "t#e%s$t&!123")
				os.Setenv("DB_NAME", "test")
			},
			os.Clearenv,
			`test:t#e%s$t&!123@tcp(test:1234)/test`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			tc.clean()

			tc.given()
			aux := &DBEnv{}
			err := aux.Populate()

			actual := aux.ConnectionDSN()

			assert.Nil(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
