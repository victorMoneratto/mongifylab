package mongifylab

import (
	"bytes"
	"database/sql"
)

type DependencyTree struct {
	DB    *sql.DB
	FKs   map[string][]FKInfo
	Roots []TableNode
}

type TableNode struct {
	Name       string
	Embedded   []FKInfo
	Referenced []FKInfo
	NxNProxy   map[string]FKInfo // NxNProxy[ForeignTable] = RelationshipTableInfo
}

func NewTableNode(name string) TableNode {
	return TableNode{Name: name}
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

func (t *DependencyTree) Add(table string, mode TransformMode) {
	switch mode {
	case Simple:
		t.Roots = append(t.Roots, NewTableNode(table))
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

type TransformMode int

const (
	Simple TransformMode = iota
	Embedded
	Referenced
	NxN
)
