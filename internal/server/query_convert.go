package server

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/varavelio/nsqlite/internal/db"
	"github.com/varavelio/nsqlite/internal/sqlite"
	"github.com/varavelio/nsqlite/internal/vdl"
)

// authorizerRoleForAuthRole maps an auth role to the corresponding SQLite authorizer role.
func authorizerRoleForAuthRole(role authRole) sqlite.AuthorizerRole {
	switch role {
	case authRoleReadWrite:
		return sqlite.AuthorizerRoleReadWrite
	case authRoleReadOnly:
		return sqlite.AuthorizerRoleReadOnly
	default:
		return sqlite.AuthorizerRoleAdmin
	}
}

// sqliteParamsFromVDL converts VDL query parameters into SQLite query parameters.
func sqliteParamsFromVDL(params *[]vdl.QueryParam) ([]sqlite.QueryParam, error) {
	if params == nil {
		return nil, nil
	}

	converted := make([]sqlite.QueryParam, 0, len(*params))
	for _, param := range *params {
		value, err := sqliteValueFromVDL(param.Value)
		if err != nil {
			return nil, err
		}

		converted = append(converted, sqlite.QueryParam{
			Name:  vdl.Val(param.Name),
			Value: value,
		})
	}

	return converted, nil
}

// sqliteValueFromVDL converts a single VDL SqliteValue into a native Go value.
// Exactly one value field must be set; blobs are base64-decoded on the fly.
func sqliteValueFromVDL(value vdl.SqliteValue) (any, error) {
	fieldCount := 0
	var converted any

	if value.Null != nil && *value.Null {
		fieldCount++
		converted = nil
	}
	if value.Integer != nil {
		fieldCount++
		converted = *value.Integer
	}
	if value.Real != nil {
		fieldCount++
		converted = *value.Real
	}
	if value.Text != nil {
		fieldCount++
		converted = *value.Text
	}
	if value.Blob != nil {
		fieldCount++
		decoded, err := base64.StdEncoding.DecodeString(*value.Blob)
		if err != nil {
			return nil, fmt.Errorf("decode blob parameter: %w", err)
		}
		converted = decoded
	}

	if fieldCount != 1 {
		return nil, fmt.Errorf("exactly one SQLite value field must be set")
	}

	return converted, nil
}

// queryResultFromDB converts a db.QueryResult into a vdl.QueryResult.
func queryResultFromDB(startedAt time.Time, result db.QueryResult) vdl.QueryResult {
	queryResult := vdl.QueryResult{
		Type: queryResultTypeFromDB(result.Type),
		Time: time.Since(startedAt).Seconds(),
	}

	if result.TxID != "" {
		queryResult.TxId = &result.TxID
	}
	if result.Type == db.QueryTypeWrite {
		queryResult.LastInsertId = &result.LastInsertID
		queryResult.RowsAffected = &result.RowsAffected
	}
	if len(result.Columns) > 0 {
		columns := append([]string(nil), result.Columns...)
		queryResult.Columns = &columns
	}
	if len(result.Types) > 0 {
		types := make([]vdl.SqliteStorageClass, 0, len(result.Types))
		for _, valueType := range result.Types {
			types = append(types, sqliteStorageClassFromString(valueType))
		}
		queryResult.Types = &types
	}
	if len(result.Rows) > 0 {
		rows := make([][]vdl.SqliteValue, 0, len(result.Rows))
		for _, row := range result.Rows {
			convertedRow := make([]vdl.SqliteValue, 0, len(row))
			for _, value := range row {
				convertedRow = append(convertedRow, sqliteValueToVDL(value))
			}
			rows = append(rows, convertedRow)
		}
		queryResult.Rows = &rows
	}

	return queryResult
}

// queryResultTypeFromDB maps a db.QueryType to the corresponding vdl.QueryResultType.
func queryResultTypeFromDB(queryType db.QueryType) vdl.QueryResultType {
	switch queryType {
	case db.QueryTypeRead:
		return vdl.QueryResultTypeRead
	case db.QueryTypeWrite:
		return vdl.QueryResultTypeWrite
	case db.QueryTypeBegin:
		return vdl.QueryResultTypeBegin
	case db.QueryTypeCommit:
		return vdl.QueryResultTypeCommit
	case db.QueryTypeRollback:
		return vdl.QueryResultTypeRollback
	default:
		return vdl.QueryResultTypeError
	}
}

// sqliteStorageClassFromString maps a SQLite type affinity string to a vdl.SqliteStorageClass.
func sqliteStorageClassFromString(valueType string) vdl.SqliteStorageClass {
	upper := strings.ToUpper(strings.TrimSpace(valueType))

	switch upper {
	case "NULL":
		return vdl.SqliteStorageClassNull
	case "INTEGER":
		return vdl.SqliteStorageClassInteger
	case "REAL":
		return vdl.SqliteStorageClassReal
	case "TEXT":
		return vdl.SqliteStorageClassText
	case "BLOB":
		return vdl.SqliteStorageClassBlob
	}

	switch {
	case strings.Contains(upper, "INT"):
		return vdl.SqliteStorageClassInteger
	case strings.Contains(upper, "CHAR"),
		strings.Contains(upper, "CLOB"),
		strings.Contains(upper, "TEXT"),
		strings.Contains(upper, "DATE"),
		strings.Contains(upper, "TIME"):
		return vdl.SqliteStorageClassText
	case strings.Contains(upper, "BLOB") || upper == "":
		return vdl.SqliteStorageClassBlob
	case strings.Contains(upper, "REAL"),
		strings.Contains(upper, "FLOA"),
		strings.Contains(upper, "DOUB"),
		strings.Contains(upper, "NUMERIC"),
		strings.Contains(upper, "DECIMAL"),
		strings.Contains(upper, "BOOLEAN"):
		return vdl.SqliteStorageClassReal
	default:
		return vdl.SqliteStorageClassText
	}
}

// sqliteValueToVDL converts a native Go value into a vdl.SqliteValue.
// Integers are normalized to int64, floats to float64, and []byte to base64-encoded strings.
func sqliteValueToVDL(value any) vdl.SqliteValue {
	switch typed := value.(type) {
	case nil:
		isNull := true
		return vdl.SqliteValue{Null: &isNull}
	case int:
		converted := int64(typed)
		return vdl.SqliteValue{Integer: &converted}
	case int8:
		converted := int64(typed)
		return vdl.SqliteValue{Integer: &converted}
	case int16:
		converted := int64(typed)
		return vdl.SqliteValue{Integer: &converted}
	case int32:
		converted := int64(typed)
		return vdl.SqliteValue{Integer: &converted}
	case int64:
		return vdl.SqliteValue{Integer: &typed}
	case uint:
		converted := int64(typed)
		return vdl.SqliteValue{Integer: &converted}
	case uint8:
		converted := int64(typed)
		return vdl.SqliteValue{Integer: &converted}
	case uint16:
		converted := int64(typed)
		return vdl.SqliteValue{Integer: &converted}
	case uint32:
		converted := int64(typed)
		return vdl.SqliteValue{Integer: &converted}
	case uint64:
		converted := int64(typed)
		return vdl.SqliteValue{Integer: &converted}
	case float32:
		converted := float64(typed)
		return vdl.SqliteValue{Real: &converted}
	case float64:
		return vdl.SqliteValue{Real: &typed}
	case string:
		return vdl.SqliteValue{Text: &typed}
	case []byte:
		converted := base64.StdEncoding.EncodeToString(typed)
		return vdl.SqliteValue{Blob: &converted}
	default:
		fallback := fmt.Sprint(typed)
		return vdl.SqliteValue{Text: &fallback}
	}
}
