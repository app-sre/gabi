package db

import (
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/app-sre/gabi/pkg/env"
)

type DBEnv struct {
	Driver     DriverType
	Host       string
	Port       int
	Username   string
	Password   string
	Name       string
	AllowWrite bool
}

func NewDBEnv() *DBEnv {
	return &DBEnv{}
}

func (d *DBEnv) Populate() error {
	driver := os.Getenv("DB_DRIVER")
	if driver == "" {
		return &env.EnvError{Name: "DB_DRIVER"}
	}
	d.Driver = DriverType(driver)

	if !d.Driver.IsValid() {
		return fmt.Errorf("unable to use driver type: %s", driver)
	}

	host := os.Getenv("DB_HOST")
	if host == "" {
		return &env.EnvError{Name: "DB_HOST"}
	}
	d.Host = host

	d.Port = d.Driver.Port()
	portString := os.Getenv("DB_PORT")
	if portString != "" {
		port, err := strconv.ParseInt(portString, 10, 0)
		if err != nil {
			return &env.EnvTypeError{Name: "DB_PORT"}
		}
		d.Port = int(port)
	}

	username := os.Getenv("DB_USER")
	if username == "" {
		return &env.EnvError{Name: "DB_USER"}
	}
	d.Username = username

	password := os.Getenv("DB_PASS")
	if password == "" {
		return &env.EnvError{Name: "DB_PASS"}
	}
	d.Password = password

	name := os.Getenv("DB_NAME")
	if name == "" {
		return &env.EnvError{Name: "DB_NAME"}
	}
	d.Name = name

	d.AllowWrite = false
	writeString := os.Getenv("DB_WRITE")
	if writeString != "" {
		write, err := strconv.ParseBool(writeString)
		if err != nil {
			return &env.EnvTypeError{Name: "DB_WRITE"}
		}
		d.AllowWrite = write
	}

	// Only do this for PostgreSQL driver as the MySQL driver will handle encoding.
	if d.Driver == driverPostgreSQL {
		d.Password = url.PathEscape(d.Password)
	}

	return nil
}

func (d *DBEnv) ConnectionDSN() string {
	return fmt.Sprintf(d.Driver.Format(), d.Username, d.Password, d.Host, d.Port, d.Name)
}
