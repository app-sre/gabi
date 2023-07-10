package audit

import "context"

type QueryData struct {
	Query     string
	User      string
	Namespace string
	Pod       string
	Timestamp int64
}

type Audit interface {
	Write(context.Context, *QueryData) error
}
