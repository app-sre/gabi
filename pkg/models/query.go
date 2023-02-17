package models

type QueryRequest struct {
	Query string `json:"query"`
}

type QueryResponse struct {
	Result [][]string `json:"result"`
	Error  string     `json:"error"`
}
