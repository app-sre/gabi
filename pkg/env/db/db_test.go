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
	assert.IsType(t, &DBEnv{}, actual)
}

func TestPopulate(t *testing.T) {
	cases := []struct {
		description string
		given       func()
		expected    *DBEnv
		error       bool
		want        string
	}{
		{
			"all environment variables set",
			func() {
				os.Clearenv()
				os.Setenv("DB_DRIVER", "pgx")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_PORT", "1234")
				os.Setenv("DB_USER", "test")
				os.Setenv("DB_PASS", "test123")
				os.Setenv("DB_NAME", "test")
				os.Setenv("DB_WRITE", "false")
			},
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
				os.Clearenv()
			},
			&DBEnv{},
			true,
			`unable to access environment variable: DB_DRIVER`,
		},
		{
			"required environment variable DB_DRIVER with empty value set",
			func() {
				os.Clearenv()
				os.Setenv("DB_DRIVER", "")
			},
			&DBEnv{},
			true,
			`unable to access environment variable: DB_DRIVER`,
		},
		{
			"missing required DB_HOST environment variable",
			func() {
				os.Clearenv()
				os.Setenv("DB_DRIVER", "pgx")
			},
			&DBEnv{Driver: "pgx"},
			true,
			`unable to access environment variable: DB_HOST`,
		},
		{
			"missing required DB_USER environment variable",
			func() {
				os.Clearenv()
				os.Setenv("DB_DRIVER", "pgx")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_PORT", "1234")
			},
			&DBEnv{Driver: "pgx", Host: "test", Port: 1234},
			true,
			`unable to access environment variable: DB_USER`,
		},
		{
			"missing required DB_PASS environment variable",
			func() {
				os.Clearenv()
				os.Setenv("DB_DRIVER", "pgx")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_PORT", "1234")
				os.Setenv("DB_USER", "test")
			},
			&DBEnv{Driver: "pgx", Host: "test", Port: 1234, Username: "test"},
			true,
			`unable to access environment variable: DB_PASS`,
		},
		{
			"missing required DB_NAME environment variable",
			func() {
				os.Clearenv()
				os.Setenv("DB_DRIVER", "pgx")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_PORT", "1234")
				os.Setenv("DB_USER", "test")
				os.Setenv("DB_PASS", "test123")
			},
			&DBEnv{Driver: "pgx", Host: "test", Port: 1234, Username: "test", Password: "test123"},
			true,
			`unable to access environment variable: DB_NAME`,
		},
		{
			"only required environment variables set",
			func() {
				os.Clearenv()
				os.Setenv("DB_DRIVER", "pgx")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_USER", "test")
				os.Setenv("DB_PASS", "test123")
				os.Setenv("DB_NAME", "test")
			},
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
				os.Clearenv()
				os.Setenv("DB_DRIVER", "postgres")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_USER", "test")
				os.Setenv("DB_PASS", "test123")
				os.Setenv("DB_NAME", "test")
			},
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
				os.Clearenv()
				os.Setenv("DB_DRIVER", "test")
			},
			&DBEnv{Driver: "test", Host: "", Port: 0, Username: "", Password: "", Name: "", AllowWrite: false},
			true,
			`unable to use driver type: test`,
		},
		{
			"environment variable with database write enabled",
			func() {
				os.Clearenv()
				os.Setenv("DB_DRIVER", "pgx")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_USER", "test")
				os.Setenv("DB_PASS", "test123")
				os.Setenv("DB_NAME", "test")
				os.Setenv("DB_WRITE", "true")
			},
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
				os.Clearenv()
				os.Setenv("DB_DRIVER", "pgx")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_PORT", "test")
			},
			&DBEnv{Driver: "pgx", Host: "test", Port: 5432, Username: "", Password: "", Name: "", AllowWrite: false},
			true,
			`unable to convert environment variable: DB_PORT`,
		},
		{
			"environment variable with invalid database write controls",
			func() {
				os.Clearenv()
				os.Setenv("DB_DRIVER", "pgx")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_USER", "test")
				os.Setenv("DB_PASS", "test123")
				os.Setenv("DB_NAME", "test")
				os.Setenv("DB_WRITE", "-1")
			},
			&DBEnv{Driver: "pgx", Host: "test", Port: 5432, Username: "test", Password: "test123", Name: "test", AllowWrite: false},
			true,
			`unable to convert environment variable: DB_WRITE`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			tc.given()

			actual := &DBEnv{}
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
				os.Clearenv()
				os.Setenv("DB_DRIVER", "pgx")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_PORT", "1234")
				os.Setenv("DB_USER", "test")
				os.Setenv("DB_PASS", "test123")
				os.Setenv("DB_NAME", "test")
			},
			`postgres://test:test123@test:1234/test`,
		},
		{
			"connection string for PostgreSQL with password using reserved characters",
			func() {
				os.Clearenv()
				os.Setenv("DB_DRIVER", "pgx")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_PORT", "1234")
				os.Setenv("DB_USER", "test")
				os.Setenv("DB_PASS", "t#e%s$t&!123")
				os.Setenv("DB_NAME", "test")
			},
			`postgres://test:t%23e%25s$t&%21123@test:1234/test`,
		},
		{
			"connection string for MySQL",
			func() {
				os.Clearenv()
				os.Setenv("DB_DRIVER", "mysql")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_PORT", "1234")
				os.Setenv("DB_USER", "test")
				os.Setenv("DB_PASS", "test123")
				os.Setenv("DB_NAME", "test")
			},
			`test:test123@tcp(test:1234)/test`,
		},
		{
			"connection string for MySQL with password using reserved characters",
			func() {
				os.Clearenv()
				os.Setenv("DB_DRIVER", "mysql")
				os.Setenv("DB_HOST", "test")
				os.Setenv("DB_PORT", "1234")
				os.Setenv("DB_USER", "test")
				os.Setenv("DB_PASS", "t#e%s$t&!123")
				os.Setenv("DB_NAME", "test")
			},
			`test:t#e%s$t&!123@tcp(test:1234)/test`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			tc.given()

			expected := &DBEnv{}
			err := expected.Populate()

			require.NoError(t, err)

			actual := expected.ConnectionDSN()

			assert.Equal(t, tc.want, actual)
		})
	}
}
