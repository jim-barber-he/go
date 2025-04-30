package texttable

import (
	"bytes"
	"testing"
)

type Row struct {
	Name  string `title:"NAME"`
	Value string `title:"VALUE"`
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

		var buf bytes.Buffer
		tbl.Write(&buf)
		table := buf.String()

		expected := "NAME  VALUE\na     a\nb     b\nc     c\n"
		if table != expected {
			t.Errorf("Append() failed, expected %s, got %s", expected, table)
		}
	})
}

func TestWrite(t *testing.T) {
	t.Parallel()

	t.Run("Write", func(t *testing.T) {
		t.Parallel()

		tbl := Table[*Row]{
			Rows: []*Row{
				{Name: "a", Value: "a"},
				{Name: "b", Value: "b"},
			},
		}

		var buf bytes.Buffer
		tbl.Write(&buf)
		table := buf.String()

		expected := "NAME  VALUE\na     a\nb     b\n"
		if table != expected {
			t.Errorf("Append() failed, expected %s, got %s", expected, table)
		}
	})
}
