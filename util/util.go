/*
Package util provides various utility functions.
*/
package util

import (
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

const (
	numSecondsPerMinute = 60
	numSecondsPerHour   = 60 * numSecondsPerMinute
	numSecondsPerDay    = 24 * numSecondsPerHour
	numSecondsPerWeek   = 7 * numSecondsPerDay

	tabStopWidth = 8
)

// FormatAge returns the age in a human readable format of the first 2 non-zero time units from weeks to seconds,
// or just the seconds if no higher time unit was above 0.
// This differs from duration.String() in that it also handles weeks and days.
func FormatAge(timestamp time.Time) string {
	var weeks, days, hours, minutes, seconds int

	duration := time.Since(timestamp).Round(time.Second)

	seconds = int(duration.Seconds())

	weeks = seconds / numSecondsPerWeek
	seconds -= weeks * numSecondsPerWeek

	days = seconds / numSecondsPerDay
	seconds -= days * numSecondsPerDay

	hours = seconds / numSecondsPerHour
	seconds -= hours * numSecondsPerHour

	minutes = seconds / numSecondsPerMinute
	seconds -= minutes * numSecondsPerMinute

	var dateStr string
	// When set to true, return as soon as the next non-zero time unit is set.
	var retNext bool

	if weeks > 0 {
		dateStr = fmt.Sprintf("%dw", weeks)
		retNext = true
	}
	if days > 0 {
		dateStr = fmt.Sprintf("%s%dd", dateStr, days)
		if retNext {
			return dateStr
		}
		retNext = true
	}
	if hours > 0 {
		dateStr = fmt.Sprintf("%s%dh", dateStr, hours)
		if retNext {
			return dateStr
		}
		retNext = true
	}
	if minutes > 0 {
		dateStr = fmt.Sprintf("%s%dm", dateStr, minutes)
		if retNext {
			return dateStr
		}
	}
	return fmt.Sprintf("%s%ds", dateStr, seconds)
}

// LastSplitItem splits a string into a slice based on a split character and returns the last item.
func LastSplitItem(str, splitChar string) string {
	result := strings.Split(str, splitChar)
	if len(result) > 0 {
		return result[len(result)-1]
	}
	return ""
}

// TerminalSize tries to return the character dimensions of the terminal.
// It works through all the standard file descriptors until they are exhausted.
// That's because if a descriptor is being redirected, the call to term.GetSize() will fail.
func TerminalSize() (int, int, error) {
	var cols, rows int
	var err error
	if cols, rows, err = term.GetSize(int(os.Stdout.Fd())); err == nil {
		return cols, rows, nil
	}
	if cols, rows, err = term.GetSize(int(os.Stderr.Fd())); err == nil {
		return cols, rows, nil
	}
	return term.GetSize(int(os.Stdin.Fd()))
}

// TimeTaken is designed to be called via a defer statement to show elapsed time for things like functions/methods.
// e.g. put something like the this at the top of your function: `defer util.TimeTaken(time.Now(), "functionName")`.
func TimeTaken(start time.Time, name string) {
	fmt.Fprintf(os.Stderr, "%s took %s\n", name, time.Since(start))
}

// lineVisualWidth returns the visual width of a line with a string added taking tab stops into account.
// The line position of where the string is to be written is passed in since it affects the tabstop width at that point.
func lineVisualWidth(linePos int, str string) int {
	width := linePos
	for _, r := range str {
		if r == '\t' {
			width += tabStopWidth - (width % tabStopWidth)
		} else {
			width++
		}
	}
	return width
}

// WrapLine takes a string representing a single line and wraps it to a specified width.
// Any tab characters are handled based on the tabstop they'd pad out to.
func WrapLine(str string, width int) string {
	if len(str) == 0 {
		return ""
	}

	var currentLine, wrappedLine strings.Builder

	pos := 0
	prevWord := ""
	for _, word := range strings.Split(str, " ") {
		wordLength := len(word)

		// Preserve leading spaces which end up as empty words when split on a space.
		if wordLength == 0 {
			word = " "
		}

		// If the length of the current line + a space + the length of the word doesn't fit in the width, then
		// write out the current line, so that the word just read will start on a new line.
		// lineVisualWidth() is used since if a word contains tabs it will likely be visually wider.
		if pos > 0 && lineVisualWidth(pos+1, word) > width {
			wrappedLine.WriteString(currentLine.String() + "\n")
			currentLine.Reset()
			pos = 0
		}

		// Pad the new word with a leading space unless it is a space itself, or the previous word was a space.
		if pos > 0 && prevWord != " " && word != " " {
			currentLine.WriteString(" ")
			pos++
		}

		// If the word contains tabs, write it character by character incrementing pos as we go, except when we
		// get to a tab character where we increment pos by the number of characters to reach the next tabstop.
		if strings.Contains(word, `\t`) {
			for _, r := range word {
				currentLine.WriteRune(r)
				if r == '\t' {
					pos += tabStopWidth - (pos % tabStopWidth)
				} else {
					pos++
				}
			}
		} else {
			currentLine.WriteString(word)
			pos += wordLength
		}
		prevWord = word
	}

	// Add the last line we were working on.
	if pos > 0 {
		wrappedLine.WriteString(currentLine.String())
	}

	return wrappedLine.String()
}

// WrapTextToWidth takes paragraphs in a multi-line string and writes them into a string, wrapping before the specified
// width when possible, without breaking up words.
// It's not possible to wrap in time if a single word is longer than the width.
// Multiple lines of the paragraph are joined into the longest line that fits within the width before wrapping.
// Each paragraph in the multi-line string is represented by a blank line between them.
// The constructed string is then returned.
func WrapTextToWidth(width int, str string) string {
	// Get rid of an initial and final newlines on the string if it has them.
	str = strings.TrimPrefix(str, string('\n'))
	str = strings.TrimRight(str, string('\n'))

	// Produce an array of paragraphs joining multi-line paragraphs into a single line.
	var paragraph strings.Builder
	var paragraphs []string
	for _, line := range strings.Split(str, "\n") {
		if line != "" {
			// Add the line just read to the existing line since it is part of the same paragraph.
			if paragraph.Len() != 0 {
				paragraph.WriteString(" ")
			}
			paragraph.WriteString(line)
		} else {
			// When we encounter a blank line, we have completed a paragraph.
			paragraphs = append(paragraphs, paragraph.String())
			paragraph.Reset()
		}
	}
	// Add the last paragraph.
	if paragraph.Len() != 0 {
		paragraphs = append(paragraphs, paragraph.String())
	}

	// Format the paragraphs to the width we want.
	var finalStr strings.Builder
	lastParagraph := len(paragraphs) - 1
	for i, line := range paragraphs {
		finalStr.WriteString(WrapLine(line, width) + "\n")
		// Insert a newline between paragraphs to separate them.
		if i < lastParagraph {
			finalStr.WriteString("\n")
		}
	}
	return finalStr.String()
}
