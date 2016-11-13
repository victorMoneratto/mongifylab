package mongifylab_test

import (
	"database/sql"
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

func TestRowSliceChan(t *testing.T) {
	rows, err := db.Query("SELECT SIGLA, NOME FROM LE01ESTADO WHERE ROWNUM <= 2")
	if err != nil {
		t.Fail()
	}

	rowCh, err := mongifylab.RowSliceChan(rows)
	if err != nil {
		t.Fail()
	}

	for i := 0; i < 2; i++ {
		select {
		case row := <-rowCh:
			if row == nil || row[0] == nil || row[1] == nil {
				t.Error()
			}
		case <-time.After(time.Millisecond * 100):
			t.Error()
		}
	}
}

func TestRowMapChan(t *testing.T) {
	rows, err := db.Query("SELECT NOME, POPULACAO FROM LE02 WHERE ROWNUM <= 2")
	if err != nil {
		t.Fail()
	}

	rowCh, err := mongifylab.RowMapChan(rows)
	if err != nil {
		t.Fail()
	}

	for i := 0; i < 2; i++ {
		select {
		case row := <-rowCh:
			if row == nil || row["NOME"] == nil || row["POPULACAO"] == nil {
				t.Error()
			}
		case <-time.After(time.Millisecond * 100):
			t.Error()
		}
	}
}
