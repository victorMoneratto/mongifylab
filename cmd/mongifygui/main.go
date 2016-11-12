package main

import (
	"database/sql"
	"fmt"

	"log"

	"os"

	"github.com/victorMoneratto/mongifylab"
	_ "gopkg.in/rana/ora.v3"
)

func main() {
	connString := os.Getenv("ORA_CONN_STRING")
	db, err := sql.Open("ora", connString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tables, err := mongifylab.ListTables(db)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(tables)
}
