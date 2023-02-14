package splunk

import (
	"os"

	"github.com/app-sre/gabi/pkg/env"
)

type Env struct {
	Index     string
	Endpoint  string
	Token     string
	Host      string
	Namespace string
	Pod       string
}

func NewSplunkEnv() *Env {
	return &Env{}
}

func (s *Env) Populate() error {
	index := os.Getenv("SPLUNK_INDEX")
	if index == "" {
		return &env.Error{Name: "SPLUNK_INDEX"}
	}
	s.Index = index

	endpoint := os.Getenv("SPLUNK_ENDPOINT")
	if endpoint == "" {
		return &env.Error{Name: "SPLUNK_ENDPOINT"}
	}
	s.Endpoint = endpoint

	token := os.Getenv("SPLUNK_TOKEN")
	if token == "" {
		return &env.Error{Name: "SPLUNK_TOKEN"}
	}
	s.Token = token

	host := os.Getenv("HOST")
	if host == "" {
		return &env.Error{Name: "HOST"}
	}
	s.Host = host

	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		return &env.Error{Name: "NAMESPACE"}
	}
	s.Namespace = namespace

	pod := os.Getenv("POD_NAME")
	if pod == "" {
		return &env.Error{Name: "POD_NAME"}
	}
	s.Pod = pod

	return nil
}
