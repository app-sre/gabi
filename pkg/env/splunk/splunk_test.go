package splunk

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSplunkEnv(t *testing.T) {
	actual := NewSplunkEnv()

	assert.NotNil(t, actual)
	assert.IsType(t, &SplunkEnv{}, actual)
}

func TestPopulate(t *testing.T) {
	cases := []struct {
		description string
		given       func()
		clean       func()
		expected    *SplunkEnv
		error       bool
		message     string
	}{
		{
			"all environment variables set",
			func() {
				os.Setenv("SPLUNK_INDEX", "test")
				os.Setenv("SPLUNK_ENDPOINT", "test")
				os.Setenv("SPLUNK_TOKEN", "test123")
				os.Setenv("HOST", "test")
				os.Setenv("NAMESPACE", "test")
				os.Setenv("POD_NAME", "test")
			},
			os.Clearenv,
			&SplunkEnv{Index: "test", Endpoint: "test", Token: "test123", Host: "test", Namespace: "test", Pod: "test"},
			false,
			``,
		},
		{
			"missing required SPLUNK_INDEX environment variable",
			func() {
				// No-op.
			},
			os.Clearenv,
			&SplunkEnv{},
			true,
			`unable to access environment variable: SPLUNK_INDEX`,
		},
		{
			"empty required SPLUNK_INDEX environment variable",
			func() {
				os.Setenv("SPLUNK_INDEX", "")
			},
			os.Clearenv,
			&SplunkEnv{},
			true,
			`unable to access environment variable: SPLUNK_INDEX`,
		},
		{
			"missing required SPLUNK_ENDPOINT environment variable",
			func() {
				os.Setenv("SPLUNK_INDEX", "test")
			},
			os.Clearenv,
			&SplunkEnv{Index: "test"},
			true,
			`unable to access environment variable: SPLUNK_ENDPOINT`,
		},
		{
			"missing required SPLUNK_TOKEN environment variable",
			func() {
				os.Setenv("SPLUNK_INDEX", "test")
				os.Setenv("SPLUNK_ENDPOINT", "test")
			},
			os.Clearenv,
			&SplunkEnv{Index: "test", Endpoint: "test", Token: "", Host: "", Namespace: "", Pod: ""},
			true,
			`unable to access environment variable: SPLUNK_TOKEN`,
		},
		{
			"missing required HOST environment variable",
			func() {
				os.Setenv("SPLUNK_INDEX", "test")
				os.Setenv("SPLUNK_ENDPOINT", "test")
				os.Setenv("SPLUNK_TOKEN", "test123")
			},
			os.Clearenv,
			&SplunkEnv{Index: "test", Endpoint: "test", Token: "test123", Host: "", Namespace: "", Pod: ""},
			true,
			`unable to access environment variable: HOST`,
		},
		{
			"missing required NAMESPACE environment variable",
			func() {
				os.Setenv("SPLUNK_INDEX", "test")
				os.Setenv("SPLUNK_ENDPOINT", "test")
				os.Setenv("SPLUNK_TOKEN", "test123")
				os.Setenv("HOST", "test")
			},
			os.Clearenv,
			&SplunkEnv{Index: "test", Endpoint: "test", Token: "test123", Host: "test", Namespace: "", Pod: ""},
			true,
			`unable to access environment variable: NAMESPACE`,
		},
		{
			"missing required POD_NAME environment variable",
			func() {
				os.Setenv("SPLUNK_INDEX", "test")
				os.Setenv("SPLUNK_ENDPOINT", "test")
				os.Setenv("SPLUNK_TOKEN", "test123")
				os.Setenv("HOST", "test")
				os.Setenv("NAMESPACE", "test")
			},
			os.Clearenv,
			&SplunkEnv{Index: "test", Endpoint: "test", Token: "test123", Host: "test", Namespace: "test", Pod: ""},
			true,
			`unable to access environment variable: POD_NAME`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			tc.clean()

			tc.given()
			actual := &SplunkEnv{}
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
