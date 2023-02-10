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
				t.Setenv("SPLUNK_INDEX", "test")
				t.Setenv("SPLUNK_ENDPOINT", "test")
				t.Setenv("SPLUNK_TOKEN", "test123")
				t.Setenv("HOST", "test")
				t.Setenv("NAMESPACE", "test")
				t.Setenv("POD_NAME", "test")
			},
			&SplunkEnv{Index: "test", Endpoint: "test", Token: "test123", Host: "test", Namespace: "test", Pod: "test"},
			false,
			``,
		},
		{
			"missing required SPLUNK_INDEX environment variable",
			func() {
			},
			&SplunkEnv{},
			true,
			`unable to access environment variable: SPLUNK_INDEX`,
		},
		{
			"empty required SPLUNK_INDEX environment variable",
			func() {
				t.Setenv("SPLUNK_INDEX", "")
			},
			&SplunkEnv{},
			true,
			`unable to access environment variable: SPLUNK_INDEX`,
		},
		{
			"missing required SPLUNK_ENDPOINT environment variable",
			func() {
				t.Setenv("SPLUNK_INDEX", "test")
			},
			&SplunkEnv{Index: "test"},
			true,
			`unable to access environment variable: SPLUNK_ENDPOINT`,
		},
		{
			"missing required SPLUNK_TOKEN environment variable",
			func() {
				t.Setenv("SPLUNK_INDEX", "test")
				t.Setenv("SPLUNK_ENDPOINT", "test")
			},
			&SplunkEnv{Index: "test", Endpoint: "test", Token: "", Host: "", Namespace: "", Pod: ""},
			true,
			`unable to access environment variable: SPLUNK_TOKEN`,
		},
		{
			"missing required HOST environment variable",
			func() {
				t.Setenv("SPLUNK_INDEX", "test")
				t.Setenv("SPLUNK_ENDPOINT", "test")
				t.Setenv("SPLUNK_TOKEN", "test123")
			},
			&SplunkEnv{Index: "test", Endpoint: "test", Token: "test123", Host: "", Namespace: "", Pod: ""},
			true,
			`unable to access environment variable: HOST`,
		},
		{
			"missing required NAMESPACE environment variable",
			func() {
				t.Setenv("SPLUNK_INDEX", "test")
				t.Setenv("SPLUNK_ENDPOINT", "test")
				t.Setenv("SPLUNK_TOKEN", "test123")
				t.Setenv("HOST", "test")
			},
			&SplunkEnv{Index: "test", Endpoint: "test", Token: "test123", Host: "test", Namespace: "", Pod: ""},
			true,
			`unable to access environment variable: NAMESPACE`,
		},
		{
			"missing required POD_NAME environment variable",
			func() {
				t.Setenv("SPLUNK_INDEX", "test")
				t.Setenv("SPLUNK_ENDPOINT", "test")
				t.Setenv("SPLUNK_TOKEN", "test123")
				t.Setenv("HOST", "test")
				t.Setenv("NAMESPACE", "test")
			},
			&SplunkEnv{Index: "test", Endpoint: "test", Token: "test123", Host: "test", Namespace: "test", Pod: ""},
			true,
			`unable to access environment variable: POD_NAME`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			t.Cleanup(func() {
				os.Clearenv()
			})

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
