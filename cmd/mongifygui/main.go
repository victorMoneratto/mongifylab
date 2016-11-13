package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"

	"log"

	"os"

	"github.com/google/gxui"
	"github.com/google/gxui/drivers/gl"
	"github.com/google/gxui/math"
	"github.com/google/gxui/themes/dark"
	"github.com/victorMoneratto/mongifylab"
	_ "gopkg.in/rana/ora.v3"
)

func main() {
	gl.StartDriver(application)
}

func application(driver gxui.Driver) {
	connString := os.Getenv("ORA_CONN_STRING")
	db, err := sql.Open("ora", connString)
	if err != nil {
		log.Fatal(err)
	}

	tables, err := mongifylab.ListTables(db)
	if err != nil {
		log.Fatal(err)
	}

	theme := dark.CreateTheme(driver)
	layout := theme.CreateLinearLayout()
	layout.SetDirection(gxui.LeftToRight)
	layout.AddChildAt(0, newTree(theme))
	layout.AddChildAt(1, newPanel(theme, db, tables))

	window := theme.CreateWindow(854, 480, "Mongify")
	window.AddChild(layout)
	window.SetPadding(math.CreateSpacing(10))
	window.OnClose(func() {
		db.Close()
		driver.Terminate()
	})
}

func newTree(theme gxui.Theme) gxui.Control {
	tree := theme.CreateTree()
	return tree
}

func newPanel(theme gxui.Theme, db *sql.DB, tables []string) gxui.Control {
	layout := theme.CreateTableLayout()
	layout.SetGrid(8, len(tables))

	for i := range tables {
		label := theme.CreateLabel()
		label.SetText(tables[i])
		layout.SetChildAt(0, i, 4, 1, label)

		add := newAddButton(theme, db, tables[i])
		layout.SetChildAt(4, i, 1, 1, add)
	}

	panel := theme.CreatePanelHolder()
	panel.AddPanel(layout, "Tables")

	return panel
}

func newAddButton(theme gxui.Theme, db *sql.DB, table string) gxui.Control {
	add := theme.CreateButton()
	add.SetType(gxui.ToggleButton)
	add.SetHorizontalAlignment(gxui.AlignCenter)
	add.SetText("Add")
	add.OnClick(func(e gxui.MouseEvent) {
		if add.IsChecked() {
			script, err := mongifylab.CreateCollectionScript(db, table)
			if err != nil {
				fmt.Println(err)
			}
			ioutil.WriteFile(table+".js", []byte(script+"\n"), 0644)
			fmt.Println(script)
		} else {
			fmt.Println("// remove", table)
		}
	})

	return add
}
