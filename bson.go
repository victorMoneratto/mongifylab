package mongifylab

import (
	"bytes"
	"database/sql"
	"fmt"
	"time"
)

// CreateCollectionScript returns the script for creating and populating the
// a corresponding collection on mongodb
func CreateCollectionScript(db *sql.DB, table string) (string, error) {
	var buf bytes.Buffer
	buf.WriteString("db.createCollection(\"" + table + "\")\n")

	rows, err := db.Query("SELECT * FROM " + table)
	if err != nil {
		return "", err
	}

	columns, err := rows.Columns()
	if err != nil {
		return "", err
	}

	pks, _, _, err := QueryConstraints(db, table)
	if err != nil {
		return "", err
	}
	nonPks := removePks(columns, pks)

	rowMapChan, err := RowMapChan(rows)
	if err != nil {
		return "", err
	}

	// for each row on the table
	for rowMap := range rowMapChan {
		buf.WriteString("db." + table + ".insert({_id:{")

		// Put all primary keys in _id (there's always one field or more)
		writeFields(&buf, rowMap, pks)

		// If there are fields remaining, write them outside _id
		if len(nonPks) > 0 {
			buf.WriteString(", ")
			writeFields(&buf, rowMap, nonPks)
		}

		buf.WriteString(")\n")
	}

	return buf.String(), nil
}

// writeFields writes 'COLUMN1: val1, COLUMN2: "val2"...}' to the buffer
func writeFields(buf *bytes.Buffer, values map[string]interface{}, columns []string) {
	written := false
	for _, column := range columns {
		value := values[column]
		if value != nil {
			if written {
				buf.WriteString(", ")
			}

			written = true
			valueStr := valueString(values[column])
			buf.WriteString(column + ": " + valueStr)
		}
	}

	buf.WriteString("}")
}

// valueString converts values to its adequate string representation for mongodb
func valueString(val interface{}) string {
	switch val.(type) {
	case string:
		return "\"" + val.(string) + "\""
	case time.Time:
		t := val.(time.Time)
		return "new Date(\"" + t.Format("2006-01-02") + "\")"
	default:
		return fmt.Sprint(val)
	}
}

// removePks returns columns that dont belong to the primary key
func removePks(columns, pks []string) []string {
	pksMap := make(map[string]bool, len(pks))
	for _, pk := range pks {
		pksMap[pk] = true
	}

	var nonPks []string
	for _, col := range columns {
		if !pksMap[col] {
			nonPks = append(nonPks, col)
		}
	}

	return nonPks
}
