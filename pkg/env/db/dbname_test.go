package db

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDBName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		dbName  string
		wantErr bool
	}{
		{name: "simple", dbName: "mydb", wantErr: false},
		{name: "underscores and digits", dbName: "app_sre_db1", wantErr: false},
		{name: "hyphen", dbName: "my-db", wantErr: false},
		{name: "hyphenated production name", dbName: "rhsm-subscriptions", wantErr: false},
		{name: "max length", dbName: strings.Repeat("a", 63), wantErr: false},
		{name: "empty", dbName: "", wantErr: true},
		{name: "too long", dbName: strings.Repeat("a", 64), wantErr: true},
		{name: "leading hyphen", dbName: "-mydb", wantErr: true},
		{name: "query injection", dbName: "db?host=evil", wantErr: true},
		{name: "ampersand injection", dbName: "db&allowAllFiles=true", wantErr: true},
		{name: "slash", dbName: "db/name", wantErr: true},
		{name: "space", dbName: "my db", wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateDBName(tc.dbName)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestConnectionDSNRejectsInjectionViaPathEscape(t *testing.T) {
	t.Parallel()

	env := &Env{
		Driver:   "pgx",
		Host:     "db.example",
		Port:     5432,
		Username: "user",
		Password: "secret",
		Name:     "main",
	}

	// Even without ValidateDBName, PathEscape encodes '?' so it cannot start a
	// URL query component. ('=' remains literal inside the path segment.)
	dsn := env.ConnectionDSN("evil?host=attacker.example")
	assert.Equal(t, "postgres://user:secret@db.example:5432/evil%3Fhost=attacker.example", dsn)
	assert.NotContains(t, dsn, "?host=")
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	assert.Empty(t, u.RawQuery)
	assert.Equal(t, "/evil%3Fhost=attacker.example", u.EscapedPath())
}

func TestConnectionDSNPreservesHyphenatedDBName(t *testing.T) {
	t.Parallel()

	env := &Env{
		Driver:   "pgx",
		Host:     "db.example",
		Port:     5432,
		Username: "user",
		Password: "secret",
		Name:     "main",
	}

	dsn := env.ConnectionDSN("rhsm-subscriptions")
	assert.Equal(t, "postgres://user:secret@db.example:5432/rhsm-subscriptions", dsn)
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	assert.Equal(t, "rhsm-subscriptions", strings.TrimPrefix(u.Path, "/"))
}
