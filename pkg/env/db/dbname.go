package db

import (
	"fmt"
	"regexp"
)

// dbNamePattern is the allowlist for database names used in DSN construction.
// Letters, digits, underscore, and hyphen (not as the first character), capped
// at the PostgreSQL NAMEDATALEN-1 limit (63). Hyphens are valid in quoted
// MySQL/PostgreSQL database names and are not DSN metacharacters.
// Rejecting other punctuation closes connection-string parameter injection
// (CWE-88) via values such as "db?host=evil" or "db&allowAllFiles=true".
var dbNamePattern = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_-]{0,62}$`)

// ValidateDBName reports whether name is safe to interpolate into a driver DSN.
func ValidateDBName(name string) error {
	if !dbNamePattern.MatchString(name) {
		return fmt.Errorf("invalid database name: must match %s", dbNamePattern.String())
	}
	return nil
}
