package main

import (
	"database/sql"

	"log"

	"os"

	"io/ioutil"

	"github.com/google/gxui"
	"github.com/google/gxui/drivers/gl"
	"github.com/google/gxui/gxfont"
	"github.com/google/gxui/math"
	"github.com/google/gxui/themes/dark"
	"github.com/victorMoneratto/mongifylab"
	_ "gopkg.in/rana/ora.v3"
)

func main() {
	gl.StartDriver(application)
}

// Some globals because I'm tired
var labelFont gxui.Font
var tree gxui.Tree
var list gxui.DropDownList

var dependencies *mongifylab.DependencyTree

func application(driver gxui.Driver) {
	// Connect to Oracle
	connString := os.Getenv("ORA_CONN_STRING")
	db, err := sql.Open("ora", connString)
	if err != nil {
		log.Fatal(err)
	}

	// List relevant tables
	tables, err := mongifylab.ListTables(db)
	if err != nil {
		log.Fatal(err)
	}

	dependencies = mongifylab.NewDependencyTree(db)

	theme := dark.CreateTheme(driver)
	overlay := theme.CreateBubbleOverlay()
	labelFont, err = driver.CreateFont(gxfont.Default, 18)
	if err != nil {
		log.Println(err)
		labelFont = theme.DefaultFont()
	}

	layout := theme.CreateLinearLayout()
	layout.SetDirection(gxui.LeftToRight)
	layout.AddChild(newTree(theme))
	layout.AddChild(newPanel(theme, driver, overlay, tables))

	window := theme.CreateWindow(854, 480, "Mongify")
	window.AddChild(layout)
	window.AddChild(overlay)
	window.SetPadding(math.CreateSpacing(10))
	window.OnClose(func() {
		db.Close()
		driver.Terminate()
	})
}

func addDependency(table string, mode mongifylab.TransformMode) {
	dependencies.Add(table, mode)

	// remove from list
	listAdapter := list.Adapter().(*gxui.DefaultAdapter)
	selID := listAdapter.ItemIndex(list.Selected())
	listItems := listAdapter.Items().([]string)
	listItems = append(listItems[:selID], listItems[selID+1:]...)
	listAdapter.SetItems(listItems)
	listAdapter.DataReplaced()
	if len(listItems) > 0 {
		list.Select(0)
	}

	// add to tree
	treeAdapter := tree.Adapter().(*TableNodeAdapter)
	treeAdapter.RemakeFromDependencies(dependencies)
	tree.ExpandAll()
}

func newTree(theme gxui.Theme) gxui.Control {
	tree = theme.CreateTree()
	adapter := NewTableNodeAdapter()
	tree.SetAdapter(adapter)
	tree.Select(adapter)
	tree.Show(tree.Selected())
	return tree
}

func newPanel(theme gxui.Theme, driver gxui.Driver, overlay gxui.BubbleOverlay, tables []string) gxui.Control {
	//
	// Tables list and buttons
	//
	listLabel := theme.CreateLabel()
	listLabel.SetText("Table:")
	listLabel.SetFont(labelFont)

	adapter := gxui.CreateDefaultAdapter()
	adapter.SetSize(math.Size{W: math.MaxSize.W, H: 20})
	adapter.SetItems(tables)

	list = theme.CreateDropDownList()
	list.SetAdapter(adapter)
	list.SetBubbleOverlay(overlay)
	list.SetMargin(math.CreateSpacing(5))

	table := theme.CreateTableLayout()
	table.SetGrid(12, 10)
	table.SetChildAt(0, 0, 1, 1, listLabel)
	table.SetChildAt(1, 0, 11, 1, list)

	addSimple := theme.CreateButton()
	addSimple.SetText("Simple")
	addSimple.SetHorizontalAlignment(gxui.AlignCenter)
	addSimple.OnClick(func(gxui.MouseEvent) {
		if selected := list.Selected(); selected != nil {
			addDependency(selected.(string), mongifylab.Simple)
		}
	})

	addReferenced := theme.CreateButton()
	addReferenced.SetText("Reference")
	addReferenced.SetHorizontalAlignment(gxui.AlignCenter)
	addReferenced.OnClick(func(gxui.MouseEvent) {
		if selected := list.Selected(); selected != nil {
			addDependency(selected.(string), mongifylab.Referenced)
		}
	})

	addEmbedded := theme.CreateButton()
	addEmbedded.SetText("Embedded")
	addEmbedded.SetHorizontalAlignment(gxui.AlignCenter)
	addEmbedded.OnClick(func(gxui.MouseEvent) {
		if selected := list.Selected(); selected != nil {
			addDependency(selected.(string), mongifylab.Embedded)
		}
	})

	addNxN := theme.CreateButton()
	addNxN.SetText("N x N")
	addNxN.SetHorizontalAlignment(gxui.AlignCenter)
	addNxN.OnClick(func(gxui.MouseEvent) {
		if selected := list.Selected(); selected != nil {
			addDependency(selected.(string), mongifylab.NxN)
		}
	})

	// forward declaration
	var code gxui.CodeEditor

	submit := theme.CreateButton()
	submit.SetText("Go!")
	submit.SetHorizontalAlignment(gxui.AlignCenter)
	submit.OnClick(func(gxui.MouseEvent) {
		script, err := dependencies.MakeCollectionScript()
		if err != nil {
			log.Println(err)
			return
		}
		code.SetText(script)
	})

	table.SetChildAt(1, 1, 2, 1, addSimple)
	table.SetChildAt(3, 1, 2, 1, addReferenced)
	table.SetChildAt(5, 1, 2, 1, addEmbedded)
	table.SetChildAt(7, 1, 2, 1, addNxN)
	table.SetChildAt(10, 1, 2, 1, submit)

	//
	// Code
	//
	code = theme.CreateCodeEditor()
	codeLabel := theme.CreateLabel()
	codeLabel.SetFont(labelFont)
	codeLabel.SetText("Output:")

	copyClip := theme.CreateButton()
	copyClip.SetText("Copy")
	copyClip.SetHorizontalAlignment(gxui.AlignCenter)
	copyClip.OnClick(func(e gxui.MouseEvent) {
		driver.SetClipboard(code.Text())
	})

	save := theme.CreateButton()
	save.SetText("Save")
	save.SetHorizontalAlignment(gxui.AlignCenter)
	save.OnClick(func(e gxui.MouseEvent) {
		err := ioutil.WriteFile("file.txt", []byte(code.Text()), 0644)
		if err != nil {
			log.Println(err)
		}
	})

	table.SetChildAt(1, 2, 11, 8, code)
	table.SetChildAt(0, 2, 1, 1, codeLabel)
	table.SetChildAt(0, 3, 1, 1, copyClip)
	table.SetChildAt(0, 4, 1, 1, save)

	panel := theme.CreatePanelHolder()
	panel.AddPanel(table, "Tables")

	return panel
}
