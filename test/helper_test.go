//go:build integration
// +build integration

package test

import (
	"crypto/tls"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/app-sre/gabi/pkg/env/user"
)

func dummyHTTPClient() http.Client {
	return http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}

func createConfigurationFile(t *testing.T, expiration time.Time, users []string) string {
	usere := &user.Env{
		Expiration: expiration,
		Users:      users,
	}

	file, err := os.CreateTemp("", "user-")
	if err != nil {
		t.Fatal(err)
	}

	if err := json.NewEncoder(file).Encode(&usere); err != nil {
		t.Fatal(err)
	}

	return file.Name()
}

func setEnvironment(configFile, dbHost, dbPort, dbWrite, splunkToken, splunkEndpoint string) {
	os.Setenv("DB_DRIVER", "pgx")
	os.Setenv("DB_HOST", dbHost)
	os.Setenv("DB_PORT", dbPort)
	os.Setenv("DB_USER", "gabi")
	os.Setenv("DB_PASS", "passwd")
	os.Setenv("DB_NAME", "mydb")
	os.Setenv("DB_WRITE", dbWrite)

	os.Setenv("SPLUNK_INDEX", "main")
	os.Setenv("SPLUNK_TOKEN", splunkToken)
	os.Setenv("SPLUNK_ENDPOINT", splunkEndpoint)

	os.Setenv("HOST", "test")
	os.Setenv("NAMESPACE", "test")
	os.Setenv("POD_NAME", "test")

	if configFile != "" {
		os.Setenv("CONFIG_FILE_PATH", configFile)
	}
}

func waitForPortOpen(port int) {
	address := net.JoinHostPort("localhost", strconv.Itoa(port))
	for {
		_, err := net.DialTimeout("tcp", address, 500*time.Millisecond)
		if err == nil {
			break
		}
	}
}
