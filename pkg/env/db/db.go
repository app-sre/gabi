package db

import (
	"fmt"
	"os"
	"strconv"

	"github.com/app-sre/gabi/pkg/env"
)

type Dbenv struct {
	DB_DRIVER string
	DB_HOST   string
	DB_PORT   string
	DB_USER   string
	DB_PASS   string
	DB_NAME   string
	DB_WRITE  bool
	ConnStr   string
}

func (dbe *Dbenv) Populate() error {
	driver, found := os.LookupEnv("DB_DRIVER")
	if !(found) {
		return &env.EnvError{Env: "DB_DRIVER"}
	}
	host, found := os.LookupEnv("DB_HOST")
	if !(found) {
		return &env.EnvError{Env: "DB_HOST"}
	}
	port, found := os.LookupEnv("DB_PORT")
	if !(found) {
		return &env.EnvError{Env: "DB_PORT"}
	}
	user, found := os.LookupEnv("DB_USER")
	if !(found) {
		return &env.EnvError{Env: "DB_USER"}
	}
	pass, found := os.LookupEnv("DB_PASS")
	if !(found) {
		return &env.EnvError{Env: "DB_PASS"}
	}
	name, found := os.LookupEnv("DB_NAME")
	if !(found) {
		return &env.EnvError{Env: "DB_NAME"}
	}
	writeStr, found := os.LookupEnv("DB_WRITE")
	if !(found) {
		return &env.EnvError{Env: "DB_WRITE"}
	}
	write, err := strconv.ParseBool(writeStr)
	if err != nil {
		return &env.EnvConvError{Env: "DB_WRITE"}
	}

	dbe.DB_DRIVER = driver
	dbe.DB_HOST = host
	dbe.DB_PORT = port
	dbe.DB_USER = user
	dbe.DB_PASS = pass
	dbe.DB_NAME = name
	dbe.DB_WRITE = write

	dbe.ConnStr = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		dbe.DB_USER,
		dbe.DB_PASS,
		dbe.DB_HOST,
		dbe.DB_PORT,
		dbe.DB_NAME,
	)

	return nil
}
