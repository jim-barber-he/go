/*
Package texttable provides functions for handling outputting a text based table.
*/
package texttable

import (
	"fmt"
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

// TableFormatter interface that a table row struct needs to implement for the table.Write() method to use it.
// Both of these methods need to return a string containing tab separated row values for the tabwriter module to use.
type TableFormatter interface {
	TabTitleRow() string
	TabValues() string
}

// Table is a generic struct for representing a table with a slice of rows.
type Table[R TableFormatter] struct {
	Rows []R
}

// Append adds a new row to existing rows in a table.
func (t *Table[R]) Append(r R) {
	t.Rows = append(t.Rows, r)
}

// Write displays the table to stdout.
func (t *Table[R]) Write() {
	tw := tabwriter.NewWriter(os.Stdout, tableMinWidth, tableTabWidth, tablePadding, tablePadChar, tableFlags)
	fmt.Fprintln(tw, t.Rows[0].TabTitleRow())
	for _, row := range t.Rows {
		fmt.Fprintln(tw, row.TabValues())
	}
	tw.Flush()
}

// ReflectedTabValues outputs the field values of a struct separated by tabs. Empty fields are ignored.
func ReflectedTabValues[R any](row *R) string {
	var s []string
	v := reflect.ValueOf(*row)
	for i := range v.NumField() {
		if str := strings.TrimSpace(v.Field(i).String()); str != "" {
			s = append(s, str)
		}
	}
	return strings.Join(s, "\t")
}

// ReflectedTitleRow returns a new struct based on the passed in struct with the field values populated via the struct
// tag called 'title'.
// If the field value of the passed in struct is unset and the title tag is set to 'omitempty' then do not include it.
func ReflectedTitleRow[R any](row *R) string {
	var result R
	resultElem := reflect.ValueOf(&result).Elem()

	v := reflect.ValueOf(*row)
	for i, sf := range reflect.VisibleFields(v.Type()) {
		titleArray := strings.Split(sf.Tag.Get("title"), ",")
		if len(titleArray) > 1 && titleArray[1] == "omitempty" && v.Field(i).String() == "" {
			continue
		}
		resultElem.Field(i).SetString(titleArray[0])
	}
	return ReflectedTabValues(&result)
}
