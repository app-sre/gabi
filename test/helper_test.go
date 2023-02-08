//go:build integration
// +build integration

package test

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/app-sre/gabi/pkg/env/user"
	"github.com/orlangure/gnomock"
	"github.com/orlangure/gnomock/preset/postgres"
	"github.com/orlangure/gnomock/preset/splunk"
	"github.com/stretchr/testify/assert"
)

func dummyHTTPClient() http.Client {
	return http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}

func createConfigurationFile(t *testing.T, expiration time.Time, users []string) string {
	usere := &user.UserEnv{
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

func createSplunkIngestToken(t *testing.T, client http.Client, host, port, password string) string {
	splunkURL := fmt.Sprintf("https://%s:%s/servicesNS/admin/splunk_httpinput/data/inputs/http?output_mode=json", host, port)

	req, err := http.NewRequest("POST", splunkURL, bytes.NewBufferString(`name=mytokexna`))
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth("admin", password)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	response := struct {
		Entry []struct {
			Content struct {
				Token string `json:"token"`
			} `json:"content"`
		} `json:"entry"`
	}{}

	err = json.Unmarshal(respBody, &response)
	if err != nil {
		t.Fatal(err)
	}

	return response.Entry[0].Content.Token
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

func startSplunk(t *testing.T, password string) *gnomock.Container {
	s := splunk.Preset(
		splunk.WithVersion("latest"),
		splunk.WithLicense(true),
		splunk.WithPassword(password),
	)

	options := s.Options()
	options = append(options, gnomock.WithRegistryAuth(os.Getenv("QUAY_TOKEN")))
	options = append(options, gnomock.WithUseLocalImagesFirst())
	splunk, err := gnomock.StartCustom("quay.io/app-sre/splunk:latest", s.Ports(),
		options...,
	)
	assert.NoError(t, err)

	t.Cleanup(func() { _ = gnomock.Stop(splunk) })

	return splunk
}

func setEnvironment(configFile, dbHost, dbPort, dbWrite, splunkToken, splunkEndpoint string) {
	os.Setenv("DB_DRIVER", "pgx")
	os.Setenv("DB_HOST", dbHost)
	os.Setenv("DB_PORT", dbPort)
	os.Setenv("DB_USER", "gnomock")
	os.Setenv("DB_PASS", "gnomick")
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
