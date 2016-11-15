package mongifylab

import (
	"database/sql"
	"sort"
)

type DependencyTree struct {
	Tables TableNodes

	Prepared struct {
		Tables []string
		Cols   map[string][]string          // Cols[TableName] = [Cols...]
		FKs    map[string]map[string]FKInfo // FKs[TableName][ForeignTable] = [ForeignKeys...]
	}
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
	t := &DependencyTree{}

	tables, err := ListTables(db)
	if err != nil {
		return nil
	}
	t.Prepared.Tables = tables
	t.Prepared.FKs = make(map[string]map[string]FKInfo)
	t.Prepared.Cols = make(map[string][]string)
	for _, table := range tables {
		//FKs
		_, fks, _, err := QueryConstraints(db, table)
		if err == nil {
			t.Prepared.FKs[table] = fks
		}

		//Cols
		cols, err := QueryColumnNames(db, table)
		if err == nil {
			t.Prepared.Cols[table] = cols
		}
	}

	return t
}

func (t *DependencyTree) Clear() {
	t.Tables = nil
}

func (t *DependencyTree) Add(newTable string, mode TransformMode) {
	// Other must know of newTable
	added := false
	if mode == EmbeddedTransform || mode == ReferencedTransform {
		for _, table := range t.Tables {
			added = t.recursiveAdd(table, newTable, mode)
		}
	}

	// newTable must be added as a top entity
	if mode == SimpleTransform || mode == ReferencedTransform || !added {
		t.Tables = append(t.Tables, NewTableNode(newTable))
		sort.Sort(t.Tables)
	}
}

func (t *DependencyTree) recursiveAdd(table *TableNode, foreignTable string, mode TransformMode) bool {
	_, found := t.Prepared.FKs[table.Name][foreignTable]
	if found {
		switch mode {
		case ReferencedTransform:
			table.Referenced = append(table.Referenced, foreignTable)
		case EmbeddedTransform:
			table.Embedded = append(table.Embedded, NewTableNode(foreignTable))
		}
	}

	for _, embedded := range table.Embedded {
		ret := t.recursiveAdd(embedded, foreignTable, mode)
		found = found || ret
	}

	return found
}

// TableNodes implements sort.Interface
type TableNodes []*TableNode

func (s TableNodes) Len() int {
	return len(s)
}
func (s TableNodes) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}
func (s TableNodes) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// TransformMode is and indication of how to relate a new table
// to preexisting ones
type TransformMode int

const (
	_ = iota
	// SimpleTransform just adds it as a top entity
	SimpleTransform TransformMode = iota

	// EmbeddedTransform inlines it into other tables
	EmbeddedTransform

	// ReferencedTransform adds it as a top entity
	// and makes references from other tables
	ReferencedTransform

	// NxNTransform adds it as a proxy between two other tables
	NxNTransform
)
