package db

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"sync"

	"github.com/app-sre/gabi/pkg/env"
)

type Env struct {
	Driver     DriverType
	Host       string
	Port       int
	Username   string
	Password   string
	Name       string
	AllowWrite bool
	sync.Mutex
}

func NewDBEnv() *Env {
	return &Env{}
}

func (d *Env) Populate(dbName string) error {
	driver := os.Getenv("DB_DRIVER")
	if driver == "" {
		return &env.Error{Name: "DB_DRIVER"}
	}
	d.Driver = DriverType(driver)

	if !d.Driver.IsValid() {
		return fmt.Errorf("unable to use driver type: %s", driver)
	}

	host := os.Getenv("DB_HOST")
	if host == "" {
		return &env.Error{Name: "DB_HOST"}
	}
	d.Host = host

	d.Port = d.Driver.Port()
	portString := os.Getenv("DB_PORT")
	if portString != "" {
		port, err := strconv.ParseInt(portString, 10, 0)
		if err != nil {
			return &env.TypeError{Name: "DB_PORT"}
		}
		d.Port = int(port)
	}

	username := os.Getenv("DB_USER")
	if username == "" {
		return &env.Error{Name: "DB_USER"}
	}
	d.Username = username

	password := os.Getenv("DB_PASS")
	if password == "" {
		return &env.Error{Name: "DB_PASS"}
	}
	d.Password = password

	if dbName != "" {
		d.Name = dbName
	} else {
		name := os.Getenv("DB_NAME")
		if name == "" {
			return &env.Error{Name: "DB_NAME"}
		}
		d.Name = name
	}

	d.AllowWrite = false
	writeString := os.Getenv("DB_WRITE")
	if writeString != "" {
		write, err := strconv.ParseBool(writeString)
		if err != nil {
			return &env.TypeError{Name: "DB_WRITE"}
		}
		d.AllowWrite = write
	}

	// Only do this for PostgreSQL driver as the MySQL driver will handle encoding.
	if d.Driver == driverPostgreSQL {
		d.Password = url.PathEscape(d.Password)
	}

	return nil
}

func (d *Env) ConnectionDSN() string {
	return fmt.Sprintf(d.Driver.Format(), d.Username, d.Password, d.Host, d.Port, d.Name)
}

func (d *Env) OverrideDBName(dbName string) {
	d.Lock()
	defer d.Unlock()
	d.Name = dbName
}

func (d *Env) GetCurrentDBName() string {
	d.Lock()
	defer d.Unlock()
	return d.Name
}
