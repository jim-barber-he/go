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

	tabWidth = 8
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

// WrapTextToWidth takes paragraphs in a long string and writes them to a string, wrapping before width when possible.
// It's not possible to wrap in time if a single word is longer than the width.
// Each paragraph is represented by having a blank line between them.
// The constructed string is then returned.
func WrapTextToWidth(width int, str string) string {
	var s string
	var line string
	var lines []string
	var nl bool

	// Get rid of an initial newline on the string if it has one.
	str = strings.TrimPrefix(str, string('\n'))

	// Produce an array of lines, each representing a paragraph.
	for _, char := range str {
		if char != '\n' {
			// Build up string of non-newline characters.
			line += string(char)
			nl = false
		} else {
			if nl {
				// Second newline encountered in a row, so end of paragraph.
				if strings.TrimSuffix(line, " ") != "" {
					lines = append(lines, strings.TrimSuffix(line, " "))
					line = ""
				}
				nl = false
			} else {
				// Only 1 newline was encountered, so add a space for now.
				// If another newline is ecountered next time around the loop the space will be removed.
				line += " "
				nl = true
			}
		}
	}
	if strings.TrimSuffix(line, " ") != "" {
		lines = append(lines, line)
	}

	// Process each line, and break it up into words by splitting on the space character.
	// Write out words on a line until they are going to exceed the width, at which point start a new line.
	// At the end of each paragraph append a blank line (except for the last paragraph).
	lastLine := len(lines) - 1
	for i, l := range lines {
		c := 0
		for _, word := range strings.Split(l, " ") {
			wordLength := len(word)
			// Preserve leading space.
			if c == 0 && wordLength == 0 {
				word = " "
			}
			if c+1+wordLength > width {
				// If characters written on the line + a space + the word > width,
				// then will need to move to a new line before writing the word.
				if c != 0 {
					s += fmt.Sprintln()
					c = 0
				}
			} else {
				// The word will fit with what has already been written plus a space.
				// Need to add the space (and update the characters written) prior to writing the word.
				if c != 0 {
					s += " "
					c++
				}
			}

			// If the word contains tabs, write it character by character incrementing c, except when we get
			// to a tab character where we increment c by the number of characters to reach the tab-stop.
			if strings.Contains(word, `\t`) {
				for _, char := range word {
					s += string(char)
					if char == '\t' {
						// Add how many characters to the next tab-stop.
						c += tabWidth - (c % tabWidth)
					} else {
						c++
					}
				}
			} else {
				s += word
				c += wordLength
			}
		}
		s += fmt.Sprintln()
		if i < lastLine {
			s += fmt.Sprintln()
		}
	}

	return s
}
