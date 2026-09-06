package texttable_test

import (
	"bytes"
	"testing"

	"github.com/jim-barber-he/go/texttable"
)

type Row struct {
	Name  string `title:"NAME"`
	Value string `title:"VALUE"`
}

func TestAppend(t *testing.T) {
	t.Parallel()

	t.Run("Append", func(t *testing.T) {
		t.Parallel()

		tbl := texttable.Table[*Row]{
			Rows: []*Row{
				{Name: "a", Value: "x"},
				{Name: "b", Value: "y"},
			},
		}
		tbl.Append(&Row{Name: "c", Value: "z"})

		expectedLen := 3
		if len(tbl.Rows) != expectedLen {
			t.Errorf("Append() failed, expected %d, got %d", expectedLen, len(tbl.Rows))
		}

		var buf bytes.Buffer
		tbl.Write(&buf)
		table := buf.String()

		expected := "NAME  VALUE\na     x\nb     y\nc     z\n"
		if table != expected {
			t.Errorf("Append() failed, expected %s, got %s", expected, table)
		}
	})
}

func TestWrite(t *testing.T) {
	t.Parallel()

	t.Run("Write", func(t *testing.T) {
		t.Parallel()

		tbl := texttable.Table[*Row]{
			Rows: []*Row{
				{Name: "a", Value: "x"},
				{Name: "b", Value: "y"},
			},
		}

		var buf bytes.Buffer
		tbl.Write(&buf)
		table := buf.String()

		expected := "NAME  VALUE\na     x\nb     y\n"
		if table != expected {
			t.Errorf("Append() failed, expected %s, got %s", expected, table)
		}
	})
}

func TestWriteNoHeaders(t *testing.T) {
	t.Parallel()

	t.Run("WriteNoHeader", func(t *testing.T) {
		t.Parallel()

		tbl := texttable.Table[*Row]{
			Rows: []*Row{
				{Name: "a", Value: "x"},
				{Name: "b", Value: "y"},
			},
		}

		var buf bytes.Buffer
		tbl.WriteNoHeaders(&buf)
		table := buf.String()

		expected := "a  x\nb  y\n"
		if table != expected {
			t.Errorf("Append() failed, expected %s, got %s", expected, table)
		}
	})
}
