//go:build integration
// +build integration

package test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/app-sre/gabi/internal/test"
	"github.com/app-sre/gabi/pkg/cmd"
	"github.com/app-sre/gabi/pkg/env/user"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthCheckOK(t *testing.T) {
	psql := startPostgres(t)

	configFile := createConfigurationFile(t, time.Now().AddDate(0, 0, 1), []string{"test"})
	defer os.Remove(configFile)

	setEnvironment(configFile, psql.Host, strconv.Itoa(psql.DefaultPort()), "false", "test123", "localhost")
	defer os.Clearenv()

	logger := test.DummyLogger(io.Discard)
	defer logger.Sync()

	go func() {
		err := cmd.Run(logger.Sugar())
		require.NoError(t, err)
	}()
	waitForPortOpen(8080)

	resp, err := http.Get("http://localhost:8080/healthcheck")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), `{"status":"OK"}`)
}

func TestHealthCheckFailure(t *testing.T) {
	configFile := createConfigurationFile(t, time.Now().AddDate(0, 0, 1), []string{"test"})
	defer os.Remove(configFile)

	setEnvironment(configFile, "localhost", "1123", "false", "test123", "localhost")
	defer os.Clearenv()

	logger := test.DummyLogger(io.Discard)
	defer logger.Sync()

	go func() {
		err := cmd.Run(logger.Sugar())
		require.NoError(t, err)
	}()
	waitForPortOpen(8080)

	resp, err := http.Get("http://localhost:8080/healthcheck")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	assert.Contains(t, string(body), `{"status":"Service Unavailable","errors":{"database":"Unable to connect to the database"}}`)
}

func TestQueryWithMissingHeader(t *testing.T) {
	client := dummyHTTPClient()

	psql := startPostgres(t)

	configFile := createConfigurationFile(t, time.Now().AddDate(0, 0, 1), []string{"test"})
	defer os.Remove(configFile)

	setEnvironment(configFile, psql.Host, strconv.Itoa(psql.DefaultPort()), "false", "test123", "localhost")
	defer os.Clearenv()

	logger := test.DummyLogger(io.Discard)
	defer logger.Sync()

	go func() {
		err := cmd.Run(logger.Sugar())
		require.NoError(t, err)
	}()
	waitForPortOpen(8080)

	req, err := http.NewRequest("POST", "http://localhost:8080/query", bytes.NewBuffer([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, string(body), `Request without required header: X-Forwarded-User`)
}

func TestQueryWithMalformedBase64EncodedQuery(t *testing.T) {
	client := dummyHTTPClient()

	psql := startPostgres(t)

	configFile := createConfigurationFile(t, time.Now().AddDate(0, 0, 1), []string{"test"})
	defer os.Remove(configFile)

	setEnvironment(configFile, psql.Host, strconv.Itoa(psql.DefaultPort()), "false", "test123", "localhost")
	defer os.Clearenv()

	logger := test.DummyLogger(io.Discard)
	defer logger.Sync()

	go func() {
		err := cmd.Run(logger.Sugar())
		require.NoError(t, err)
	}()
	waitForPortOpen(8080)

	req, err := http.NewRequest("POST", "http://localhost:8080/query", bytes.NewBuffer([]byte(`{"query": "dGhpcyBpcyBhIHRlc3Q=="}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Forwarded-User", "test")

	q := req.URL.Query()
	q.Add("base64_query", "true")

	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, string(body), `Unable to decode Base64-encoded query`)
}

func TestQueryWithMissingBody(t *testing.T) {
	client := dummyHTTPClient()

	psql := startPostgres(t)

	configFile := createConfigurationFile(t, time.Now().AddDate(0, 0, 1), []string{"test"})
	defer os.Remove(configFile)

	setEnvironment(configFile, psql.Host, strconv.Itoa(psql.DefaultPort()), "false", "test123", "localhost")
	defer os.Clearenv()

	logger := test.DummyLogger(io.Discard)
	defer logger.Sync()

	go func() {
		err := cmd.Run(logger.Sugar())
		require.NoError(t, err)
	}()
	waitForPortOpen(8080)

	req, err := http.NewRequest("POST", "http://localhost:8080/query", &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Forwarded-User", "test")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, string(body), `Request body cannot be empty`)
}

func TestQueryWithExpiredInstance(t *testing.T) {
	client := dummyHTTPClient()

	psql := startPostgres(t)

	configFile := createConfigurationFile(t, time.Now().AddDate(0, 0, -1), []string{"test"})
	defer os.Remove(configFile)

	setEnvironment(configFile, psql.Host, strconv.Itoa(psql.DefaultPort()), "false", "test123", "localhost")
	defer os.Clearenv()

	var output bytes.Buffer

	logger := test.DummyLogger(&output)
	defer logger.Sync()

	go func() {
		err := cmd.Run(logger.Sugar())
		require.NoError(t, err)
	}()
	waitForPortOpen(8080)

	req, err := http.NewRequest("POST", "http://localhost:8080/query", bytes.NewBuffer([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Forwarded-User", "test")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, output.String(), `expired: true`)
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	assert.Contains(t, string(body), `The service instance has expired`)
}

func TestQueryWithUnauthorizedAccess(t *testing.T) {
	client := dummyHTTPClient()

	configFile := createConfigurationFile(t, time.Now().AddDate(0, 0, 1), []string{})
	defer os.Remove(configFile)

	psql := startPostgres(t)
	setEnvironment(configFile, psql.Host, strconv.Itoa(psql.DefaultPort()), "false", "test123", "localhost")
	defer os.Clearenv()

	logger := test.DummyLogger(io.Discard)
	defer logger.Sync()

	go func() {
		err := cmd.Run(logger.Sugar())
		require.NoError(t, err)
	}()
	waitForPortOpen(8080)

	req, err := http.NewRequest("POST", "http://localhost:8080/query", bytes.NewBuffer([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Forwarded-User", "test")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Contains(t, string(body), `Request cannot be authorized`)
}

func TestQueryWithForbiddenUserAccess(t *testing.T) {
	client := dummyHTTPClient()

	configFile := createConfigurationFile(t, time.Now().AddDate(0, 0, 1), []string{"test"})
	defer os.Remove(configFile)

	psql := startPostgres(t)
	setEnvironment(configFile, psql.Host, strconv.Itoa(psql.DefaultPort()), "false", "test123", "localhost")
	defer os.Clearenv()

	logger := test.DummyLogger(io.Discard)
	defer logger.Sync()

	go func() {
		err := cmd.Run(logger.Sugar())
		require.NoError(t, err)
	}()
	waitForPortOpen(8080)

	req, err := http.NewRequest("POST", "http://localhost:8080/query", bytes.NewBuffer([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Forwarded-User", "user-without-access-permissions")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	assert.Contains(t, string(body), `User does not have required permissions`)
}

func TestQueryWithAccessUsingEnvironment(t *testing.T) {
	client := dummyHTTPClient()

	psql := startPostgres(t)

	os.Setenv("EXPIRATION_DATE", time.Now().AddDate(0, 0, 1).Format(user.ExpiryDateLayout))
	os.Setenv("AUTHORIZED_USERS", "test")

	setEnvironment("", psql.Host, strconv.Itoa(psql.DefaultPort()), "false", "test123", "localhost")
	defer os.Clearenv()

	var output bytes.Buffer

	logger := test.DummyLogger(&output)
	defer logger.Sync()

	go func() {
		err := cmd.Run(logger.Sugar())
		require.NoError(t, err)
	}()
	waitForPortOpen(8080)

	req, err := http.NewRequest("POST", "http://localhost:8080/query", &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Forwarded-User", "test")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	assert.Contains(t, output.String(), `Authorized users: [test]`)
}

func TestQueryWithSplunkWrite(t *testing.T) {
	client := dummyHTTPClient()
	splunkPassword := "foobarPassword123!"

	psql := startPostgres(t)
	splunk := startSplunk(t, splunkPassword)

	token := createSplunkIngestToken(t, client, "localhost", strconv.Itoa(splunk.Port("api")), splunkPassword)

	configFile := createConfigurationFile(t, time.Now().AddDate(0, 0, 1), []string{"test"})
	defer os.Remove(configFile)

	setEnvironment(
		configFile,
		psql.Host,
		strconv.Itoa(psql.DefaultPort()),
		"false",
		token,
		fmt.Sprintf("https://%s:%d", "localhost", splunk.Port("collector")),
	)
	defer os.Clearenv()

	logger := test.DummyLogger(io.Discard)
	defer logger.Sync()

	go func() {
		err := cmd.Run(logger.Sugar())
		require.NoError(t, err)
	}()
	waitForPortOpen(8080)

	req, err := http.NewRequest("POST", "http://localhost:8080/query", bytes.NewBuffer([]byte(`{"query":"select 1;"}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Forwarded-User", "test")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), `{"result":[["?column?"],["1"]],"error":""}`)
}

func TestQueryWithSplunkWriteFailure(t *testing.T) {
	client := dummyHTTPClient()
	splunkPassword := "foobarPassword123!"

	splunk := startSplunk(t, splunkPassword)

	configFile := createConfigurationFile(t, time.Now().AddDate(0, 0, 1), []string{"test"})
	defer os.Remove(configFile)

	setEnvironment(
		configFile,
		"test",
		"1234",
		"false",
		"test",
		fmt.Sprintf("https://%s:%d", "localhost", splunk.Port("collector")),
	)
	defer os.Clearenv()

	var output bytes.Buffer

	logger := test.DummyLogger(&output)
	defer logger.Sync()

	go func() {
		err := cmd.Run(logger.Sugar())
		require.NoError(t, err)
	}()
	waitForPortOpen(8080)

	req, err := http.NewRequest("POST", "http://localhost:8080/query", bytes.NewBuffer([]byte(`{"query":"select 1;"}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Forwarded-User", "test")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, output.String(), `Unable to send audit to Splunk`)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Contains(t, string(body), `An internal error has occurred`)
}

func TestQueryWithDatabaseWriteAccess(t *testing.T) {
	client := dummyHTTPClient()
	splunkPassword := "foobarPassword123!"

	psql := startPostgres(t)
	splunk := startSplunk(t, splunkPassword)

	token := createSplunkIngestToken(t, client, "localhost", strconv.Itoa(splunk.Port("api")), splunkPassword)

	configFile := createConfigurationFile(t, time.Now().AddDate(0, 0, 1), []string{"test"})
	defer os.Remove(configFile)

	setEnvironment(
		configFile,
		psql.Host,
		strconv.Itoa(psql.DefaultPort()),
		"true",
		token,
		fmt.Sprintf("https://%s:%d", "localhost", splunk.Port("collector")),
	)
	defer os.Clearenv()

	var output bytes.Buffer

	logger := test.DummyLogger(&output)
	defer logger.Sync()

	go func() {
		err := cmd.Run(logger.Sugar())
		require.NoError(t, err)
	}()
	waitForPortOpen(8080)

	req, err := http.NewRequest("POST", "http://localhost:8080/query", bytes.NewBuffer([]byte(`{"query":"create table test(test text);"}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Forwarded-User", "test")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, output.String(), `write access: true`)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), `[null]`)
}

func TestQueryWithDatabaseWriteAccessFailure(t *testing.T) {
	client := dummyHTTPClient()
	splunkPassword := "foobarPassword123!"

	psql := startPostgres(t)
	splunk := startSplunk(t, splunkPassword)

	token := createSplunkIngestToken(t, client, "localhost", strconv.Itoa(splunk.Port("api")), splunkPassword)

	configFile := createConfigurationFile(t, time.Now().AddDate(0, 0, 1), []string{"test"})
	defer os.Remove(configFile)

	setEnvironment(
		configFile,
		psql.Host,
		strconv.Itoa(psql.DefaultPort()),
		"false",
		token,
		fmt.Sprintf("https://%s:%d", "localhost", splunk.Port("collector")),
	)
	defer os.Clearenv()

	var output bytes.Buffer

	logger := test.DummyLogger(&output)
	defer logger.Sync()

	go func() {
		err := cmd.Run(logger.Sugar())
		assert.Nil(t, err)
	}()
	waitForPortOpen(8080)

	req, err := http.NewRequest("POST", "http://localhost:8080/query", bytes.NewBuffer([]byte(`{"query":"drop table test"}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Forwarded-User", "test")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, output.String(), `Unable to query database`)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, string(body), `cannot execute DROP TABLE in a read-only transaction`)
}

func TestQueryWithBase64EncodedQuery(t *testing.T) {
	client := dummyHTTPClient()
	splunkPassword := "foobarPassword123!"

	psql := startPostgres(t)
	splunk := startSplunk(t, splunkPassword)

	token := createSplunkIngestToken(t, client, "localhost", strconv.Itoa(splunk.Port("api")), splunkPassword)

	configFile := createConfigurationFile(t, time.Now().AddDate(0, 0, 1), []string{"test"})
	defer os.Remove(configFile)

	setEnvironment(
		configFile,
		psql.Host,
		strconv.Itoa(psql.DefaultPort()),
		"false",
		token,
		fmt.Sprintf("https://%s:%d", "localhost", splunk.Port("collector")),
	)
	defer os.Clearenv()

	var output bytes.Buffer

	logger := test.DummyLogger(&output)
	defer logger.Sync()

	go func() {
		err := cmd.Run(logger.Sugar())
		assert.Nil(t, err)
	}()
	waitForPortOpen(8080)

	req, err := http.NewRequest("POST", "http://localhost:8080/query", bytes.NewBuffer([]byte(`{"query":"c2VsZWN0IDE7"}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Forwarded-User", "test")

	q := req.URL.Query()
	q.Add("base64_query", "true")

	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, output.String(), `"Query": "select 1;"`)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), `{"result":[["?column?"],["1"]],"error":""}`)
}

func TestQueryWithBase64EncodedResults(t *testing.T) {
	client := dummyHTTPClient()
	splunkPassword := "foobarPassword123!"

	psql := startPostgres(t)
	splunk := startSplunk(t, splunkPassword)

	token := createSplunkIngestToken(t, client, "localhost", strconv.Itoa(splunk.Port("api")), splunkPassword)

	configFile := createConfigurationFile(t, time.Now().AddDate(0, 0, 1), []string{"test"})
	defer os.Remove(configFile)

	setEnvironment(
		configFile,
		psql.Host,
		strconv.Itoa(psql.DefaultPort()),
		"false",
		token,
		fmt.Sprintf("https://%s:%d", "localhost", splunk.Port("collector")),
	)
	defer os.Clearenv()

	var output bytes.Buffer

	logger := test.DummyLogger(&output)
	defer logger.Sync()

	go func() {
		err := cmd.Run(logger.Sugar())
		assert.Nil(t, err)
	}()
	waitForPortOpen(8080)

	req, err := http.NewRequest("POST", "http://localhost:8080/query", bytes.NewBuffer([]byte(`{"query":"select current_schema();"}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Forwarded-User", "test")

	q := req.URL.Query()
	q.Add("base64_results", "true")

	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, output.String(), `"Query": "select current_schema();"`)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), `{"result":[["current_schema"],["cHVibGlj"]],"error":""}`)
}
