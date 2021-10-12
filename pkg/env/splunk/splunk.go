package splunk

import (
	"os"

	"github.com/app-sre/gabi/pkg/env"
)

type Splunkenv struct {
	SPLUNK_INDEX    string
	SPLUNK_TOKEN    string
	SPLUNK_ENDPOINT string
	URL             string
	NAMESPACE       string
	POD             string
	SOURCE          string
	SOURCETYPE      string
	HOST            string
}

func (se *Splunkenv) Populate() error {
	index, found := os.LookupEnv("SPLUNK_INDEX")
	if !(found) {
		return &env.EnvError{Env: "SPLUNK_INDEX"}
	}
	token, found := os.LookupEnv("SPLUNK_TOKEN")
	if !(found) {
		return &env.EnvError{Env: "SPLUNK_TOKEN"}
	}
	endpoint, found := os.LookupEnv("SPLUNK_ENDPOINT")
	if !(found) {
		return &env.EnvError{Env: "SPLUNK_ENDPOINT"}
	}
	namespace, found := os.LookupEnv("NAMESPACE")
	if !(found) {
		return &env.EnvError{Env: "NAMESPACE"}
	}
	pod, found := os.LookupEnv("POD_NAME")
	if !(found) {
		return &env.EnvError{Env: "POD_NAME"}
	}
	host, found := os.LookupEnv("HOST")
	if !(found) {
		return &env.EnvError{Env: "HOST"}
	}

	se.SPLUNK_INDEX = index
	se.SPLUNK_TOKEN = token
	se.SPLUNK_ENDPOINT = endpoint
	se.URL = endpoint + "/services/collector/event"
	se.NAMESPACE = namespace
	se.POD = pod
	se.SOURCE = "gabi"
	se.SOURCETYPE = "json"
	se.HOST = host

	return nil
}
