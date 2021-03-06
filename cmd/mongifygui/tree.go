package main

import (
	"github.com/google/gxui"
	"github.com/google/gxui/math"
	"github.com/victorMoneratto/mongifylab"
)

var nextItem int

type TableNode struct {
	Name     string
	Children []*TableNode
	Changed  func()
	item     int
}

func (n *TableNode) Add(name string) *TableNode {
	child := &TableNode{
		Name:    name,
		Changed: n.Changed,
		item:    nextItem,
	}

	nextItem++
	n.Children = append(n.Children, child)
	return child
}

func (n *TableNode) Item() gxui.AdapterItem {
	return n.item
}

func (n *TableNode) Create(theme gxui.Theme) gxui.Control {
	layout := theme.CreateLinearLayout()
	layout.SetDirection(gxui.LeftToRight)

	label := theme.CreateLabel()
	label.SetText(n.Name)

	layout.AddChild(label)

	return layout
}

func (n *TableNode) Count() int {
	return len(n.Children)
}

func (n *TableNode) NodeAt(i int) gxui.TreeNode {
	return n.Children[i]
}

func (n *TableNode) ItemIndex(item gxui.AdapterItem) int {
	for i, child := range n.Children {
		if child.item == item || child.ItemIndex(item) >= 0 {
			return i
		}
	}
	return -1
}

type TableNodeAdapter struct {
	gxui.AdapterBase
	TableNode
}

func (a *TableNodeAdapter) Root() *TableNode {
	return a.Children[0]
}

func (a *TableNodeAdapter) Size(t gxui.Theme) math.Size {
	return math.Size{W: 150, H: 20}
}

func NewTableNodeAdapter() *TableNodeAdapter {
	adapter := &TableNodeAdapter{}
	adapter.Changed = func() { adapter.DataChanged(false) }
	adapter.Add("Tables")
	return adapter
}

func (a *TableNodeAdapter) RemakeFromDependencies(dp *mongifylab.DependencyTree) {
	root := a.Root()
	root.Children = nil
	for _, dpRoot := range dp.Root {
		a.addTable(root, dpRoot, mongifylab.SimpleTransform)
	}
	a.DataChanged(true)
}

func (a *TableNodeAdapter) addTable(parent *TableNode, table *mongifylab.TableNode, mode mongifylab.TransformMode) {
	prefix := ""
	if mode == mongifylab.ReferencedTransform {
		prefix = "-> "
	}
	node := parent.Add(prefix + table.Name)

	for _, embedded := range table.Embedded {
		a.addTable(node, embedded, mongifylab.SimpleTransform)
	}

	for _, ref := range table.Referenced {
		node.Add("-> " + ref)
	}

	for _, nxn := range table.NxNProxy {
		nxnNode := node.Add("(N:N) " + nxn.Name)
		for _, embedded := range nxn.Embedded {
			a.addTable(nxnNode, embedded, mongifylab.SimpleTransform)
		}

		for _, ref := range nxn.Referenced {
			nxnNode.Add("-> " + ref)
		}
	}
}
