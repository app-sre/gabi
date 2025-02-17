package db

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDBEnv(t *testing.T) {
	t.Parallel()

	actual := NewDBEnv()

	require.NotNil(t, actual)
	assert.IsType(t, &Env{}, actual)
}

func TestPopulate(t *testing.T) {
	cases := []struct {
		description string
		given       func()
		expected    *Env
		error       bool
		want        string
	}{
		{
			"all environment variables set",
			func() {
				t.Setenv("DB_DRIVER", "pgx")
				t.Setenv("DB_HOST", "test")
				t.Setenv("DB_PORT", "1234")
				t.Setenv("DB_USER", "test")
				t.Setenv("DB_PASS", "test123")
				t.Setenv("DB_NAME", "test")
				t.Setenv("DB_WRITE", "false")
			},
			&Env{
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
			},
			&Env{},
			true,
			`unable to access environment variable: DB_DRIVER`,
		},
		{
			"required environment variable DB_DRIVER with empty value set",
			func() {
				t.Setenv("DB_DRIVER", "")
			},
			&Env{},
			true,
			`unable to access environment variable: DB_DRIVER`,
		},
		{
			"missing required DB_HOST environment variable",
			func() {
				t.Setenv("DB_DRIVER", "pgx")
			},
			&Env{Driver: "pgx"},
			true,
			`unable to access environment variable: DB_HOST`,
		},
		{
			"missing required DB_USER environment variable",
			func() {
				t.Setenv("DB_DRIVER", "pgx")
				t.Setenv("DB_HOST", "test")
				t.Setenv("DB_PORT", "1234")
			},
			&Env{Driver: "pgx", Host: "test", Port: 1234},
			true,
			`unable to access environment variable: DB_USER`,
		},
		{
			"missing required DB_PASS environment variable",
			func() {
				t.Setenv("DB_DRIVER", "pgx")
				t.Setenv("DB_HOST", "test")
				t.Setenv("DB_PORT", "1234")
				t.Setenv("DB_USER", "test")
			},
			&Env{Driver: "pgx", Host: "test", Port: 1234, Username: "test"},
			true,
			`unable to access environment variable: DB_PASS`,
		},
		{
			"missing required DB_NAME environment variable",
			func() {
				t.Setenv("DB_DRIVER", "pgx")
				t.Setenv("DB_HOST", "test")
				t.Setenv("DB_PORT", "1234")
				t.Setenv("DB_USER", "test")
				t.Setenv("DB_PASS", "test123")
			},
			&Env{Driver: "pgx", Host: "test", Port: 1234, Username: "test", Password: "test123"},
			true,
			`unable to access environment variable: DB_NAME`,
		},
		{
			"only required environment variables set",
			func() {
				t.Setenv("DB_DRIVER", "pgx")
				t.Setenv("DB_HOST", "test")
				t.Setenv("DB_USER", "test")
				t.Setenv("DB_PASS", "test123")
				t.Setenv("DB_NAME", "test")
			},
			&Env{
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
				t.Setenv("DB_DRIVER", "postgres")
				t.Setenv("DB_HOST", "test")
				t.Setenv("DB_USER", "test")
				t.Setenv("DB_PASS", "test123")
				t.Setenv("DB_NAME", "test")
			},
			&Env{
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
				t.Setenv("DB_DRIVER", "test")
			},
			&Env{Driver: "test", Host: "", Port: 0, Username: "", Password: "", Name: "", AllowWrite: false},
			true,
			`unable to use driver type: test`,
		},
		{
			"environment variable with database write enabled",
			func() {
				t.Setenv("DB_DRIVER", "pgx")
				t.Setenv("DB_HOST", "test")
				t.Setenv("DB_USER", "test")
				t.Setenv("DB_PASS", "test123")
				t.Setenv("DB_NAME", "test")
				t.Setenv("DB_WRITE", "true")
			},
			&Env{
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
				t.Setenv("DB_DRIVER", "pgx")
				t.Setenv("DB_HOST", "test")
				t.Setenv("DB_PORT", "test")
			},
			&Env{Driver: "pgx", Host: "test", Port: 5432, Username: "", Password: "", Name: "", AllowWrite: false},
			true,
			`unable to convert environment variable: DB_PORT`,
		},
		{
			"environment variable with invalid database write controls",
			func() {
				t.Setenv("DB_DRIVER", "pgx")
				t.Setenv("DB_HOST", "test")
				t.Setenv("DB_USER", "test")
				t.Setenv("DB_PASS", "test123")
				t.Setenv("DB_NAME", "test")
				t.Setenv("DB_WRITE", "-1")
			},
			&Env{Driver: "pgx", Host: "test", Port: 5432, Username: "test", Password: "test123", Name: "test", AllowWrite: false},
			true,
			`unable to convert environment variable: DB_WRITE`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Cleanup(func() {
				os.Clearenv()
			})

			tc.given()

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

func TestConnectionDSN(t *testing.T) {
	cases := []struct {
		description string
		given       func()
		want        string
	}{
		{
			"connection string for PostgreSQL",
			func() {
				t.Setenv("DB_DRIVER", "pgx")
				t.Setenv("DB_HOST", "test")
				t.Setenv("DB_PORT", "1234")
				t.Setenv("DB_USER", "test")
				t.Setenv("DB_PASS", "test123")
				t.Setenv("DB_NAME", "test")
			},
			`postgres://test:test123@test:1234/test`,
		},
		{
			"connection string for PostgreSQL with password using reserved characters",
			func() {
				t.Setenv("DB_DRIVER", "pgx")
				t.Setenv("DB_HOST", "test")
				t.Setenv("DB_PORT", "1234")
				t.Setenv("DB_USER", "test")
				t.Setenv("DB_PASS", "t#e%s$t&!123")
				t.Setenv("DB_NAME", "test")
			},
			`postgres://test:t%23e%25s$t&%21123@test:1234/test`,
		},
		{
			"connection string for MySQL",
			func() {
				t.Setenv("DB_DRIVER", "mysql")
				t.Setenv("DB_HOST", "test")
				t.Setenv("DB_PORT", "1234")
				t.Setenv("DB_USER", "test")
				t.Setenv("DB_PASS", "test123")
				t.Setenv("DB_NAME", "test")
			},
			`test:test123@tcp(test:1234)/test`,
		},
		{
			"connection string for MySQL with password using reserved characters",
			func() {
				t.Setenv("DB_DRIVER", "mysql")
				t.Setenv("DB_HOST", "test")
				t.Setenv("DB_PORT", "1234")
				t.Setenv("DB_USER", "test")
				t.Setenv("DB_PASS", "t#e%s$t&!123")
				t.Setenv("DB_NAME", "test")
			},
			`test:t#e%s$t&!123@tcp(test:1234)/test`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Cleanup(func() {
				os.Clearenv()
			})

			tc.given()

			expected := &Env{}
			err := expected.Populate()

			require.NoError(t, err)

			actual := expected.ConnectionDSN("")

			assert.Equal(t, tc.want, actual)
		})
	}
}
