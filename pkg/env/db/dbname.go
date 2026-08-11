package db

import (
	"fmt"
	"regexp"
)

// dbNamePattern is the allowlist for database names used in DSN construction.
// Alphanumeric and underscore only, matching common MySQL/PostgreSQL unquoted
// identifier rules and capping at the PostgreSQL NAMEDATALEN-1 limit (63).
// Rejecting punctuation closes connection-string parameter injection (CWE-88)
// via values such as "db?host=evil" or "db&allowAllFiles=true".
var dbNamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{1,63}$`)

// ValidateDBName reports whether name is safe to interpolate into a driver DSN.
func ValidateDBName(name string) error {
	if !dbNamePattern.MatchString(name) {
		return fmt.Errorf("invalid database name: must match %s", dbNamePattern.String())
	}
	return nil
}
