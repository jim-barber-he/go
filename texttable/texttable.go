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
	val := reflect.ValueOf(t.Rows[0]).Elem()
	fields := reflect.VisibleFields(val.Type())
	numFields := len(fields)

	// Pre-allocate slices with known capacity for better performance
	omit := make([]bool, numFields)

	// Pre-analyze tags to avoid repeated string splits
	titleTags := make([][]string, numFields)
	for i, sf := range fields {
		titleTags[i] = strings.Split(sf.Tag.Get("title"), ",")
		if len(titleTags[i]) > 1 && titleTags[i][1] == "omitempty" {
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

	// Count visible columns and pre-allocate string slice
	visibleCols := 0
	for i := range numFields {
		if !omit[i] {
			visibleCols++
		}
	}
	
	// Pre-allocate string slice with known capacity to reduce allocations
	s := make([]string, 0, visibleCols)

	// Add the title row of the table skipping any `omitempty` columns where all its values are empty.
	for i := range numFields {
		if omit[i] {
			continue
		}
		s = append(s, titleTags[i][0])
	}

	fmt.Fprintln(tw, strings.Join(s, "\t"))

	// Add the table rows skipping any `omitempty` columns where all its values are empty.
	for _, row := range t.Rows {
		s = s[:0] // Reset slice but keep capacity
		val = reflect.ValueOf(row).Elem()

		for i := range numFields {
			if omit[i] {
				continue
			}

			s = append(s, val.Field(i).String())
		}

		fmt.Fprintln(tw, strings.Join(s, "\t"))
	}

	// Display the table.
	if err := tw.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
