package splunk

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSplunkEnv(t *testing.T) {
	t.Parallel()

	actual := NewSplunkEnv()

	require.NotNil(t, actual)
	assert.IsType(t, &SplunkEnv{}, actual)
}

func TestPopulate(t *testing.T) {
	cases := []struct {
		description string
		given       func()
		expected    *SplunkEnv
		error       bool
		want        string
	}{
		{
			"all environment variables set",
			func() {
				os.Clearenv()
				os.Setenv("SPLUNK_INDEX", "test")
				os.Setenv("SPLUNK_ENDPOINT", "test")
				os.Setenv("SPLUNK_TOKEN", "test123")
				os.Setenv("HOST", "test")
				os.Setenv("NAMESPACE", "test")
				os.Setenv("POD_NAME", "test")
			},
			&SplunkEnv{Index: "test", Endpoint: "test", Token: "test123", Host: "test", Namespace: "test", Pod: "test"},
			false,
			``,
		},
		{
			"missing required SPLUNK_INDEX environment variable",
			func() {
				os.Clearenv()
			},
			&SplunkEnv{},
			true,
			`unable to access environment variable: SPLUNK_INDEX`,
		},
		{
			"empty required SPLUNK_INDEX environment variable",
			func() {
				os.Clearenv()
				os.Setenv("SPLUNK_INDEX", "")
			},
			&SplunkEnv{},
			true,
			`unable to access environment variable: SPLUNK_INDEX`,
		},
		{
			"missing required SPLUNK_ENDPOINT environment variable",
			func() {
				os.Clearenv()
				os.Setenv("SPLUNK_INDEX", "test")
			},
			&SplunkEnv{Index: "test"},
			true,
			`unable to access environment variable: SPLUNK_ENDPOINT`,
		},
		{
			"missing required SPLUNK_TOKEN environment variable",
			func() {
				os.Clearenv()
				os.Setenv("SPLUNK_INDEX", "test")
				os.Setenv("SPLUNK_ENDPOINT", "test")
			},
			&SplunkEnv{Index: "test", Endpoint: "test", Token: "", Host: "", Namespace: "", Pod: ""},
			true,
			`unable to access environment variable: SPLUNK_TOKEN`,
		},
		{
			"missing required HOST environment variable",
			func() {
				os.Clearenv()
				os.Setenv("SPLUNK_INDEX", "test")
				os.Setenv("SPLUNK_ENDPOINT", "test")
				os.Setenv("SPLUNK_TOKEN", "test123")
			},
			&SplunkEnv{Index: "test", Endpoint: "test", Token: "test123", Host: "", Namespace: "", Pod: ""},
			true,
			`unable to access environment variable: HOST`,
		},
		{
			"missing required NAMESPACE environment variable",
			func() {
				os.Clearenv()
				os.Setenv("SPLUNK_INDEX", "test")
				os.Setenv("SPLUNK_ENDPOINT", "test")
				os.Setenv("SPLUNK_TOKEN", "test123")
				os.Setenv("HOST", "test")
			},
			&SplunkEnv{Index: "test", Endpoint: "test", Token: "test123", Host: "test", Namespace: "", Pod: ""},
			true,
			`unable to access environment variable: NAMESPACE`,
		},
		{
			"missing required POD_NAME environment variable",
			func() {
				os.Clearenv()
				os.Setenv("SPLUNK_INDEX", "test")
				os.Setenv("SPLUNK_ENDPOINT", "test")
				os.Setenv("SPLUNK_TOKEN", "test123")
				os.Setenv("HOST", "test")
				os.Setenv("NAMESPACE", "test")
			},
			&SplunkEnv{Index: "test", Endpoint: "test", Token: "test123", Host: "test", Namespace: "test", Pod: ""},
			true,
			`unable to access environment variable: POD_NAME`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			tc.given()

			actual := &SplunkEnv{}
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
