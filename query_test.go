package mongifylab_test

import (
	"database/sql"
	"math/rand"
	"os"
	"testing"

	"time"

	"github.com/victorMoneratto/mongifylab"
	_ "gopkg.in/rana/ora.v3"
)

var db *sql.DB

func TestMain(m *testing.M) {
	var err error
	connString := os.Getenv("ORA_CONN_STRING")
	db, err = sql.Open("ora", connString)
	if err != nil {
		os.Exit(1)
	}
	defer db.Close()

	os.Exit(m.Run())
}

func TestListTables(t *testing.T) {
	tables, err := mongifylab.ListTables(db)
	if err != nil || len(tables) == 0 {
		t.Fail()
	}
}

func TestRowsChan(t *testing.T) {
	tables, err := mongifylab.ListTables(db)
	if err != nil {
		t.Fail()
	}
	table := tables[rand.Intn(len(tables))]

	rows, err := db.Query("SELECT * FROM " + table + " WHERE ROWNUM <= 2")
	if err != nil {
		t.Fail()
	}

	rowCh, err := mongifylab.RowsChan(rows)
	if err != nil {
		t.Fail()
	}

	for i := 0; i < 2; i++ {
		select {
		case rowCh := <-rowCh:
			if rowCh == nil {
				t.Error()
			}
		case <-time.After(time.Millisecond * 100):
			t.Error()
		}
	}
}
