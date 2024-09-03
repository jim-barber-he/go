package texttable

import (
	"fmt"
	"testing"
)

type Row struct {
	Name  string `title:"NAME"`
	Value string `title:"VALUE"`
}

// Implement the texttab.TableFormatter interface.
func (tr *Row) TabTitleRow() string {
	return ReflectedTitleRow(tr)
}

// Implement the texttab.TableFormatter interface.
func (tr *Row) TabValues() string {
	return ReflectedTabValues(tr)
}

func TestReflectedTitleRow(t *testing.T) {
	t.Parallel()

	t.Run("ReflectedTitleRow", func(t *testing.T) {
		t.Parallel()

		row := &Row{}
		expected := "NAME\tVALUE"
		if row.TabTitleRow() != expected {
			t.Errorf("TabTitleRow() failed, expected %s, got %s", expected, row.TabTitleRow())
		}
	})
}

func TestReflectedTabValues(t *testing.T) {
	t.Parallel()

	t.Run("ReflectedTabValues", func(t *testing.T) {
		t.Parallel()

		row := &Row{Name: "a", Value: "b"}
		expected := "a\tb"
		if row.TabValues() != expected {
			t.Errorf("TabValues() failed, expected %s, got %s", expected, row.TabValues())
		}
	})
}

func TestAppend(t *testing.T) {
	t.Parallel()

	t.Run("Append", func(t *testing.T) {
		t.Parallel()

		tbl := Table[*Row]{
			Rows: []*Row{
				{Name: "a", Value: "a"},
				{Name: "b", Value: "b"},
			},
		}
		tbl.Append(&Row{Name: "c", Value: "c"})

		expectedLen := 3
		if len(tbl.Rows) != expectedLen {
			t.Errorf("Append() failed, expected %d, got %d", expectedLen, len(tbl.Rows))
		}

		table := fmt.Sprintln(tbl.Rows[0].TabTitleRow())
		for _, row := range tbl.Rows {
			table += fmt.Sprintln(row.TabValues())
		}
		expected := "NAME\tVALUE\na\ta\nb\tb\nc\tc\n"
		if table != expected {
			t.Errorf("Append() failed, expected %s, got %s", expected, table)
		}
	})
}
