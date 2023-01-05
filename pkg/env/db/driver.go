package db

const (
	driverFormatMySQL      = `%s:%s@tcp(%s:%d)/%s`
	driverFormatPostgreSQL = `postgres://%s:%s@%s:%d/%s`
)

type DriverType string

func (t DriverType) String() string {
	return t.Name()
}

func (t DriverType) Name() (name string) {
	switch t {
	case "mysql":
		name = "mysql"
	case "postgres", "postgresql", "pgx":
		name = "pgx"
	}
	return
}

func (t DriverType) Port() (port int) {
	switch t.String() {
	case "mysql":
		port = 3306
	case "pgx":
		port = 5432
	}
	return
}

func (t DriverType) Format() (format string) {
	switch t.String() {
	case "mysql":
		format = driverFormatMySQL
	case "pgx":
		format = driverFormatPostgreSQL
	}
	return
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
