package models

type QueryRequest struct {
	Query string
}

type QueryResponse struct {
	Result [][]string `json:"result"`
	Error  string     `json:"error"`
}
