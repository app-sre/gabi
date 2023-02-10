package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDriveType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		description string
		given       string
		want        string
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

			require.Equal(t, tc.want, actual.String())

			assert.Equal(t, tc.port, actual.Port())
			assert.Equal(t, tc.format, actual.Format())
			assert.Equal(t, tc.valid, actual.IsValid())
		})
	}
}
