package mongifylab

import (
	"database/sql"
	"sort"
)

type DependencyTree struct {
	Root TableNodes

	// AddedTables is a map of all added tables. It is important because
	// not everytime it is possible to add the table to the tree,
	// but it may be added later, when new tables arrive and relate to it.
	AddedTables []AddedTable

	Prepared struct {
		Tables []string
		Cols   map[string][]string          // Cols[TableName] = [Cols...]
		PKs    map[string][]string          // PKs[TableName] = [PkCols...]
		FKs    map[string]map[string]FKInfo // FKs[TableName][ForeignTable] = [ForeignKeys...]
	}
}

type TableNode struct {
	Name       string
	Embedded   []*TableNode
	Referenced []string
	NxNProxy   map[string]FKInfo // NxNProxy[RelationshipTable] = ForeignTable
}

type AddedTable struct {
	Table *TableNode
	Mode  TransformMode
}

func NewTableNode(name string) *TableNode {
	return &TableNode{Name: name}
}

func NewDependencyTree(db *sql.DB) *DependencyTree {
	t := &DependencyTree{}

	// Prepare database data
	tables, err := ListTables(db)
	if err != nil {
		return nil
	}
	t.Prepared.Tables = tables
	t.Prepared.PKs = make(map[string][]string)
	t.Prepared.FKs = make(map[string]map[string]FKInfo)
	t.Prepared.Cols = make(map[string][]string)
	for _, table := range tables {
		//FKs
		pks, fks, _, err := QueryConstraints(db, table)
		if err == nil {
			t.Prepared.PKs[table] = pks
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
	t.Root = nil
}

func (t *DependencyTree) Add(newTableName string, mode TransformMode) {
	newTable := NewTableNode(newTableName)
	t.AddedTables = append(t.AddedTables, AddedTable{Table: newTable, Mode: mode})

	for _, added := range t.AddedTables {
		t.addPreviousTables(mode, added, newTable)
	}

	// Other must know of newTable
	if mode != SimpleTransform {
		for _, table := range t.Root {
			t.recursiveAdd(table, newTable, mode)
		}
	}

	// newTable must be added as a top entity
	if mode == SimpleTransform || mode == ReferencedTransform {
		t.Root = append(t.Root, newTable)
		sort.Sort(t.Root)
	}
}

func (t *DependencyTree) addPreviousTables(mode TransformMode, oldTable AddedTable, newTable *TableNode) {
	if _, found := t.Prepared.FKs[newTable.Name][oldTable.Table.Name]; found {
		switch oldTable.Mode {
		case EmbeddedTransform:
			newTable.Embedded = append(newTable.Embedded, oldTable.Table)

		case ReferencedTransform:
			newTable.Referenced = append(newTable.Referenced, oldTable.Table.Name)
		}
	}
}

func (t *DependencyTree) recursiveAdd(table *TableNode, foreignNode *TableNode, mode TransformMode) bool {
	_, found := t.Prepared.FKs[table.Name][foreignNode.Name]
	if found {
		switch mode {
		case ReferencedTransform:
			table.Referenced = append(table.Referenced, foreignNode.Name)
		case EmbeddedTransform:
			table.Embedded = append(table.Embedded, foreignNode)
		}
	}

	for _, embedded := range table.Embedded {
		ret := t.recursiveAdd(embedded, foreignNode, mode)
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
