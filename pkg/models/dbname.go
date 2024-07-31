package models

type SwitchDBNameRequest struct {
	DBName string `json:"db_name"`
}

type DBNameResponse struct {
	DBName string `json:"db_name"`
}
