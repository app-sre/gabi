package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDriveType(t *testing.T) {
	cases := []struct {
		description string
		given       string
		name        string
		port        int
		format      string
		valid       bool
	}{
		{
			"valid MySQL driver",
			"mysql",
			"mysql",
			3306,
			`%s:%s@tcp(%s:%d)/%s`,
			true,
		},
		{
			"valid PostgreSQL driver using default name",
			"pgx",
			"pgx",
			5432,
			`postgres://%s:%s@%s:%d/%s`,
			true,
		},
		{
			"valid PostgreSQL driver using postgres as name",
			"postgres",
			"pgx",
			5432,
			`postgres://%s:%s@%s:%d/%s`,
			true,
		},
		{
			"valid PostgreSQL driver using postgresql as name",
			"postgresql",
			"pgx",
			5432,
			`postgres://%s:%s@%s:%d/%s`,
			true,
		},
		{
			"invalid SQL driver with nothing set",
			"",
			"",
			0,
			``,
			false,
		},
		{
			"invalid SQL driver using test as name",
			"test",
			"",
			0,
			``,
			false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			actual := DriverType(tc.given)

			assert.Equal(t, tc.name, actual.Name())
			assert.Equal(t, tc.port, actual.Port())
			assert.Equal(t, tc.format, actual.Format())
			assert.Equal(t, tc.valid, actual.IsValid())
			assert.Equal(t, actual.Name(), actual.String())
		})
	}
}
