/*
Package texttable provides functions for handling outputting a text based table.
*/
package texttable

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"
)

const (
	tableFlags    = 0
	tableMinWidth = 0
	tablePadChar  = ' '
	tablePadding  = 2
	tableTabWidth = 8
)

// Table is a generic struct for representing a table with a slice of rows.
type Table[R any] struct {
	Rows []R
}

// Append adds a new row to existing rows in a table.
func (t *Table[R]) Append(r R) {
	t.Rows = append(t.Rows, r)
}

// Write the table to an io.Writer. If no io.Writer is provided, it defaults to os.Stdout.
func (t *Table[R]) Write(w ...io.Writer) {
	if len(t.Rows) == 0 {
		return
	}

	// Using the first row of the table, use reflection to determine what fields are in the table row.
	v := reflect.ValueOf(t.Rows[0]).Elem()
	fields := reflect.VisibleFields(v.Type())
	numFields := len(fields)

	// Create a slice to determine which fields to omit from the output.
	// Initially any struct field with the `omitempty` tag is set to true to be omitted.
	omit := make([]bool, numFields)

	for i, sf := range fields {
		titleArray := strings.Split(sf.Tag.Get("title"), ",")
		if len(titleArray) > 1 && titleArray[1] == "omitempty" {
			omit[i] = true
		}
	}
	// For any field set to be omitted, don't omit it if it has a value set in any of its rows.
	for i := range omit {
		if omit[i] {
			for _, row := range t.Rows {
				if reflect.ValueOf(row).Elem().Field(i).String() != "" {
					omit[i] = false
					break
				}
			}
		}
	}

	// Create a tab writer to display the table. Each row needs to consist of tab separated strings.
	var tw *tabwriter.Writer
	if len(w) > 0 {
		tw = tabwriter.NewWriter(w[0], tableMinWidth, tableTabWidth, tablePadding, tablePadChar, tableFlags)
	} else {
		tw = tabwriter.NewWriter(os.Stdout, tableMinWidth, tableTabWidth, tablePadding, tablePadChar, tableFlags)
	}

	// Add the title row of the table skipping any `omitempty` columns where all its values are empty.
	var s []string

	for i, sf := range fields {
		if omit[i] {
			continue
		}

		s = append(s, strings.Split(sf.Tag.Get("title"), ",")[0])
	}

	fmt.Fprintln(tw, strings.Join(s, "\t"))

	// Add the table rows skipping any `omitempty` columns where all its values are empty.
	for _, row := range t.Rows {
		s = nil
		v = reflect.ValueOf(row).Elem()

		for i := range numFields {
			if omit[i] {
				continue
			}

			s = append(s, v.Field(i).String())
		}

		fmt.Fprintln(tw, strings.Join(s, "\t"))
	}

	// Display the table.
	if err := tw.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
