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

// TableConfig holds configuration for table formatting.
type TableConfig struct {
	MinWidth int
	TabWidth int
	Padding  int
	PadChar  byte
	Flags    uint
}

// DefaultTableConfig returns a TableConfig with sensible defaults.
func DefaultTableConfig() TableConfig {
	return TableConfig{
		MinWidth: tableMinWidth,
		TabWidth: tableTabWidth,
		Padding:  tablePadding,
		PadChar:  tablePadChar,
		Flags:    tableFlags,
	}
}

// FieldInfo contains metadata about a table field.
type FieldInfo struct {
	Field       reflect.StructField
	Title       string
	OmitIfEmpty bool
	Index       int
}

// Table is a generic struct for representing a table with a slice of rows.
type Table[R any] struct {
	Rows   []R
	Config TableConfig
}

// NewTable creates a new table with default configuration.
func NewTable[R any]() *Table[R] {
	return &Table[R]{
		Config: DefaultTableConfig(),
	}
}

// Append adds a new row to existing rows in a table.
func (t *Table[R]) Append(r R) {
	t.Rows = append(t.Rows, r)
}

// analyzeFields extracts field information from the table row type.
func (t *Table[R]) analyzeFields() []FieldInfo {
	if len(t.Rows) == 0 {
		return nil
	}

	val := reflect.ValueOf(t.Rows[0]).Elem()
	fields := reflect.VisibleFields(val.Type())
	fieldInfos := make([]FieldInfo, 0, len(fields))

	for i, field := range fields {
		titleTag := field.Tag.Get("title")
		titleArray := strings.Split(titleTag, ",")
		
		info := FieldInfo{
			Field:       field,
			Title:       titleArray[0],
			OmitIfEmpty: len(titleArray) > 1 && titleArray[1] == "omitempty",
			Index:       i,
		}
		
		fieldInfos = append(fieldInfos, info)
	}

	return fieldInfos
}

// determineOmissions calculates which fields should be omitted based on their values.
func (t *Table[R]) determineOmissions(fieldInfos []FieldInfo) []bool {
	omit := make([]bool, len(fieldInfos))

	// Mark fields for omission if they have the omitempty tag
	for i, info := range fieldInfos {
		omit[i] = info.OmitIfEmpty
	}

	// Don't omit fields that have values in any row
	for i, shouldOmit := range omit {
		if shouldOmit {
			for _, row := range t.Rows {
				if reflect.ValueOf(row).Elem().Field(i).String() != "" {
					omit[i] = false
					break
				}
			}
		}
	}

	return omit
}

// Write the table to an io.Writer. If no io.Writer is provided, it defaults to os.Stdout.
func (t *Table[R]) Write(w ...io.Writer) {
	if len(t.Rows) == 0 {
		return
	}

	fieldInfos := t.analyzeFields()
	if fieldInfos == nil {
		return
	}

	omit := t.determineOmissions(fieldInfos)

	// Use default config if not set
	config := t.Config
	if config.MinWidth == 0 && config.TabWidth == 0 && config.Padding == 0 && config.PadChar == 0 && config.Flags == 0 {
		config = DefaultTableConfig()
	}

	// Create a tab writer to display the table
	var tw *tabwriter.Writer
	if len(w) > 0 {
		tw = tabwriter.NewWriter(w[0], config.MinWidth, config.TabWidth, config.Padding, config.PadChar, config.Flags)
	} else {
		tw = tabwriter.NewWriter(os.Stdout, config.MinWidth, config.TabWidth, config.Padding, config.PadChar, config.Flags)
	}

	// Add the title row of the table skipping any omitted columns
	var titleRow []string
	for i, info := range fieldInfos {
		if omit[i] {
			continue
		}
		titleRow = append(titleRow, info.Title)
	}
	fmt.Fprintln(tw, strings.Join(titleRow, "\t"))

	// Add the table rows skipping any omitted columns
	for _, row := range t.Rows {
		var rowData []string
		val := reflect.ValueOf(row).Elem()

		for i := range len(fieldInfos) {
			if omit[i] {
				continue
			}
			rowData = append(rowData, val.Field(i).String())
		}

		fmt.Fprintln(tw, strings.Join(rowData, "\t"))
	}

	// Display the table
	if err := tw.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
