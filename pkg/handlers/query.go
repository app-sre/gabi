package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"

	gabi "github.com/app-sre/gabi/pkg"
	"github.com/app-sre/gabi/pkg/models"
)

func Query(env *gabi.Env) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request models.QueryRequest

		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			env.Logger.Errorf("Unable to decode request body: %s", err)
			if errors.Is(err, io.EOF) {
				http.Error(w, "Request body cannot be empty", http.StatusBadRequest)
				return
			}
			_ = queryErrorResponse(w, err)
			return
		}

		tx, err := env.DB.BeginTx(context.Background(), &sql.TxOptions{
			ReadOnly: !env.DBEnv.AllowWrite,
		})
		if err != nil {
			env.Logger.Errorf("Unable to start database transaction: %s", err)
			_ = queryErrorResponse(w, err)
			return
		}
		defer func() { _ = tx.Rollback() }()

		rows, err := tx.Query(request.Query)
		if err != nil {
			env.Logger.Errorf("Unable to query database: %s", err)
			_ = queryErrorResponse(w, err)
			return
		}
		defer func() { _ = rows.Close() }()

		cols, err := rows.Columns() // Remember to check err afterwards
		if err != nil {
			env.Logger.Errorf("Unable to process database query: %s", err)
			_ = queryErrorResponse(w, err)
			return
		}

		vals := make([]interface{}, len(cols))

		var (
			result [][]string
			keys   []string
		)

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
				env.Logger.Errorf("Unable to process database query: %s", err)
				_ = queryErrorResponse(w, err)
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
			env.Logger.Errorf("Unable to process database query: %s", err)
			_ = queryErrorResponse(w, err)
			return
		}

		err = tx.Commit()
		if err != nil {
			env.Logger.Errorf("Unable to commit database changes: %s", err)
			_ = queryErrorResponse(w, err)
			return
		}

		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(&models.QueryResponse{
			Result: result,
		})
	}
}

func queryErrorResponse(w http.ResponseWriter, err error) error {
	var (
		parseError   *url.Error
		syscallError *os.SyscallError
	)

	// Stop the SQL drivers from leaking credentials on connection errors.
	if errors.As(err, &parseError) || errors.As(err, &syscallError) {
		http.Error(w, "Unable to connect to the database", http.StatusServiceUnavailable)
		return nil
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)

	return json.NewEncoder(w).Encode(&models.QueryResponse{
		Error: err.Error(),
	})
}
