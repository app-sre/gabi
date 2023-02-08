package audit

type QueryData struct {
	Query     string
	User      string
	Namespace string
	Pod       string
	Timestamp int64
}

type Audit interface {
	Write(*QueryData) error
}
