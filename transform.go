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

	NxN map[string]*TableNode

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
	NxNProxy   []*TableNode
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
	t.NxN = make(map[string]*TableNode)

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

	for _, added := range t.AddedTables {
		t.recursiveAdd(newTable, added.Table, added.Mode, false)
	}
	t.AddedTables = append(t.AddedTables, AddedTable{Table: newTable, Mode: mode})

	// Other must know of newTable
	if mode != SimpleTransform {
		for _, table := range t.Root {
			t.recursiveAdd(table, newTable, mode, true)
		}
	}

	// newTable must be added as a top entity
	if mode == SimpleTransform || mode == ReferencedTransform {
		t.Root = append(t.Root, newTable)
		sort.Sort(t.Root)
	}
}

func (t *DependencyTree) recursiveAdd(table *TableNode, foreignNode *TableNode, mode TransformMode, recurse bool) bool {
	// fmt.Println("Table:", table.Name, "New Table:", foreignNode.Name, "Mode:", mode)
	_, found := t.Prepared.FKs[table.Name][foreignNode.Name]
	if found {
		switch mode {
		case ReferencedTransform:
			table.Referenced = append(table.Referenced, foreignNode.Name)
		case EmbeddedTransform:
			table.Embedded = append(table.Embedded, foreignNode)
		}
	}
	if mode == NxNTransform {
		_, hasRel := t.Prepared.FKs[foreignNode.Name][table.Name]
		if _, newNxN := t.NxN[foreignNode.Name]; hasRel && !newNxN {
			found = true
			for i, node := range foreignNode.Embedded {
				if node == table {
					foreignNode.Embedded = append(foreignNode.Embedded[:i], foreignNode.Embedded[i+1:]...)
					break
				}
			}

			for i, node := range foreignNode.Referenced {
				if node == table.Name {
					foreignNode.Referenced = append(foreignNode.Referenced[:i], foreignNode.Referenced[i+1:]...)
					break
				}
			}

			for i, node := range foreignNode.NxNProxy {
				if node == table {
					foreignNode.NxNProxy = append(foreignNode.NxNProxy[:i], foreignNode.NxNProxy[i+1:]...)
					break
				}
			}

			t.NxN[foreignNode.Name] = foreignNode
			table.NxNProxy = append(table.NxNProxy, foreignNode)
		}
	}

	// if recurse {
	for _, embedded := range table.Embedded {
		ret := t.recursiveAdd(embedded, foreignNode, mode, true)
		found = found || ret
	}
	// }

	for _, nxn := range table.NxNProxy {
		ret := t.recursiveAdd(nxn, foreignNode, mode, true)
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
