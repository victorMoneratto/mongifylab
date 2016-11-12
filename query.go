package mongifylab

import (
	"database/sql"
	"log"
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

// RowsChan returns an unbuffered channel that sends each row from as a []interface{}
func RowsChan(rows *sql.Rows) (<-chan []interface{}, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// Allocate an interface{} per column
	interfaces := make([]interface{}, len(cols))
	for i := range interfaces {
		var ii interface{}
		interfaces[i] = &ii
	}

	rowChan := make(chan []interface{}, 1)

	go func() {
		for rows.Next() {
			// Try scanning value
			err := rows.Scan(interfaces...)
			if err != nil {
				// TODO pipe these into a separate channel for errors and remove logging
				log.Println("scanning:", err)
				continue
			}

			// copy scanned value to results slice
			results := make([]interface{}, len(cols))
			for i := range interfaces {
				results[i] = *(interfaces[i]).(*interface{})
			}

			rowChan <- results
		}
		close(rowChan)
	}()

	return rowChan, nil
}
