package handlers

import (
	"bytes"
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

// beginQuery parses the request, opens a transaction, runs the SQL, and reads column metadata.
// On failure it writes the HTTP response and returns ok == false. The caller must close rows and
// commit or roll back the transaction when ok is true.
func beginQuery(cfg *gabi.Config, w http.ResponseWriter, r *http.Request) (base64Mode byte, tx *sql.Tx, rows *sql.Rows, cols []string, ok bool) {
	ctx := r.Context()

	var request models.QueryRequest

	if s := r.URL.Query().Get("base64_results"); s != "" {
		if b, err := strconv.ParseBool(s); err == nil && b {
			base64Mode |= base64EncodeResults
		}
	}

	if ctxQuery := ctx.Value(middleware.ContextKeyQuery); ctxQuery != nil {
		if s, typed := ctxQuery.(string); typed {
			request.Query = s
		}
	}
	if request.Query == "" {
		if s := r.URL.Query().Get("base64_query"); s != "" {
			if b, err := strconv.ParseBool(s); err == nil && b {
				base64Mode |= base64DecodeQuery
			}
		}

		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			cfg.Logger.Errorf("Unable to decode request body: %s", err)
			if errors.Is(err, io.EOF) {
				http.Error(w, "Request body cannot be empty", http.StatusBadRequest)
				return 0, nil, nil, nil, false
			}
			_ = queryErrorResponse(w, err)
			return 0, nil, nil, nil, false
		}

		if base64Mode&base64DecodeQuery != 0 {
			decoded, err := cfg.Encoder.DecodeString(request.Query)
			if err != nil {
				l := "Unable to decode Base64-encoded query"
				cfg.Logger.Errorf("%s: %s", l, err)
				http.Error(w, l, http.StatusBadRequest)
				return 0, nil, nil, nil, false
			}
			request.Query = string(decoded)
		}
	}

	tx, err := cfg.DB.BeginTx(ctx, &sql.TxOptions{
		ReadOnly: !cfg.DBEnv.AllowWrite,
	})
	if err != nil {
		cfg.Logger.Errorf("Unable to start database transaction: %s", err)
		_ = queryErrorResponse(w, err)
		return 0, nil, nil, nil, false
	}

	rows, err = tx.QueryContext(ctx, request.Query)
	if err != nil {
		cfg.Logger.Errorf("Unable to query database: %s", err)
		_ = tx.Rollback()
		_ = queryErrorResponse(w, err)
		return 0, nil, nil, nil, false
	}

	cols, err = rows.Columns()
	if err != nil {
		cfg.Logger.Errorf("Unable to process database columns: %s", err)
		_ = rows.Close()
		_ = tx.Rollback()
		_ = queryErrorResponse(w, err)
		return 0, nil, nil, nil, false
	}

	return base64Mode, tx, rows, cols, true
}

func Query(cfg *gabi.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		base64Mode, tx, rows, cols, ok := beginQuery(cfg, w, r)
		if !ok {
			return
		}
		defer func() { _ = rows.Close() }()
		defer func() { _ = tx.Rollback() }()

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
			err := rows.Scan(vals...)
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
				content, typed := reflect.ValueOf(value).Interface().(*sql.RawBytes)
				if !typed {
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

		err := rows.Err()
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

// encodeStreamJSONArrayElement encodes one JSON value the same way json.Encoder does on an
// http.ResponseWriter (including a trailing newline).
func encodeStreamJSONArrayElement(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// StreamQuery streams result rows as JSON array elements to reduce peak memory for large results.
// The wire format is a single JSON object: {"result":[ <newline-separated encoded rows> ],"error":""}
// It is served at POST /streamquery (same middleware chain as POST /query).
func StreamQuery(cfg *gabi.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		base64Mode, tx, rows, cols, ok := beginQuery(cfg, w, r)
		if !ok {
			return
		}
		defer func() { _ = rows.Close() }()
		defer func() { _ = tx.Rollback() }()

		vals := make([]interface{}, len(cols))
		var keys []string
		for i := range cols {
			vals[i] = new(sql.RawBytes)
			keys = append(keys, cols[i])
		}

		bodyStarted := false
		writeStreamFatal := func(logMsg string, err error) {
			cfg.Logger.Errorf("%s: %s", logMsg, err)
			if !bodyStarted {
				_ = queryErrorResponse(w, err)
			}
		}

		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		headerRow, err := encodeStreamJSONArrayElement(keys)
		if err != nil {
			writeStreamFatal("Unable to encode JSON header row", err)
			return
		}

		if _, err = w.Write([]byte("{\"result\":[")); err != nil {
			writeStreamFatal("Unable to write JSON array start", err)
			return
		}
		bodyStarted = true

		if _, err = w.Write(headerRow); err != nil {
			cfg.Logger.Errorf("Unable to write JSON header row: %s", err)
			return
		}

		for rows.Next() {
			err = rows.Scan(vals...)
			if err != nil {
				cfg.Logger.Errorf("Unable to process database rows: %s", err)
				return
			}

			var row []string
			for _, value := range vals {
				content, typed := reflect.ValueOf(value).Interface().(*sql.RawBytes)
				if !typed {
					err = fmt.Errorf("unable to convert value type %T to *sql.RawBytes", value)
					cfg.Logger.Errorf("Unable to process database query: %s", err)
					return
				}
				s := string(*content)
				if base64Mode&base64EncodeResults != 0 {
					s = cfg.Encoder.EncodeToString(*content)
				}
				row = append(row, s)
			}

			rowBytes, err := encodeStreamJSONArrayElement(row)
			if err != nil {
				cfg.Logger.Errorf("Unable to encode JSON data row: %s", err)
				return
			}
			if _, err = w.Write([]byte(",")); err != nil {
				cfg.Logger.Errorf("Unable to write JSON row separator: %s", err)
				return
			}
			if _, err = w.Write(rowBytes); err != nil {
				cfg.Logger.Errorf("Unable to write JSON data row: %s", err)
				return
			}
		}

		if err = rows.Err(); err != nil {
			cfg.Logger.Errorf("Unable to process database rows: %s", err)
			return
		}

		if err = tx.Commit(); err != nil {
			cfg.Logger.Errorf("Unable to commit database changes: %s", err)
			return
		}

		if _, err = w.Write([]byte("],\"error\":\"\"}\n")); err != nil {
			cfg.Logger.Errorf("Unable to write JSON array close: %s", err)
			return
		}
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
