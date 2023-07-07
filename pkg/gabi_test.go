package gabi

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProduction(t *testing.T) {
	cases := []struct {
		description string
		given       func()
		want        bool
	}{
		{
			"environment variable value set to production",
			func() {
				t.Setenv("ENVIRONMENT", "production")
			},
			true,
		},
		{
			"environment variable value set to other environment",
			func() {
				t.Setenv("ENVIRONMENT", "test")
			},
			false,
		},
		{
			"environment variable without value",
			func() {
				t.Setenv("ENVIRONMENT", "")
			},
			false,
		},
		{
			"environment variable not set",
			func() {
				// No-op.
			},
			false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Cleanup(func() {
				os.Clearenv()
			})

			tc.given()

			actual := Production()

			assert.Equal(t, tc.want, actual)
		})
	}
}

func TestRequestTimeout(t *testing.T) {
	cases := []struct {
		description string
		given       func()
		want        time.Duration
	}{
		{
			"overridden duration with environment variable value with unit",
			func() {
				t.Setenv("REQUEST_TIMEOUT", "5s")
			},
			time.Duration(5 * time.Second),
		},
		{
			"overridden duration with environment variable value without unit",
			func() {
				t.Setenv("REQUEST_TIMEOUT", "5")
			},
			time.Duration(5 * time.Second),
		},
		{
			"default duration with environment variable without value",
			func() {
				t.Setenv("REQUEST_TIMEOUT", "")
			},
			time.Duration(2 * time.Minute),
		},
		{
			"default duration with environment variable not set",
			func() {
				// No-op.
			},
			time.Duration(2 * time.Minute),
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Cleanup(func() {
				os.Clearenv()
			})

			tc.given()

			actual := RequestTimeout()

			assert.Equal(t, tc.want, actual)
		})
	}
}

func TestParseDuration(t *testing.T) {
	t.Parallel()

	cases := []struct {
		description string
		given       string
		expected    time.Duration
		error       bool
		want        string
	}{
		{
			"valid duration value with unit",
			"5s",
			time.Duration(5 * time.Second),
			false,
			``,
		},
		{
			"valid duration value without unit",
			"5",
			time.Duration(5 * time.Second),
			false,
			``,
		},
		{
			"valid negative duration value without unit",
			"-5",
			time.Duration(5 * time.Second),
			false,
			``,
		},
		{
			"valid negative duration value with unit",
			"-5s",
			time.Duration(5 * time.Second),
			false,
			``,
		},
		{
			"valid floating point durtation value with unit",
			"1.234s",
			time.Duration(1.234 * float64(time.Second)),
			false,
			``,
		},
		{
			"invalid floating point duration value without unit",
			"1.234",
			time.Duration(0),
			true,
			`unable to parse duration: time: missing unit in duration "1.234"`,
		},
		{
			"invalid duration value",
			"test",
			time.Duration(0),
			true,
			`unable to parse duration: time: invalid duration "test"`,
		},
		{
			"invalid empty duration value",
			"",
			time.Duration(0),
			true,
			`unable to parse duration: time: invalid duration ""`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			actual, err := parseDuration(tc.given)

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
