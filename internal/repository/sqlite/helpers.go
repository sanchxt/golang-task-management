package sqlite

import (
	"database/sql"
	"fmt"
	"strings"
)

func nullInt64(i *int64) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: *i, Valid: true}
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func buildINQuery(query string, ids []int64) (string, []interface{}) {
	if len(ids) == 0 {
		return query, []interface{}{}
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	placeholder := strings.Join(placeholders, ", ")
	query = strings.Replace(query, "(?)", fmt.Sprintf("(%s)", placeholder), 1)

	return query, args
}
