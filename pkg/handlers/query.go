package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"reflect"
	"strconv"
	"time"

	gabi "github.com/app-sre/gabi/pkg"
	"github.com/app-sre/gabi/pkg/audit"
)

type QueryRequest struct {
	Query string
}

type QueryResponse struct {
	Result [][]string `json:"result"`
	Error string      `json:"error"`
}

func Query(env *gabi.Env) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var q QueryRequest
		var ret QueryResponse

		err := json.NewDecoder(r.Body).Decode(&q)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		now := time.Now()
		user := r.Header.Get("X-Forwarded-User")
		if user == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		qd := &audit.QueryData{
			Query:     q.Query,
			User:      user,
			Timestamp: now.Unix(),
		}

		env.Audit.Write(qd)

		sed := &audit.SplunkEventData{
			Query: q.Query,
			User: user,
		}

		sqd := &audit.SplunkQueryData{
			Event: sed,
			Time: now.Unix(),
		}

		resp, err := env.SplunkAudit.Write(sqd)
		if err != nil {
			ret.Error = err.Error()
			json.NewEncoder(w).Encode(ret)
			return
		} else if resp.Code != 0 {
			ret.Error = "Splunk error: " + resp.Text + " - Code: " + strconv.Itoa(resp.Code)
			json.NewEncoder(w).Encode(ret)
			return
		}

		rows, err := env.DB.Query(q.Query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer rows.Close()

		cols, err := rows.Columns() // Remember to check err afterwards
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		vals := make([]interface{}, len(cols))

		var result [][]string
		var keys []string

		for i := range cols {
			vals[i] = new(sql.RawBytes)
			keys = append(keys, cols[i])
		}

		result = append(result, keys)

		for rows.Next() {
			err = rows.Scan(vals...)
			// Now you can check each element of vals for nil-ness,
			// and you can use type introspection and type assertions
			// to fetch the column into a typed variable.
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			var row []string

			for _, value := range vals {
				content := reflect.ValueOf(value).Interface().(*sql.RawBytes)
				row = append(row, string(*content))
			}
			result = append(result, row)
		}

		err = rows.Err()
		if err != nil {
			ret.Error = err.Error()
			json.NewEncoder(w).Encode(ret)
			return
		}

		ret.Result = result
		ret.Error = ""
		json.NewEncoder(w).Encode(ret)
	}
}
