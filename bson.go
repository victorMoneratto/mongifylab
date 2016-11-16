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
	for _, table := range t.Root {
		buf.WriteString(sep)
		buf.WriteString("/* " + table.Name + " */\n")
		buf.WriteString("db.createCollection(\"" + table.Name + "\")\n")
		buf.WriteString("db." + table.Name + ".insert([")
		script, err := t.toBSON(table, db)
		if err != nil {
			return "", err
		}
		buf.WriteString(script)
		buf.WriteString("\n])\n")
		sep = "\n"
	}

	return buf.String(), nil
}

func (t *DependencyTree) toBSON(table *TableNode, db *sql.DB) (string, error) {
	// Query all rows
	query := t.QueryForAll(table)
	rows, err := db.Query(query)
	if err != nil {
		return "", err
	}

	cols := t.prepareColumns(db, table, false)

	rowMapChan, err := RowMapChan(rows)
	if err != nil {
		return "", err
	}

	// for each row on the table
	var buf bytes.Buffer
	for rowMap := range rowMapChan {
		buf.WriteString("\n\t")
		sep := "{"
		for _, col := range cols {
			buf.WriteString(sep)
			buf.WriteString(col.Bson(rowMap))
			sep = ", "
		}
		buf.WriteString("},")
	}

	return buf.String(), nil
}

type BsonColumn struct {
	Table        string
	Name         string
	InnerColumns []*BsonColumn
}

func NewColumn(table, name string) *BsonColumn {
	return &BsonColumn{Table: table, Name: name}
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
	} else if value := m[c.Table+"."+c.Name]; value != nil {
		buf.WriteString(c.Name + ": ")
		buf.WriteString(valueString(value))
	}

	return buf.String()
}

func (t *DependencyTree) prepareColumns(db *sql.DB, table *TableNode, isEmbedded bool) []*BsonColumn {
	var cols []*BsonColumn

	pks := t.Prepared.PKs[table.Name]
	fks := t.Prepared.FKs[table.Name]

	embeddedCols := make(map[string]*TableNode) // embeddedCols[ColumnName] = ForeignTableNode
	referencedCols := make(map[string]string)   // referencedCols[ColumnName] = ForeignTableName

	for _, embedded := range table.Embedded {
		if fk, found := fks[embedded.Name]; found {
			for iCol := range fk.Columns {
				embeddedCols[fk.Columns[iCol]] = embedded
			}
		}
	}

	for _, referenced := range table.Referenced {
		if fk, found := fks[referenced]; found {
			for iCol := range fk.Columns {
				referencedCols[fk.Columns[iCol]] = referenced
			}
		}
	}

	written := make(map[string]bool) // written[ForeignTableName] = bool

	PKParent := &cols
	if !isEmbedded {
		id := NewColumn("", "_id")
		cols = append(cols, id)
		PKParent = &id.InnerColumns
	}

	for _, pk := range t.Prepared.PKs[table.Name] {
		t.prepareSingleColumn(db, PKParent, table.Name, pk, embeddedCols, referencedCols, written)
	}

	nonPks := removeDuplicate(t.Prepared.Cols[table.Name], pks)
	for _, field := range nonPks {
		t.prepareSingleColumn(db, &cols, table.Name, field, embeddedCols, referencedCols, written)
	}

	return cols
}

func (t *DependencyTree) prepareSingleColumn(db *sql.DB, parent *[]*BsonColumn, table, col string, embeddedCols map[string]*TableNode, referencedCols map[string]string, written map[string]bool) {
	// embedded columns will be replaced with the whole other object
	if embedded, found := embeddedCols[col]; found {
		if !written[embedded.Name] {
			written[embedded.Name] = true
			embeddedCol := NewColumn("", embedded.Name)
			embeddedCol.InnerColumns = t.prepareColumns(db, embedded, true)

			*parent = append(*parent, embeddedCol)
		}

		// referenced columns will be replaced with a reference to the other object
	} else if referenced, found := referencedCols[col]; found {
		if !written[referenced] {
			written[referenced] = true
			referencedCol := NewColumn("", referenced)
			for _, referPK := range t.Prepared.PKs[referenced] {
				referencedCol.InnerColumns = append(referencedCol.InnerColumns, NewColumn(referenced, referPK))
			}

			*parent = append(*parent, referencedCol)
		}

		// column will be put plainly
	} else {
		*parent = append(*parent, NewColumn(table, col))
	}
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
