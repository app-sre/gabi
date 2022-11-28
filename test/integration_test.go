package test

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/app-sre/gabi/pkg/cmd"
	"github.com/orlangure/gnomock"
	"github.com/orlangure/gnomock/preset/postgres"
	"github.com/orlangure/gnomock/preset/splunk"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type splunkTokenResponse struct {
	Entry []struct {
		Content struct {
			Token string `json:"token"`
		} `json:"content"`
	} `json:"entry"`
}

func insecureHttpClient() http.Client {
	client := http.Client{}
	transport := &http.Transport{}
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client.Transport = transport

	return client
}

func createIngestToken(client http.Client, host, port, password string) splunkTokenResponse {
	api := fmt.Sprintf("https://%s:%s/servicesNS/admin/splunk_httpinput/data/inputs/http?output_mode=json", host, port)
	body := []byte(`name=mytokexna`)

	req, err := http.NewRequest("POST", api, bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}
	req.SetBasicAuth("admin", password)

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var token splunkTokenResponse
	resp_body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(resp_body, &token)
	if err != nil {
		panic(err)
	}
	return token
}

func startPostgres(t *testing.T) *gnomock.Container {
	p := postgres.Preset(
		postgres.WithUser("gnomock", "gnomick"),
		postgres.WithDatabase("mydb"),
	)
	options := p.Options()
	options = append(options, gnomock.WithRegistryAuth(os.Getenv("QUAY_TOKEN")))
	options = append(options, gnomock.WithUseLocalImagesFirst())
	psql, err := gnomock.StartCustom("quay.io/app-sre/postgres:12.5", p.Ports(),
		options...,
	)
	assert.NoError(t, err)
	t.Cleanup(func() { _ = gnomock.Stop(psql) })
	return psql
}

func TestHealthCheckOkay(t *testing.T) {
	userfile := mustCreateUserTestFile()
	defer os.Remove(userfile)

	psql := startPostgres(t)
	setEnv(userfile, psql.Host, fmt.Sprint(psql.DefaultPort()), "", "localhost")

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	loggerS := logger.Sugar()

	go cmd.Run(loggerS)
	waitForGabi()

	resp, err := http.Get("http://localhost:8080/healthcheck")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	assert.Contains(t, string(body), "{\"status\":\"OK\"}")
}

func TestHealthCheckFail(t *testing.T) {
	userfile := mustCreateUserTestFile()
	defer os.Remove(userfile)

	setEnv(userfile, "localhost", "1123", "", "localhost")

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	loggerS := logger.Sugar()

	go cmd.Run(loggerS)
	waitForGabi()

	resp, err := http.Get("http://localhost:8080/healthcheck")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	assert.Contains(t, string(body), "{\"status\":\"Service Unavailable\",\"errors\":{\"database\":\"failed to connect database.")
}

func TestWithSplunkWrite(t *testing.T) {
	client := insecureHttpClient()
	splunk_password := "foobarPassword123!"

	psql := startPostgres(t)

	s := splunk.Preset(
		splunk.WithVersion("latest"),
		splunk.WithLicense(true),
		splunk.WithPassword(splunk_password),
	)

	options := s.Options()
	options = append(options, gnomock.WithRegistryAuth(os.Getenv("QUAY_TOKEN")))
	options = append(options, gnomock.WithUseLocalImagesFirst())
	splunk, err := gnomock.StartCustom("quay.io/app-sre/splunk:latest", s.Ports(),
		options...,
	)
	assert.NoError(t, err)
	t.Cleanup(func() { _ = gnomock.Stop(splunk) })

	token := createIngestToken(client, "localhost", fmt.Sprint(splunk.Port("api")), "foobarPassword123!")

	userfile := mustCreateUserTestFile()
	defer os.Remove(userfile)

	setEnv(userfile, psql.Host, fmt.Sprint(psql.DefaultPort()), token.Entry[0].Content.Token, fmt.Sprintf("https://%s:%s", "localhost", fmt.Sprint(splunk.Port("collector"))))

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	loggerS := logger.Sugar()

	go cmd.Run(loggerS)
	waitForGabi()

	req, err := http.NewRequest("POST", "http://localhost:8080/query", bytes.NewBuffer([]byte(`{"query":"select 1"}`)))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-User", "test")

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	assert.Contains(t, string(body), "{\"result\":[[\"?column?\"],[\"1\"]],\"error\":\"\"}")
}

func setEnv(userfile, dbHost, dbPort, splunkToken, splunkEndpoint string) {
	os.Setenv("DB_DRIVER", "pgx")
	os.Setenv("DB_HOST", dbHost)
	os.Setenv("DB_PORT", dbPort)
	os.Setenv("DB_USER", "gnomock")
	os.Setenv("DB_PASS", "gnomick")
	os.Setenv("DB_NAME", "mydb")
	os.Setenv("DB_WRITE", "false")

	os.Setenv("SPLUNK_INDEX", "main")
	os.Setenv("SPLUNK_TOKEN", splunkToken)
	os.Setenv("SPLUNK_ENDPOINT", splunkEndpoint)
	os.Setenv("NAMESPACE", "a")
	os.Setenv("POD_NAME", "a")
	os.Setenv("HOST", "a")
	os.Setenv("USERS_FILE_PATH", userfile)
}

func mustCreateUserTestFile() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	file, err := ioutil.TempFile(wd, "user")
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(file.Name(), []byte(`test`), os.ModeAppend)
	if err != nil {
		panic(err)
	}

	path, err := filepath.Abs(file.Name())
	if err != nil {
		panic(err)
	}

	return path
}

func waitForGabi() {
	for {
		server, _ := net.ResolveTCPAddr("tcp", "localhost:8080")
		client, _ := net.ResolveTCPAddr("tcp", ":")
		_, err := net.DialTCP("tcp", client, server)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}
