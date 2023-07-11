package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"

	gabi "github.com/app-sre/gabi/pkg"
	"github.com/app-sre/gabi/pkg/middleware"
	"github.com/app-sre/gabi/pkg/models"
)

const (
	base64EncodeResults byte = 1 << iota
	base64DecodeQuery
)

func Query(cfg *gabi.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var (
			base64Mode byte
			request    models.QueryRequest
		)

		if s := r.URL.Query().Get("base64_results"); s != "" {
			if ok, err := strconv.ParseBool(s); err == nil && ok {
				base64Mode |= base64EncodeResults
			}
		}

		if ctxQuery := ctx.Value(middleware.ContextKeyQuery); ctxQuery != nil {
			if s, ok := ctxQuery.(string); ok {
				request.Query = s
			}
		}
		if request.Query == "" {
			if s := r.URL.Query().Get("base64_query"); s != "" {
				if ok, err := strconv.ParseBool(s); err == nil && ok {
					base64Mode |= base64DecodeQuery
				}
			}

			err := json.NewDecoder(r.Body).Decode(&request)
			if err != nil {
				cfg.Logger.Errorf("Unable to decode request body: %s", err)
				if errors.Is(err, io.EOF) {
					http.Error(w, "Request body cannot be empty", http.StatusBadRequest)
					return
				}
				_ = queryErrorResponse(w, err)
				return
			}

			if base64Mode&base64DecodeQuery != 0 {
				bytes, err := cfg.Encoder.DecodeString(request.Query)
				if err != nil {
					l := "Unable to decode Base64-encoded query"
					cfg.Logger.Errorf("%s: %s", l, err)
					http.Error(w, l, http.StatusBadRequest)
					return
				}
				request.Query = string(bytes)
			}
		}

		tx, err := cfg.DB.BeginTx(ctx, &sql.TxOptions{
			ReadOnly: !cfg.DBEnv.AllowWrite,
		})
		if err != nil {
			cfg.Logger.Errorf("Unable to start database transaction: %s", err)
			_ = queryErrorResponse(w, err)
			return
		}
		defer func() { _ = tx.Rollback() }()

		rows, err := tx.QueryContext(ctx, request.Query)
		if err != nil {
			cfg.Logger.Errorf("Unable to query database: %s", err)
			_ = queryErrorResponse(w, err)
			return
		}
		defer func() { _ = rows.Close() }()

		// Remember to check err afterwards.
		cols, err := rows.Columns()
		if err != nil {
			cfg.Logger.Errorf("Unable to process database columns: %s", err)
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
				cfg.Logger.Errorf("Unable to process database rows: %s", err)
				_ = queryErrorResponse(w, err)
				return
			}

			var row []string

			for _, value := range vals {
				content, ok := reflect.ValueOf(value).Interface().(*sql.RawBytes)
				if !ok {
					err = fmt.Errorf("unable to convert value type %T to *sql.RawBytes", value)
					cfg.Logger.Errorf("Unable to process database query: %s", err)
					_ = queryErrorResponse(w, err)
					return
				}
				s := string(*content)

				if base64Mode&base64EncodeResults != 0 {
					s = cfg.Encoder.EncodeToString(*content)
				}
				row = append(row, s)
			}
			result = append(result, row)
		}

		err = rows.Err()
		if err != nil {
			cfg.Logger.Errorf("Unable to process database rows: %s", err)
			_ = queryErrorResponse(w, err)
			return
		}

		err = tx.Commit()
		if err != nil {
			cfg.Logger.Errorf("Unable to commit database changes: %s", err)
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

	err = json.NewEncoder(w).Encode(&models.QueryResponse{
		Error: err.Error(),
	})
	if err != nil {
		return fmt.Errorf("unable to marshal error response: %w", err)
	}
	return nil
}
