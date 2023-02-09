package audit

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/app-sre/gabi/pkg/env/splunk"
	"github.com/app-sre/gabi/pkg/version"
)

const (
	splunkSource     = "gabi"
	splunkSourceType = "json"

	connectTimeout = 5 * time.Second
	requestTimeout = 30 * time.Second
)

type SplunkAudit struct {
	SplunkEnv *splunk.SplunkEnv

	client *http.Client
}

var _ Audit = (*SplunkAudit)(nil)

type SplunkEventData struct {
	Query     string `json:"query"`
	User      string `json:"user"`
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
}

type SplunkQueryData struct {
	Event      *SplunkEventData `json:"event"`
	Index      string           `json:"index"`
	Host       string           `json:"host"`
	Source     string           `json:"source"`
	SourceType string           `json:"sourcetype"`
	Time       int64            `json:"time"`
}

type Option func(*SplunkAudit)

func WithHTTPClient(client *http.Client) Option {
	return func(s *SplunkAudit) {
		s.SetHTTPClient(client)
	}
}

func NewSplunkAudit(splunk *splunk.SplunkEnv, options ...Option) *SplunkAudit {
	s := &SplunkAudit{SplunkEnv: splunk}

	s.client = &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: connectTimeout,
			}).DialContext,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	for _, option := range options {
		option(s)
	}

	return s
}

func (d *SplunkAudit) SetHTTPClient(client *http.Client) {
	d.client = client
}

func (d *SplunkAudit) Write(q *QueryData) error {
	query := &SplunkQueryData{
		Index:      d.SplunkEnv.Index,
		Host:       d.SplunkEnv.Host,
		Source:     splunkSource,
		SourceType: splunkSourceType,
		Time:       q.Timestamp,
	}

	query.Event = &SplunkEventData{
		Query:     q.Query,
		User:      q.User,
		Namespace: d.SplunkEnv.Namespace,
		Pod:       d.SplunkEnv.Pod,
	}

	content, err := json.Marshal(query)
	if err != nil {
		return fmt.Errorf("unable to marshal Splunk audit: %w", err)
	}

	url := fmt.Sprintf("%s/services/collector/event", d.SplunkEnv.Endpoint)

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(content))
	if err != nil {
		return fmt.Errorf("unable to create request to Splunk: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Splunk %s", d.SplunkEnv.Token))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("User-Agent", fmt.Sprintf("GABI/%s", version.Version()))

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("unable to send request to Splunk: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("unable to read Splunk response body: %w", err)
	}

	splunk := struct {
		Code int    `json:"code"`
		Text string `json:"text"`
	}{}

	err = json.Unmarshal(body, &splunk)
	if err != nil {
		return fmt.Errorf("unable to unmarshal Splunk response: %w", err)
	}
	if splunk.Code > 0 {
		return fmt.Errorf("unable to write to Splunk: %s (%d)", splunk.Text, splunk.Code)
	}

	return nil
}
