package util

import (
	"testing"
	"time"
)

func TestFormatAge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		timestamp time.Time
		expected  string
	}{
		{
			timestamp: time.Now().Add(-time.Duration(0)),
			expected:  "0s",
		},
		{
			timestamp: time.Now().Add(-time.Hour * 24 * 7 * 3),
			expected:  "3w",
		},
		{
			timestamp: time.Now().Add(-time.Hour * 24 * 7 * 3).Add(-time.Second),
			expected:  "3w1s",
		},
		{
			timestamp: time.Now().Add(-time.Hour * 24 * 7 * 3).Add(-time.Minute),
			expected:  "3w1m",
		},
		{
			timestamp: time.Now().Add(-time.Hour * 24 * 7 * 3).Add(-time.Hour).Add(-time.Minute),
			expected:  "3w1h",
		},
	}

	for _, tt := range tests {
		t.Run("FormatAge", func(t *testing.T) {
			t.Parallel()

			if FormatAge(tt.timestamp) != tt.expected {
				t.Errorf("FormatAge() failed, expected %s, got %s", tt.expected, FormatAge(tt.timestamp))
			}
		})
	}
}

func TestLastSplitItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		sep      string
		expected string
	}{
		{
			input:    "foo/bar/baz",
			sep:      "/",
			expected: "baz",
		},
		{
			input:    "foo/bar/baz",
			sep:      "b",
			expected: "az",
		},
		{
			input:    "foo/bar/baz",
			sep:      "z",
			expected: "",
		},
		{
			input:    "foo/bar/baz",
			sep:      "",
			expected: "z",
		},
	}

	for _, tt := range tests {
		t.Run("LastSplitItem", func(t *testing.T) {
			t.Parallel()

			if LastSplitItem(tt.input, tt.sep) != tt.expected {
				t.Errorf("LastSplitItem() failed, expected %s, got %s", tt.expected, LastSplitItem(tt.input, tt.sep))
			}
		})
	}
}

func TestLineVisualWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		str      string
		linePos  int
		expected int
	}{
		{
			str:      "\t",
			linePos:  8,
			expected: 16,
		},
		{
			str:      "foo",
			linePos:  0,
			expected: 3,
		},
		{
			str:      "foo\t",
			linePos:  0,
			expected: 8,
		},
		{
			str:      "foo\tbar",
			linePos:  0,
			expected: 11,
		},
		{
			str:      "foo\tbar\t",
			linePos:  0,
			expected: 16,
		},
		{
			str:      "foo\tbar\tbaz",
			linePos:  0,
			expected: 19,
		},
	}

	for _, tt := range tests {
		t.Run("LineVisualWidth", func(t *testing.T) {
			t.Parallel()

			if lineVisualWidth(tt.linePos, tt.str) != tt.expected {
				t.Errorf("LineVisualWidth() failed, expected %d, got %d", tt.expected, lineVisualWidth(tt.linePos, tt.str))
			}
		})
	}
}

func TestWrapLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		str      string
		width    int
		expected string
	}{
		{
			str:      "",
			width:    80,
			expected: "",
		},
		{
			str:      "foo bar baz",
			width:    80,
			expected: "foo bar baz",
		},
		{
			str:      "foo bar baz",
			width:    3,
			expected: "foo\nbar\nbaz",
		},
		{
			str:      "foo bar baz",
			width:    5,
			expected: "foo\nbar\nbaz",
		},
		{
			str:      "foo bar baz",
			width:    6,
			expected: "foo\nbar\nbaz",
		},
		{
			str:      "foo bar baz",
			width:    7,
			expected: "foo bar\nbaz",
		},
		{
			str:      "foo bar baz",
			width:    8,
			expected: "foo bar\nbaz",
		},
		{
			str:      "foo bar baz",
			width:    9,
			expected: "foo bar\nbaz",
		},
		{
			str:      "foo bar baz",
			width:    10,
			expected: "foo bar\nbaz",
		},
		{
			str:      "foo bar baz",
			width:    11,
			expected: "foo bar baz",
		},
		{
			str:      "foo bar baz",
			width:    12,
			expected: "foo bar baz",
		},
	}

	for _, tt := range tests {
		t.Run("WrapLine", func(t *testing.T) {
			t.Parallel()

			if WrapLine(tt.str, tt.width) != tt.expected {
				t.Errorf("WrapLine() failed, expected %s, got %s", tt.expected, WrapLine(tt.str, tt.width))
			}
		})
	}
}

func TestWrapTextToWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		str      string
		width    int
		expected string
	}{
		{
			str:      "",
			width:    80,
			expected: "\n",
		},

		{
			str:      "foo bar baz",
			width:    80,
			expected: "foo bar baz\n",
		},

		{
			str:      "foo bar baz",
			width:    3,
			expected: "foo\nbar\nbaz\n",
		},
		{
			str:      "foo bar baz",
			width:    5,
			expected: "foo\nbar\nbaz\n",
		},
		{
			str:      "foo bar baz",
			width:    6,
			expected: "foo\nbar\nbaz\n",
		},
		{
			str:      "foo bar baz",
			width:    7,
			expected: "foo bar\nbaz\n",
		},
		{
			str:      "foo bar baz",
			width:    8,
			expected: "foo bar\nbaz\n",
		},
		{
			str:      "foo bar baz",
			width:    9,
			expected: "foo bar\nbaz\n",
		},
		{
			str:      "foo bar baz",
			width:    10,
			expected: "foo bar\nbaz\n",
		},
		{
			str:      "foo bar baz",
			width:    11,
			expected: "foo bar baz\n",
		},
		{
			str:      "foo bar baz",
			width:    12,
			expected: "foo bar baz\n",
		},
		// Preserve leading tab.
		{
			str:      "\ttesting 1 2 3",
			width:    80,
			expected: "\ttesting 1 2 3\n",
		},
		// Lines will be joined to a single line.
		{
			str:      "testing\n1 2 3",
			width:    80,
			expected: "testing 1 2 3\n",
		},
		// Two paragraphs, joine to a single line each with a blank line separating the two.
		{
			str:      "testing\n123\n\n2nd\nparagraph",
			width:    80,
			expected: "testing 123\n\n2nd paragraph\n",
		},
	}

	for _, tt := range tests {
		t.Run("WrapTextToWidth", func(t *testing.T) {
			t.Parallel()

			if WrapTextToWidth(tt.width, tt.str) != tt.expected {
				t.Errorf("WrapTextToWidth() failed, expected [%s], got [%s]", tt.expected, WrapTextToWidth(tt.width, tt.str))
			}
		})
	}
}
