package main

import (
	"github.com/google/gxui"
	"github.com/google/gxui/math"
	"github.com/victorMoneratto/mongifylab"
)

type ColInput struct {
	OpList gxui.DropDownList
	Text   gxui.TextBox
}

var inputs map[string]ColInput
var colsLayout gxui.TableLayout

func newQueryPanel(window gxui.Window, theme gxui.Theme, driver gxui.Driver) gxui.Control {
	layout := theme.CreateLinearLayout()
	layout.SetDirection(gxui.BottomToTop)

	code := theme.CreateCodeEditor()

	tableOverlay := theme.CreateBubbleOverlay()
	overlays = append(overlays, tableOverlay)

	tableLayout := theme.CreateLinearLayout()
	tableLayout.SetDirection(gxui.LeftToRight)

	tableLabel := theme.CreateLabel()
	tableLabel.SetText("Table:")
	tableLabel.SetFont(labelFont)

	tableList := theme.CreateDropDownList()
	tableList.SetAdapter(NewListAdapter())
	tableList.SetBubbleOverlay(tableOverlay)
	tableList.OnSelectionChanged(func(i gxui.AdapterItem) {
		table := i.(string)
		cols := dependencies.Prepared.Cols[table]
		inputs = make(map[string]ColInput)
		if colsLayout != nil {
			colsLayout.RemoveChild(code)
			layout.RemoveChild(colsLayout)
		}

		colsLayout = theme.CreateTableLayout()
		colsLayout.SetGrid(7, len(cols)+1)
		for i, col := range cols {
			colLabel := theme.CreateLabel()
			colLabel.SetText(col + ":")

			opList := NewOperatorList(theme)
			opOverlay := theme.CreateBubbleOverlay()
			opList.SetBubbleOverlay(opOverlay)
			window.AddChild(opOverlay)
			opList.Select("=")

			text := theme.CreateTextBox()
			text.SetSize(math.Size{W: math.MaxSize.W, H: 20})

			inputs[col] = ColInput{OpList: opList, Text: text}

			colsLayout.SetChildAt(0, i, 1, 1, colLabel)
			colsLayout.SetChildAt(1, i, 1, 1, opList)
			colsLayout.SetChildAt(2, i, 5, 1, text)
		}

		colsLayout.SetChildAt(0, len(cols), 7, 1, code)
		layout.AddChild(colsLayout)
	})

	submit := theme.CreateButton()
	submit.SetText("Go!")
	submit.OnClick(func(gxui.MouseEvent) {
		conditions := make(map[string]mongifylab.QueryInput)
		for col, input := range inputs {
			if operator, text := input.OpList.Selected().(string), input.Text.Text(); operator != "" && text != "" {
				conditions[col] = mongifylab.QueryInput{Operator: operator, Text: text}
			}
		}

		if table := tableList.Selected().(string); table != "" {
			script := dependencies.CreateFindScript(table, conditions)
			code.SetText(script)
		}
	})

	tableLayout.AddChild(tableLabel)
	tableLayout.AddChild(tableList)
	tableLayout.AddChild(submit)

	layout.AddChild(tableLayout)

	return layout
}

func NewOperatorList(theme gxui.Theme) gxui.DropDownList {
	adapter := gxui.CreateDefaultAdapter()
	adapter.SetSize(math.Size{W: 100, H: 20})
	adapter.SetItems([]string{"=", ">", ">=", "<", "<=", "!=", "in", "nin", "exists", "type"})
	adapter.DataReplaced()

	list := theme.CreateDropDownList()
	list.SetAdapter(adapter)
	return list
}
