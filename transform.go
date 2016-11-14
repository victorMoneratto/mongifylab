package mongifylab

import (
	"bytes"
	"database/sql"
	"sort"
)

type DependencyTree struct {
	DB    *sql.DB
	FKs   map[string][]FKInfo // FKs[TableName] = []ForeignKeys
	Roots TableNodeSlice
}

type TableNode struct {
	Name       string
	Embedded   []*TableNode
	Referenced []string
	NxNProxy   map[string]FKInfo // NxNProxy[RelationshipTable] = ForeignTable
}

func NewTableNode(name string) *TableNode {
	return &TableNode{Name: name}
}

func NewDependencyTree(db *sql.DB) *DependencyTree {
	tables, err := ListTables(db)
	if err != nil {
		return nil
	}

	t := &DependencyTree{DB: db}
	t.FKs = make(map[string][]FKInfo)
	for _, table := range tables {
		_, fks, _, err := QueryConstraints(db, table)
		if err != nil {
			continue
		}
		t.FKs[table] = fks
	}
	return t
}

func (t *DependencyTree) Add(newTable string, mode TransformMode) {
	switch mode {
	case Embedded:
		fallthrough
	case Referenced:
		for _, table := range t.Roots {
			t.findAndAdd(table, newTable, mode)
		}
	}

	if mode != Embedded {
		t.Roots = append(t.Roots, NewTableNode(newTable))
		sort.Sort(t.Roots)
	}
}

func (t *DependencyTree) findAndAdd(table *TableNode, foreignTable string, mode TransformMode) {
	for _, constraint := range t.FKs[table.Name] {
		if constraint.ForeignTable == foreignTable {
			switch mode {
			case Referenced:
				table.Referenced = append(table.Referenced, foreignTable)
			case Embedded:
				table.Embedded = append(table.Embedded, NewTableNode(foreignTable))
			}
		}
	}
	for _, embedded := range table.Embedded {
		t.findAndAdd(embedded, foreignTable, mode)
	}
}

func (t *DependencyTree) MakeCollectionScript() (string, error) {
	var buf bytes.Buffer
	var lastErr error
	for _, table := range t.Roots {
		script, err := CreateCollectionScript(t.DB, table)
		if err != nil {
			lastErr = err
			continue
		}
		buf.WriteString(script)
		buf.WriteString("\n")
	}

	return buf.String(), lastErr
}

type TableNodeSlice []*TableNode

func (s TableNodeSlice) Len() int {
	return len(s)
}

func (s TableNodeSlice) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

func (s TableNodeSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type TransformMode int

const (
	Simple TransformMode = iota
	Embedded
	Referenced
	NxN
)
