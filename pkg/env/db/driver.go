package db

const (
	driverMySQL      = "mysql"
	driverPostgreSQL = "pgx"

	driverMySQLPort      = 3306
	driverPostgreSQLPort = 5432

	driverMySQLFormat      = `%s:%s@tcp(%s:%d)/%s`
	driverPostgreSQLFormat = `postgres://%s:%s@%s:%d/%s`
)

type DriverType string

func (t DriverType) String() string {
	return string(t.driver())
}

func (t DriverType) Port() int {
	switch t.driver() {
	case driverMySQL:
		return driverMySQLPort
	case driverPostgreSQL:
		return driverPostgreSQLPort
	default:
		return 0
	}
}

func (t DriverType) Format() string {
	switch t.driver() {
	case driverMySQL:
		return driverMySQLFormat
	case driverPostgreSQL:
		return driverPostgreSQLFormat
	default:
		return ""
	}
}

func (t DriverType) IsValid() bool {
	types := map[string]interface{}{
		"mysql":      struct{}{},
		"postgres":   struct{}{},
		"postgresql": struct{}{},
		"pgx":        struct{}{},
	}
	_, ok := types[string(t)]

	return ok
}

func (t DriverType) driver() DriverType {
	switch t {
	case "mysql":
		t = driverMySQL
	case "postgresql", "postgres", "pgx":
		t = driverPostgreSQL
	default:
		t = ""
	}

	return t
}
