package models

type QueryRequest struct {
	Query string `json:"query"`
}

type QueryResponse struct {
	Result   [][]string `json:"result"`
	Warnings []string   `json:"warnings,omitempty"`
	Error    string     `json:"error"`
}
