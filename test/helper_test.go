//go:build integration
// +build integration

package test

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/app-sre/gabi/pkg/env/user"
	"github.com/orlangure/gnomock"
	"github.com/orlangure/gnomock/preset/postgres"
	"github.com/orlangure/gnomock/preset/splunk"
	"github.com/stretchr/testify/require"
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

func createSplunkIngestToken(t *testing.T, client http.Client, host, port, password string) string {
	splunkURL := fmt.Sprintf("https://%s:%s/servicesNS/admin/splunk_httpinput/data/inputs/http?output_mode=json", host, port)

	// Use url.Values for proper form encoding
	data := url.Values{}
	data.Set("name", "mytokexna")

	req, err := http.NewRequest(http.MethodPost, splunkURL, strings.NewReader(data.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth("admin", password)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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

func deleteSplunkIngestToken(t *testing.T, client http.Client, host, port, password, tokenName string) {
	splunkURL := fmt.Sprintf("https://%s:%s/servicesNS/admin/splunk_httpinput/data/inputs/http/%s", host, port, tokenName)

	req, err := http.NewRequest(http.MethodDelete, splunkURL, nil)
	if err != nil {
		t.Logf("Failed to create delete request: %v", err)
		return
	}
	req.SetBasicAuth("admin", password)

	resp, err := client.Do(req)
	if err != nil {
		t.Logf("Failed to delete token: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Logf("Token deletion returned status: %d", resp.StatusCode)
	}
}

func startPostgres(t *testing.T) *gnomock.Container {
	p := postgres.Preset(
		postgres.WithUser("gnomock", "gnomick"),
		postgres.WithDatabase("mydb"),
	)

	healthcheck := func(ctx context.Context, c *gnomock.Container) error {
		connStr := fmt.Sprintf("host=%s port=%d user=gnomock password=gnomick dbname=mydb sslmode=disable",
			c.Host, c.DefaultPort())
		db, err := sql.Open("pgx", connStr)
		if err != nil {
			return err
		}
		defer db.Close()
		return db.PingContext(ctx)
	}

	options := []gnomock.Option{
		gnomock.WithUseLocalImagesFirst(),
		gnomock.WithEnv("POSTGRESQL_USER=gnomock"),
		gnomock.WithEnv("POSTGRESQL_PASSWORD=gnomick"),
		gnomock.WithEnv("POSTGRESQL_DATABASE=mydb"),
		gnomock.WithHealthCheck(healthcheck),
	}

	psql, err := gnomock.StartCustom("registry.redhat.io/rhel9/postgresql-16:9.6", p.Ports(),
		options...,
	)
	require.NoError(t, err)

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
	options = append(options,
		gnomock.WithUseLocalImagesFirst(),
		gnomock.WithEnv("SPLUNK_GENERAL_TERMS=--accept-sgt-current-at-splunk-com"),
	)

	splunk, err := gnomock.StartCustom("quay.io/app-sre/splunk:latest", s.Ports(),
		options...,
	)
	require.NoError(t, err)

	t.Cleanup(func() { _ = gnomock.Stop(splunk) })

	return splunk
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
