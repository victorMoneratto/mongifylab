package mongifylab

import (
	"bytes"
	"database/sql"
	"log"
	"strconv"
)

// ListTables returns all relevant tables
func ListTables(db *sql.DB) ([]string, error) {
	query := `SELECT TABLE_NAME FROM USER_TABLES
	WHERE TABLE_NAME LIKE 'LE%'
	ORDER BY TABLE_NAME ASC`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}

	var tables []string
	for rows.Next() {
		var table string
		rows.Scan(&table)
		tables = append(tables, table)
	}

	return tables, rows.Err()
}

// RowSliceChan returns an unbuffered channel that sends each row as a []interface{}
func RowSliceChan(rows *sql.Rows) (<-chan []interface{}, error) {
	colNames, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// Allocate an interface{} per column and save its adress
	ptrs := make([]interface{}, len(colNames))
	for i := range ptrs {
		var ii interface{}
		ptrs[i] = &ii
	}

	rowChan := make(chan []interface{}, 1)

	go func() {
		for rows.Next() {
			// Try scanning value
			err := rows.Scan(ptrs...)
			if err != nil {
				// TODO pipe these into a separate channel for errors and remove logging
				log.Println("scanning:", err)
				continue
			}

			// save onto a map (column:value)
			results := make([]interface{}, len(colNames))
			for i := range colNames {
				results[i] = *(ptrs[i]).(*interface{})
			}

			rowChan <- results
		}
		close(rowChan)
	}()

	return rowChan, nil
}

// RowMapChan returns an unbuffered channel that sends each row as a map (column -> value)
func RowMapChan(rows *sql.Rows) (<-chan map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	ptrs, err := allocateForScan(len(columns))
	if err != nil {
		return nil, err
	}

	rowChan := make(chan map[string]interface{}, 1)

	go func() {
		for rows.Next() {
			// Try scanning value
			err := rows.Scan(ptrs...)
			if err != nil {
				// TODO pipe these into a separate channel for errors and remove logging
				log.Println("scanning:", err)
				continue
			}

			// save onto a map (column:value)
			results := make(map[string]interface{}, len(ptrs))
			for i := range ptrs {
				results[columns[i]] = *(ptrs[i]).(*interface{})
			}

			rowChan <- results
		}
		close(rowChan)
	}()

	return rowChan, nil
}

// Allocate an interface{} per column and save its address
func allocateForScan(size int) ([]interface{}, error) {
	ptrs := make([]interface{}, size)
	for i := range ptrs {
		var ii interface{}
		ptrs[i] = &ii
	}

	return ptrs, nil
}

// FKInfo is the relation between foreign key columns
type FKInfo struct {
	// Table   string
	// ForeignTable   string
	Columns        []string
	ForeignColumns []string
}

// QueryConstraints returns a map relating the column name to all it's constraints
func QueryConstraints(db *sql.DB, table string) (pks []string, fks map[string]FKInfo, uns [][]string, err error) {
	query := `SELECT CONS.CONSTRAINT_NAME, CONS.CONSTRAINT_TYPE, COLS.COLUMN_NAME, FK.TABLE_NAME, FK.COLUMN_NAME
	FROM USER_CONSTRAINTS CONS
	LEFT JOIN USER_CONS_COLUMNS COLS ON CONS.CONSTRAINT_NAME = COLS.CONSTRAINT_NAME
	LEFT JOIN USER_CONS_COLUMNS FK ON FK.CONSTRAINT_NAME = CONS.R_CONSTRAINT_NAME AND FK.POSITION = COLS.POSITION
	WHERE (CONSTRAINT_TYPE = 'P' OR CONSTRAINT_TYPE = 'R' OR CONSTRAINT_TYPE = 'U') AND COLS.TABLE_NAME = (:t)
	ORDER BY COLS.TABLE_NAME, COLS.COLUMN_NAME`

	rows, err := db.Query(query, table)
	if err != nil {
		return nil, nil, nil, err
	}
	rowsChan, err := RowSliceChan(rows)
	if err != nil {
		return nil, nil, nil, err
	}

	// reference FK and UN constraints by its name
	// important because these can be separate into multiple rows
	// fkMap := make(map[string]FKInfo)
	fks = make(map[string]FKInfo)
	unMap := make(map[string][]string)

	for row := range rowsChan {
		index := 0
		constraintName := row[index].(string)
		index++
		constraintType := row[index].(string)[0]
		index++
		columnName := row[index].(string)
		index++
		fkTable := row[index].(string)
		index++
		fkColumn := row[index].(string)

		switch constraintType {
		// append to primary key slice
		case 'P':
			pks = append(pks, columnName)

		// append to the correspondent unique constraint
		case 'U':
			unSlice := unMap[constraintName]
			unSlice = append(unSlice, columnName)
			unMap[constraintName] = unSlice

		// append to the correspondent foreign key constraint
		case 'R':
			info := fks[fkTable]
			info.Columns = append(info.Columns, columnName)
			info.ForeignColumns = append(info.ForeignColumns, fkColumn)
			fks[fkTable] = info
		}
	}

	for _, v := range unMap {
		uns = append(uns, v)
	}

	return pks, fks, uns, nil
}

func QueryColumnNames(db *sql.DB, table string) ([]string, error) {
	query := `SELECT COLUMN_NAME FROM USER_TAB_COLS
	WHERE TABLE_NAME = (:t)
	ORDER BY COLUMN_ID`

	rows, err := db.Query(query, table)
	if err != nil {
		return nil, err
	}
	var cols []string
	var col string
	for rows.Next() {
		err := rows.Scan(&col)
		if err != nil {
			continue
		}
		cols = append(cols, col)
	}

	return cols, nil
}

func (t *DependencyTree) QueryForAll(table *TableNode) string {
	var colBuf, tableBuf bytes.Buffer
	colBuf.WriteString("SELECT ")
	t.writeColumns(&colBuf, t.Prepared.Cols[table.Name], table.Name, true)
	tableBuf.WriteString(" FROM ")
	tableBuf.WriteString(table.Name)

	t.writeJoinedTables(&colBuf, &tableBuf, table)

	return colBuf.String() + tableBuf.String()
}

func (t *DependencyTree) writeColumns(buf *bytes.Buffer, cols []string, table string, firstStuff bool) {
	sep := ""
	if !firstStuff {
		sep = ", "
	}
	for _, col := range cols {
		buf.WriteString(sep)
		buf.WriteString(table)
		buf.WriteRune('.')
		buf.WriteString(col)
		buf.WriteString(" AS \"")
		buf.WriteString(table)
		buf.WriteRune('.')
		buf.WriteString(col)
		buf.WriteRune('"')
		sep = ", "
	}
}

func (t *DependencyTree) writeJoinedTables(colBuf, tableBuf *bytes.Buffer, table *TableNode) {

	writeTable := func(newTable string) {
		tableBuf.WriteString(" LEFT JOIN ")
		tableBuf.WriteString(newTable)
		tableBuf.WriteString(" ON")

		fk := t.Prepared.FKs[table.Name][newTable]
		sep := " "
		for i := 0; i < len(fk.Columns); i++ {
			tableBuf.WriteString(sep)
			tableBuf.WriteString(table.Name)
			tableBuf.WriteRune('.')
			tableBuf.WriteString(fk.Columns[i])
			tableBuf.WriteString(" = ")
			tableBuf.WriteString(newTable)
			tableBuf.WriteRune('.')
			tableBuf.WriteString(fk.ForeignColumns[i])

			sep = " AND "
		}
	}

	for _, embedded := range table.Embedded {
		t.writeColumns(colBuf, t.Prepared.Cols[embedded.Name], embedded.Name, false)
		writeTable(embedded.Name)
		// colBuf.WriteString(", ")
		t.writeJoinedTables(colBuf, tableBuf, embedded)
	}

	for _, referenced := range table.Referenced {
		t.writeColumns(colBuf, t.Prepared.PKs[referenced], referenced, false)
		writeTable(referenced)
	}
}

func QueryNxN(cols []string, nxn string) string {
	var buf bytes.Buffer
	buf.WriteString("SELECT * FROM ")
	buf.WriteString(nxn)
	buf.WriteString(" WHERE")

	sep := " "
	param := 1
	for _, col := range cols {
		buf.WriteString(sep)
		buf.WriteString(col)
		buf.WriteString(" = (:")
		buf.WriteString(strconv.Itoa(param))
		buf.WriteString(")")

		sep = " AND "
		param++
	}

	return buf.String()
}
