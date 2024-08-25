package repo

import (
	"database/sql"
	//"fmt"

	sq "github.com/Masterminds/squirrel"

	"github.com/jackc/pgx/v5"

	//"github.com/jackc/pgx/v5"
	//"github.com/jackc/pgx/v5/pgxpool"

	"context"

	"reflect"
	//"time"
	"github.com/templatedop/ftptemplate/db"

	"github.com/Masterminds/squirrel"
)

/**
 * psql holds a reference to squirrel.StatementBuilderType
 * which is used to build SQL queries that compatible with PostgreSQL syntax
 */
var Psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

// nullString converts a string to sql.NullString for empty string check
func nullString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}

	return sql.NullString{
		String: value,
		Valid:  true,
	}
}

// nullUint64 converts an uint64 to sql.NullInt64 for empty uint64 check
func nullUint64(value uint64) sql.NullInt64 {
	if value == 0 {
		return sql.NullInt64{}
	}

	valueInt64 := int64(value)

	return sql.NullInt64{
		Int64: valueInt64,
		Valid: true,
	}
}

// nullInt64 converts an int64 to sql.NullInt64 for empty int64 check
func nullInt64(value int64) sql.NullInt64 {
	if value == 0 {
		return sql.NullInt64{}
	}

	return sql.NullInt64{
		Int64: value,
		Valid: true,
	}
}

// nullFloat64 converts a float64 to sql.NullFloat64 for empty float64 check
func nullFloat64(value float64) sql.NullFloat64 {
	if value == 0 {
		return sql.NullFloat64{}
	}

	return sql.NullFloat64{
		Float64: value,
		Valid:   true,
	}
}

func QueryWithSquirrel[T any](ctx context.Context, db *db.DB, stmt squirrel.SelectBuilder, resultSlice interface{}) error {
	sql, args, err := stmt.ToSql()
	if err != nil {
		return err
	}

	rows, err := db.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	results := reflect.ValueOf(resultSlice)
	elementType := results.Type().Elem()
	//fmt.Println("Rows", rows)
	for rows.Next() {
		result := reflect.New(elementType).Interface()
		if err := rows.Scan(result); err != nil {
			return err
		}
		results = reflect.Append(results, reflect.ValueOf(result).Elem())
	}

	return nil
}

type TableSchema struct {
	TableName   string
	FieldNames  []string
	FieldTypes  []reflect.Type
	FieldValues []any
}

func copyRows[T any](rows pgx.Rows, destination []*T, schema *TableSchema) error {
	numFields := len(schema.FieldNames)

	for i := 0; i < numFields; i++ {
		schema.FieldValues[i] = reflect.New(schema.FieldTypes[i]).Interface()
	}

	for rows.Next() {
		err := rows.Scan(schema.FieldValues...)
		if err != nil {
			return err
		}

		for i, fieldValue := range schema.FieldValues {
			destElement := reflect.ValueOf(destination).Elem()
			field := destElement.FieldByName(schema.FieldNames[i])
			field.Set(reflect.ValueOf(fieldValue))
		}
	}

	return nil
}

// Select executes sql with args on db and returns the []T produced by scanFn.
