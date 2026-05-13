package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	stdjson "encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/varavelio/nsqlite/internal/util/httputil"
	"github.com/varavelio/nsqlite/internal/vdl"
)

type rqliteEndpointMode string

const (
	rqliteModeQuery   rqliteEndpointMode = "query"
	rqliteModeExecute rqliteEndpointMode = "execute"
	rqliteModeRequest rqliteEndpointMode = "request"
)

// rqliteResponse is the compatibility response shape expected by rqlite clients.
type rqliteResponse struct {
	Results []rqliteResult `json:"results"`
	Time    *float64       `json:"time,omitempty"`
}

// rqliteResult is one result entry in a rqlite compatibility response.
type rqliteResult struct {
	LastInsertID *int64           `json:"last_insert_id,omitempty"`
	RowsAffected *int64           `json:"rows_affected,omitempty"`
	Columns      []string         `json:"columns,omitempty"`
	Types        any              `json:"types,omitempty"`
	Values       [][]any          `json:"values,omitempty"`
	Rows         []map[string]any `json:"rows,omitempty"`
	Error        string           `json:"error,omitempty"`
	Time         *float64         `json:"time,omitempty"`
}

type rqliteResponseOptions struct {
	associative bool
	blobArray   bool
	timings     bool
}

// rqliteQueryHandler handles rqlite-compatible read requests.
func (s *Server) rqliteQueryHandler(w http.ResponseWriter, r *http.Request) error {
	return s.handleRQLite(w, r, rqliteModeQuery)
}

// rqliteExecuteHandler handles rqlite-compatible write requests.
func (s *Server) rqliteExecuteHandler(w http.ResponseWriter, r *http.Request) error {
	return s.handleRQLite(w, r, rqliteModeExecute)
}

// rqliteRequestHandler handles rqlite-compatible mixed read/write requests.
func (s *Server) rqliteRequestHandler(w http.ResponseWriter, r *http.Request) error {
	return s.handleRQLite(w, r, rqliteModeRequest)
}

// handleRQLite translates rqlite-compatible requests into NSQLite query execution.
func (s *Server) handleRQLite(
	w http.ResponseWriter,
	r *http.Request,
	mode rqliteEndpointMode,
) error {
	props, err := s.authorizeRQLiteRequest(r)
	if err != nil {
		return err
	}

	queries, err := rqliteQueriesFromRequest(r)
	if err != nil {
		return badRQLiteRequest(err)
	}

	if mode == rqliteModeQuery {
		props.Role = authRoleReadOnly
	}

	s.DBStats.IncHTTPRequests()
	s.DBStats.IncQueuedHTTPRequests()
	defer s.DBStats.DecQueuedHTTPRequests()

	startedAt := time.Now()
	output := s.executeRQLiteQueries(r.Context(), props, queries, r.URL.Query().Has("transaction"))
	options := rqliteResponseOptions{
		associative: r.URL.Query().Has("associative"),
		blobArray:   r.URL.Query().Has("blob_array"),
		timings:     r.URL.Query().Has("timings"),
	}

	response := rqliteResponseFromVDL(output, options, time.Since(startedAt))
	return writeRQLiteJSON(w, response, r.URL.Query().Has("pretty"))
}

// authorizeRQLiteRequest authenticates rqlite compatibility requests.
// Basic auth is accepted by treating the password as the configured NSQLite token.
func (s *Server) authorizeRQLiteRequest(r *http.Request) (requestProps, error) {
	if s.authIsDisabled() {
		return requestProps{Role: authRoleAdmin}, nil
	}

	_, password, ok := r.BasicAuth()
	if ok {
		role, principal, authenticated := s.checkAuthWithCache(password)
		if !authenticated {
			return requestProps{}, unauthorizedError()
		}
		return requestProps{Role: role, Principal: principal}, nil
	}

	role, principal, err := s.authenticateRequest(r)
	if err != nil {
		return requestProps{}, err
	}

	return requestProps{Role: role, Principal: principal}, nil
}

// executeRQLiteQueries executes translated rqlite statements.
// When transaction is enabled, statement results exclude the internal BEGIN and
// COMMIT/ROLLBACK statements so the response matches rqlite's API shape.
func (s *Server) executeRQLiteQueries(
	ctx context.Context,
	props requestProps,
	queries []vdl.Query,
	transaction bool,
) vdl.DatabaseQueryOutput {
	if !transaction || len(queries) == 0 {
		return s.executeQueries(ctx, props, queries)
	}

	startedAt := time.Now()
	begin := s.executeRequestQuery(ctx, props, vdl.Query{Query: "BEGIN"})
	if begin.Error != nil || begin.TxId == nil {
		return vdl.DatabaseQueryOutput{
			Time:    time.Since(startedAt).Seconds(),
			Results: []vdl.QueryResult{begin},
		}
	}

	txID := *begin.TxId
	results := make([]vdl.QueryResult, 0, len(queries))
	for _, query := range queries {
		query.TxId = &txID
		result := s.executeRequestQuery(ctx, props, query)
		results = append(results, result)
		if result.Error != nil {
			_ = s.executeRequestQuery(ctx, props, vdl.Query{TxId: &txID, Query: "ROLLBACK"})
			return vdl.DatabaseQueryOutput{Time: time.Since(startedAt).Seconds(), Results: results}
		}
	}

	commit := s.executeRequestQuery(ctx, props, vdl.Query{TxId: &txID, Query: "COMMIT"})
	if commit.Error != nil {
		results = append(results, commit)
	}

	return vdl.DatabaseQueryOutput{Time: time.Since(startedAt).Seconds(), Results: results}
}

// rqliteQueriesFromRequest converts a rqlite HTTP request into VDL queries.
func rqliteQueriesFromRequest(r *http.Request) ([]vdl.Query, error) {
	if r.Method == http.MethodGet {
		values := r.URL.Query()["q"]
		if len(values) == 0 {
			return nil, errors.New("missing q query parameter")
		}

		queries := make([]vdl.Query, 0, len(values))
		for _, value := range values {
			queries = append(queries, vdl.Query{Query: value})
		}
		return queries, nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("read request body: %w", err)
	}
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil, errors.New("request body is required")
	}

	contentType := strings.ToLower(
		strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]),
	)
	if contentType == "text/plain" {
		return []vdl.Query{{Query: string(body)}}, nil
	}

	return rqliteQueriesFromJSON(body)
}

// rqliteQueriesFromJSON parses the rqlite JSON statement array format.
func rqliteQueriesFromJSON(body []byte) ([]vdl.Query, error) {
	var statements []stdjson.RawMessage
	decoder := stdjson.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&statements); err != nil {
		return nil, fmt.Errorf("decode JSON statements: %w", err)
	}

	queries := make([]vdl.Query, 0, len(statements))
	for _, statement := range statements {
		query, err := rqliteQueryFromJSONStatement(statement)
		if err != nil {
			return nil, err
		}
		queries = append(queries, query)
	}

	return queries, nil
}

// rqliteQueryFromJSONStatement parses one raw or parameterized rqlite statement.
func rqliteQueryFromJSONStatement(statement stdjson.RawMessage) (vdl.Query, error) {
	var sql string
	if err := stdjson.Unmarshal(statement, &sql); err == nil {
		return vdl.Query{Query: sql}, nil
	}

	var parts []stdjson.RawMessage
	decoder := stdjson.NewDecoder(bytes.NewReader(statement))
	decoder.UseNumber()
	if err := decoder.Decode(&parts); err != nil {
		return vdl.Query{}, fmt.Errorf("decode parameterized statement: %w", err)
	}
	if len(parts) == 0 {
		return vdl.Query{}, errors.New("statement array must not be empty")
	}
	if err := stdjson.Unmarshal(parts[0], &sql); err != nil || strings.TrimSpace(sql) == "" {
		return vdl.Query{}, errors.New("statement array must start with a SQL string")
	}

	params, err := rqliteParamsFromJSON(parts[1:])
	if err != nil {
		return vdl.Query{}, err
	}

	return vdl.Query{Query: sql, Params: params}, nil
}

// rqliteParamsFromJSON converts rqlite positional or named parameters into VDL parameters.
func rqliteParamsFromJSON(rawParams []stdjson.RawMessage) (*[]vdl.QueryParam, error) {
	if len(rawParams) == 0 {
		return nil, nil
	}

	if len(rawParams) == 1 && bytes.HasPrefix(bytes.TrimSpace(rawParams[0]), []byte("{")) {
		var named map[string]stdjson.RawMessage
		decoder := stdjson.NewDecoder(bytes.NewReader(rawParams[0]))
		decoder.UseNumber()
		if err := decoder.Decode(&named); err != nil {
			return nil, fmt.Errorf("decode named parameters: %w", err)
		}

		params := make([]vdl.QueryParam, 0, len(named))
		for name, rawValue := range named {
			value, err := rqliteValueFromJSON(rawValue)
			if err != nil {
				return nil, err
			}
			paramName := name
			params = append(params, vdl.QueryParam{Name: &paramName, Value: value})
		}

		return &params, nil
	}

	params := make([]vdl.QueryParam, 0, len(rawParams))
	for _, rawValue := range rawParams {
		value, err := rqliteValueFromJSON(rawValue)
		if err != nil {
			return nil, err
		}
		params = append(params, vdl.QueryParam{Value: value})
	}

	return &params, nil
}

// rqliteValueFromJSON converts one JSON parameter value into a VDL SQLite value.
func rqliteValueFromJSON(rawValue stdjson.RawMessage) (vdl.SqliteValue, error) {
	trimmed := bytes.TrimSpace(rawValue)
	if bytes.Equal(trimmed, []byte("null")) {
		isNull := true
		return vdl.SqliteValue{Null: &isNull}, nil
	}
	if bytes.HasPrefix(trimmed, []byte("[")) {
		var bytesValue []byte
		if err := stdjson.Unmarshal(trimmed, &bytesValue); err == nil {
			blob := encodeBlob(bytesValue)
			return vdl.SqliteValue{Blob: &blob}, nil
		}
	}

	decoder := stdjson.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return vdl.SqliteValue{}, fmt.Errorf("decode parameter value: %w", err)
	}

	switch typed := value.(type) {
	case bool:
		integer := int64(0)
		if typed {
			integer = 1
		}
		return vdl.SqliteValue{Integer: &integer}, nil
	case stdjson.Number:
		if integer, err := typed.Int64(); err == nil {
			return vdl.SqliteValue{Integer: &integer}, nil
		}
		real, err := typed.Float64()
		if err != nil {
			return vdl.SqliteValue{}, fmt.Errorf("decode numeric parameter: %w", err)
		}
		return vdl.SqliteValue{Real: &real}, nil
	case string:
		if blob, ok := rqliteHexBlobString(typed); ok {
			return vdl.SqliteValue{Blob: &blob}, nil
		}
		return vdl.SqliteValue{Text: &typed}, nil
	default:
		return vdl.SqliteValue{}, fmt.Errorf("unsupported parameter value %T", value)
	}
}

// rqliteHexBlobString converts rqlite X'hex' blob parameter strings into base64 blobs.
func rqliteHexBlobString(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) < 3 || (trimmed[0] != 'x' && trimmed[0] != 'X') || trimmed[1] != '\'' ||
		trimmed[len(trimmed)-1] != '\'' {
		return "", false
	}

	decoded, err := hex.DecodeString(trimmed[2 : len(trimmed)-1])
	if err != nil {
		return "", false
	}

	return encodeBlob(decoded), true
}

// rqliteResponseFromVDL converts NSQLite query output into the rqlite response shape.
func rqliteResponseFromVDL(
	output vdl.DatabaseQueryOutput,
	options rqliteResponseOptions,
	elapsed time.Duration,
) rqliteResponse {
	results := make([]rqliteResult, 0, len(output.Results))
	for _, result := range output.Results {
		results = append(results, rqliteResultFromVDL(result, options))
	}

	response := rqliteResponse{Results: results}
	if options.timings {
		total := elapsed.Seconds()
		if output.Time > 0 {
			total = output.Time
		}
		response.Time = &total
	}

	return response
}

// rqliteResultFromVDL converts one VDL query result into one rqlite result entry.
func rqliteResultFromVDL(result vdl.QueryResult, options rqliteResponseOptions) rqliteResult {
	converted := rqliteResult{}
	if options.timings {
		converted.Time = &result.Time
	}
	if result.Error != nil {
		converted.Error = *result.Error
		return converted
	}
	if result.LastInsertId != nil {
		converted.LastInsertID = result.LastInsertId
	}
	if result.RowsAffected != nil {
		converted.RowsAffected = result.RowsAffected
	}

	columns := vdl.Val(result.Columns)
	types := vdl.Val(result.Types)
	if len(columns) > 0 {
		if options.associative {
			converted.Types = rqliteAssociativeTypes(columns, types)
			converted.Rows = rqliteAssociativeRows(columns, vdl.Val(result.Rows), options)
		} else {
			converted.Columns = append([]string(nil), columns...)
			converted.Types = rqliteTypes(types)
			converted.Values = rqliteValues(vdl.Val(result.Rows), options)
		}
	}

	return converted
}

// rqliteTypes converts VDL storage classes into rqlite's lowercase type strings.
func rqliteTypes(types []vdl.SqliteStorageClass) []string {
	converted := make([]string, 0, len(types))
	for _, valueType := range types {
		converted = append(converted, strings.ToLower(valueType.String()))
	}
	return converted
}

// rqliteAssociativeTypes maps columns to lowercase rqlite type strings.
func rqliteAssociativeTypes(columns []string, types []vdl.SqliteStorageClass) map[string]string {
	converted := make(map[string]string, len(columns))
	for index, column := range columns {
		valueType := ""
		if index < len(types) {
			valueType = strings.ToLower(types[index].String())
		}
		converted[column] = valueType
	}
	return converted
}

// rqliteValues converts VDL rows into rqlite's array-of-arrays value shape.
func rqliteValues(rows [][]vdl.SqliteValue, options rqliteResponseOptions) [][]any {
	convertedRows := make([][]any, 0, len(rows))
	for _, row := range rows {
		convertedRow := make([]any, 0, len(row))
		for _, value := range row {
			convertedRow = append(convertedRow, rqliteValue(value, options))
		}
		convertedRows = append(convertedRows, convertedRow)
	}
	return convertedRows
}

// rqliteAssociativeRows converts VDL rows into rqlite's array-of-maps row shape.
func rqliteAssociativeRows(
	columns []string,
	rows [][]vdl.SqliteValue,
	options rqliteResponseOptions,
) []map[string]any {
	convertedRows := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		convertedRow := make(map[string]any, len(columns))
		for index, column := range columns {
			if index < len(row) {
				convertedRow[column] = rqliteValue(row[index], options)
			}
		}
		convertedRows = append(convertedRows, convertedRow)
	}
	return convertedRows
}

// rqliteValue converts one VDL SQLite value into its rqlite JSON value representation.
func rqliteValue(value vdl.SqliteValue, options rqliteResponseOptions) any {
	if value.Null != nil && *value.Null {
		return nil
	}
	if value.Integer != nil {
		return *value.Integer
	}
	if value.Real != nil {
		return *value.Real
	}
	if value.Text != nil {
		return *value.Text
	}
	if value.Blob != nil {
		if options.blobArray {
			return decodeBlobArray(*value.Blob)
		}
		return *value.Blob
	}
	return nil
}

// decodeBlobArray converts a base64-encoded VDL blob into rqlite's byte-array form.
func decodeBlobArray(encoded string) []int {
	decoded, err := decodeBlob(encoded)
	if err != nil {
		return nil
	}

	values := make([]int, 0, len(decoded))
	for _, value := range decoded {
		values = append(values, int(value))
	}
	return values
}

// badRQLiteRequest returns a standard 400 error for malformed compatibility requests.
func badRQLiteRequest(err error) error {
	return httputil.NewJSONError(http.StatusBadRequest, err, "Bad Request")
}

// writeRQLiteJSON writes a compact or pretty-printed rqlite compatibility response.
func writeRQLiteJSON(w http.ResponseWriter, response rqliteResponse, pretty bool) error {
	var (
		body []byte
		err  error
	)
	if pretty {
		body, err = stdjson.MarshalIndent(response, "", "  ")
	} else {
		body, err = stdjson.Marshal(response)
	}
	if err != nil {
		return fmt.Errorf("marshal rqlite response: %w", err)
	}

	return httputil.WriteJSONBytes(w, http.StatusOK, body)
}

// encodeBlob converts rqlite byte-array parameters into VDL's base64 blob encoding.
func encodeBlob(value []byte) string {
	return base64.StdEncoding.EncodeToString(value)
}

// decodeBlob decodes VDL's base64 blob encoding.
func decodeBlob(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}
