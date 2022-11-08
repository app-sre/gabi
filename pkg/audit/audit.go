package audit

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"

	"go.uber.org/zap"

	"github.com/app-sre/gabi/pkg/env/splunk"
)

type QueryData struct {
	Query     string
	User      string
	Timestamp int64
}

type Audit interface {
	Write(*QueryData) error
}

type LoggingAudit struct {
	Logger *zap.SugaredLogger
}

func (d *LoggingAudit) Write(q *QueryData) error {
	d.Logger.Infow("gabi API audit record",
		"Query", q.Query,
		"User", q.User,
		"Timestamp", q.Timestamp,
	)
	return nil
}

type SplunkEventData struct {
	Query     string `json:"query"`
	User      string `json:"user"`
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
}

type SplunkQueryData struct {
	Event      *SplunkEventData `json:"event"`
	Time       int64 `json:"time"`
	Host       string `json:"host"`
	Source     string `json:"source"`
	Sourcetype string `json:"sourcetype"`
	Index      string `json:"index"`
}

type SplunkResponse struct {
	Text string
	Code int
}

type SplunkAudit struct {
	Env *splunk.Splunkenv
}

func (d *SplunkAudit) Write(s *SplunkQueryData) (SplunkResponse, error) {
	splunkResp := &SplunkResponse{}

	s.Index = d.Env.SPLUNK_INDEX
	s.Host = d.Env.HOST
	s.Source = d.Env.SOURCE
	s.Sourcetype = d.Env.SOURCETYPE
	s.Event.Namespace = d.Env.NAMESPACE
	s.Event.Pod = d.Env.POD

	postBody, _ := json.Marshal(s)
	responseBody := bytes.NewBuffer(postBody)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	req, err := http.NewRequest("POST", d.Env.URL, responseBody)
	if err != nil {
    	return *splunkResp, err
	}
	req.Header.Add("Authorization", "Splunk " + d.Env.SPLUNK_TOKEN)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return *splunkResp, err
	}
   	defer resp.Body.Close()
	
   	body, err := io.ReadAll(resp.Body)
   	if err != nil {
		return  *splunkResp, err
   	}
	
	err = json.Unmarshal(body, splunkResp)
	if err != nil {
		return  *splunkResp, err
	}

   	return *splunkResp, nil
}
