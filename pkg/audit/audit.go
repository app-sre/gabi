package audit

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
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
	Index      string
	Token      string
	Endpoint   string
	Host       string
	Pod        string
	Namespace  string
	Source     string
	Sourcetype string
}

func (d *SplunkAudit) Init(se *splunk.Splunkenv) {
	d.Index = se.SPLUNK_INDEX
	d.Token = se.SPLUNK_TOKEN
	d.Endpoint = se.URL
	d.Source = se.SOURCE
	d.Sourcetype = se.SOURCETYPE
	d.Namespace = se.NAMESPACE
	d.Pod = se.POD
	d.Host = se.HOST
}

func (d *SplunkAudit) Write(s *SplunkQueryData) (SplunkResponse, error) {
	splunkResp := &SplunkResponse{}

	s.Index = d.Index
	s.Host = d.Host
	s.Source = d.Source
	s.Sourcetype = d.Sourcetype
	s.Event.Namespace = d.Namespace
	s.Event.Pod = d.Pod

	postBody, _ := json.Marshal(s)
	responseBody := bytes.NewBuffer(postBody)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	req, err := http.NewRequest("POST", d.Endpoint, responseBody)
	if err != nil {
    	return *splunkResp, err
	}
	req.Header.Add("Authorization", "Splunk " + d.Token)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return *splunkResp, err
	}
   	defer resp.Body.Close()
	
   	body, err := ioutil.ReadAll(resp.Body)
   	if err != nil {
		return  *splunkResp, err
   	}
	
	err = json.Unmarshal(body, splunkResp)
	if err != nil {
		return  *splunkResp, err
	}

   	return *splunkResp, nil
}