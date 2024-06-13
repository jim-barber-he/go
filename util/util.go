/*
Package util provides various utility functions.
*/
package util

import (
	"fmt"
	"strings"
	"time"
)

const (
	numSecondsPerWeek   = 60 * 60 * 24 * 7
	numSecondsPerDay    = 60 * 60 * 24
	numSecondsPerHour   = 60 * 60
	numSecondsPerMinute = 60
)

// Return the age in a human readable format of the first 2 non-zero time units from weeks to seconds,
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

// LastSplitItem() splits a string into a slice based on a split character and returns the last item.
func LastSplitItem(str, splitChar string) string {
	result := strings.Split(str, splitChar)
	if len(result) > 0 {
		return result[len(result)-1]
	}
	return ""
}
