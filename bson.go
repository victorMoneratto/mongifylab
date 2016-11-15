package mongifylab

import (
	"bytes"
	"database/sql"
	"fmt"
	"time"
)

// CreateCollectionScript returns the script for creating and populating the
// a corresponding collection on mongodb
func (t *DependencyTree) CreateCollectionScript(db *sql.DB) (string, error) {
	var buf bytes.Buffer
	sep := ""
	for _, table := range t.Tables {
		buf.WriteString(sep)
		buf.WriteString("/* " + table.Name + " */\n")
		buf.WriteString("db.createCollection(\"" + table.Name + "\")\n")
		buf.WriteString("db." + table.Name + ".insert([")
		script, err := t.toBSON(table, db)
		if err != nil {
			return "", err
		}
		buf.WriteString(script)
		buf.WriteString("])\n")
		sep = "\n"
	}

	return buf.String(), nil
}

func (t *DependencyTree) toBSON(table *TableNode, db *sql.DB) (string, error) {
	// Query all rows
	rows, err := db.Query(t.QueryForAll(table))
	if err != nil {
		return "", err
	}

	cols, err := t.prepareColumns(db, table)
	if err != nil {
		return "", err
	}

	rowMapChan, err := RowMapChan(rows)
	if err != nil {
		return "", err
	}

	// for each row on the table
	var buf bytes.Buffer
	for rowMap := range rowMapChan {
		sep := "{"
		for _, col := range cols {
			buf.WriteString(sep)
			buf.WriteString(col.Bson(rowMap))
			sep = ", "
		}
		buf.WriteString("}, ")
	}

	return buf.String(), nil
}

type BsonColumn struct {
	Name         string
	InnerColumns []*BsonColumn
}

func NewColumn(name string) *BsonColumn {
	return &BsonColumn{Name: name}
}

func (c *BsonColumn) Bson(m map[string]interface{}) string {
	var buf bytes.Buffer

	if len(c.InnerColumns) > 0 {
		buf.WriteString(c.Name + ": {")
		sep := ""
		for _, inner := range c.InnerColumns {
			buf.WriteString(sep)
			buf.WriteString(inner.Bson(m))
			sep = ", "
		}
		buf.WriteRune('}')
	} else if value := m[c.Name]; value != nil {
		buf.WriteString(c.Name + ": ")
		buf.WriteString(valueString(value))
	}

	return buf.String()
}

func (t *DependencyTree) prepareColumns(db *sql.DB, table *TableNode) ([]*BsonColumn, error) {
	var cols []*BsonColumn
	pks, _, _, err := QueryConstraints(db, table.Name)
	if err != nil {
		return nil, err
	}

	id := NewColumn("_id")
	cols = append(cols, id)
	for _, pk := range pks {
		id.InnerColumns = append(id.InnerColumns, NewColumn(pk))
	}

	nonPks := removeDuplicate(t.Prepared.Cols[table.Name], pks)
	for _, field := range nonPks {
		cols = append(cols, NewColumn(field))
	}

	return cols, nil
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
func removeDuplicate(columns, pks []string) []string {
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
