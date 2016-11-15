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
var overlay gxui.BubbleOverlay
var labelFont gxui.Font
var tree gxui.Tree
var list gxui.DropDownList
var table gxui.TableLayout

var db *sql.DB
var dependencies *mongifylab.DependencyTree

func application(driver gxui.Driver) {
	// Connect to Oracle
	connString := os.Getenv("ORA_CONN_STRING")
	var err error
	db, err = sql.Open("ora", connString)
	if err != nil {
		log.Fatal(err)
	}

	dependencies = mongifylab.NewDependencyTree(db)

	theme := dark.CreateTheme(driver)
	overlay = theme.CreateBubbleOverlay()
	labelFont, err = driver.CreateFont(gxfont.Default, 18)
	if err != nil {
		log.Println(err)
		labelFont = theme.DefaultFont()
	}

	layout := theme.CreateLinearLayout()
	layout.SetDirection(gxui.LeftToRight)
	layout.AddChild(newTree(theme))
	layout.AddChild(newPanel(theme, driver))

	window := theme.CreateWindow(960, 540, "Mongify")
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

func newPanel(theme gxui.Theme, driver gxui.Driver) gxui.Control {
	//
	// Tables list and buttons
	//
	listLabel := theme.CreateLabel()
	listLabel.SetText("Table:")
	listLabel.SetFont(labelFont)

	list = theme.CreateDropDownList()
	list.SetAdapter(NewListAdapter())
	list.SetBubbleOverlay(overlay)
	list.SetMargin(math.CreateSpacing(5))

	table = theme.CreateTableLayout()
	table.SetGrid(17, 20)
	table.SetChildAt(0, 0, 2, 2, listLabel)
	table.SetChildAt(2, 0, 15, 2, list)

	addSimple := theme.CreateButton()
	addSimple.SetText("Simple")
	addSimple.SetHorizontalAlignment(gxui.AlignCenter)
	addSimple.OnClick(func(gxui.MouseEvent) {
		if selected := list.Selected(); selected != nil {
			addDependency(selected.(string), mongifylab.SimpleTransform)
		}
	})

	addReferenced := theme.CreateButton()
	addReferenced.SetText("Reference")
	addReferenced.SetHorizontalAlignment(gxui.AlignCenter)
	addReferenced.OnClick(func(gxui.MouseEvent) {
		if selected := list.Selected(); selected != nil {
			addDependency(selected.(string), mongifylab.ReferencedTransform)
		}
	})

	addEmbedded := theme.CreateButton()
	addEmbedded.SetText("Embedded")
	addEmbedded.SetHorizontalAlignment(gxui.AlignCenter)
	addEmbedded.OnClick(func(gxui.MouseEvent) {
		if selected := list.Selected(); selected != nil {
			addDependency(selected.(string), mongifylab.EmbeddedTransform)
		}
	})

	addNxN := theme.CreateButton()
	addNxN.SetText("N x N")
	addNxN.SetHorizontalAlignment(gxui.AlignCenter)
	addNxN.OnClick(func(gxui.MouseEvent) {
		if selected := list.Selected(); selected != nil {
			addDependency(selected.(string), mongifylab.NxNTransform)
		}
	})

	// forward declaration
	var code gxui.CodeEditor

	// reset := theme.CreateButton()
	// reset.SetText("Reset")
	// reset.SetHorizontalAlignment(gxui.AlignCenter)
	// reset.OnClick(func(gxui.MouseEvent) {
	// 	dependencies.Clear()
	// 	tree.Adapter().(*TableNodeAdapter).RemakeFromDependencies(dependencies)
	// 	var tables []string
	// 	copy(tables, dependencies.Prepared.Tables)
	// 	list.Adapter().(*gxui.DefaultAdapter).SetItems(tables)
	// 	list.Adapter().(*gxui.DefaultAdapter).DataReplaced()
	// 	code.SetText("")
	// })

	recommended := theme.CreateButton()
	recommended.SetText("Premade")
	recommended.SetHorizontalAlignment(gxui.AlignCenter)

	submit := theme.CreateButton()
	submit.SetText("Go!")
	submit.SetHorizontalAlignment(gxui.AlignCenter)
	// submit.SetBorderPen(gxui.WhitePen)
	submit.OnClick(func(gxui.MouseEvent) {
		script, err := dependencies.CreateCollectionScript(db)
		if err != nil {
			log.Println(err)
			return
		}
		code.SetText(script)
	})

	table.SetChildAt(2, 2, 2, 1, addSimple)
	table.SetChildAt(4, 2, 2, 1, addReferenced)
	table.SetChildAt(6, 2, 2, 1, addEmbedded)
	table.SetChildAt(8, 2, 2, 1, addNxN)
	// table.SetChildAt(11, 2, 2, 1, reset)
	table.SetChildAt(13, 2, 2, 1, recommended)
	table.SetChildAt(15, 2, 2, 1, submit)

	//
	// Code
	//
	code = theme.CreateCodeEditor()
	// code.(*gxui.Control).SetBorderPen(gxui.WhitePen)

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

	table.SetChildAt(2, 3, 15, 17, code)
	// table.SetChildAt(0, 2, 1, 1, codeLabel)
	table.SetChildAt(0, 3, 2, 1, copyClip)
	table.SetChildAt(0, 4, 2, 1, save)

	panel := theme.CreatePanelHolder()
	panel.AddPanel(table, "Tables")

	return panel
}

func NewListAdapter() gxui.ListAdapter {
	adapter := gxui.CreateDefaultAdapter()
	adapter.SetSize(math.Size{W: math.MaxSize.W, H: 20})
	adapter.SetItems(dependencies.Prepared.Tables)
	adapter.DataReplaced()
	return adapter
}
