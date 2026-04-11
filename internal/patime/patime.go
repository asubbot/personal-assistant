// Package patime provides calendar and clock helpers in pa_timezone (EP-002).
package patime

import "time"

// PreviousCalendarDate returns the calendar date (year, month, day) in loc
// for the day immediately before the calendar date of t in loc.
func PreviousCalendarDate(loc *time.Location, t time.Time) (year int, month time.Month, day int) {
	if loc == nil {
		loc = time.UTC
	}
	y, m, d := t.In(loc).Date()
	mid := time.Date(y, m, d, 12, 0, 0, 0, loc)
	prev := mid.AddDate(0, 0, -1)
	py, pm, pd := prev.Date()
	return py, pm, pd
}

// CalendarDateOf returns the calendar components of t in loc (wall date).
func CalendarDateOf(loc *time.Location, t time.Time) (year int, month time.Month, day int) {
	if loc == nil {
		loc = time.UTC
	}
	return t.In(loc).Date()
}

// NoonOnCalendar returns noon on the given calendar date in loc (avoids DST edge at midnight).
func NoonOnCalendar(loc *time.Location, year int, month time.Month, day int) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	return time.Date(year, month, day, 12, 0, 0, 0, loc)
}

// NextClockAfter returns the earliest instant strictly after `from` such that
// the local time in loc is hour:min:00 (seconds/nanos zero). If `from` is already
// past that wall time today, the next calendar day is used.
//
// If hour or min is outside 0–23 / 0–59, NextClockAfter returns from unchanged
// (no error). Callers must pass a validated wall clock; built-in summarization
// uses fixed constants (01:00).
func NextClockAfter(loc *time.Location, from time.Time, hour, min int) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	if hour < 0 || hour > 23 || min < 0 || min > 59 {
		return from
	}
	local := from.In(loc)
	y, m, d := local.Date()
	candidate := time.Date(y, m, d, hour, min, 0, 0, loc)
	if !candidate.After(from) {
		candidate = candidate.AddDate(0, 0, 1)
	}
	return candidate
}

// IsFirstDayOfMonth reports whether t's calendar date in loc is the first day of its month.
func IsFirstDayOfMonth(loc *time.Location, t time.Time) bool {
	if loc == nil {
		loc = time.UTC
	}
	_, _, d := t.In(loc).Date()
	return d == 1
}

// IsFirstDayOfYear reports whether t's calendar date in loc is January 1.
func IsFirstDayOfYear(loc *time.Location, t time.Time) bool {
	if loc == nil {
		loc = time.UTC
	}
	_, m, d := t.In(loc).Date()
	return m == time.January && d == 1
}

// PreviousMonth returns (year, month) of the calendar month immediately before t's month in loc.
func PreviousMonth(loc *time.Location, t time.Time) (year int, month time.Month) {
	if loc == nil {
		loc = time.UTC
	}
	y, m, _ := t.In(loc).Date()
	first := time.Date(y, m, 1, 12, 0, 0, 0, loc)
	prev := first.AddDate(0, -1, 0)
	py, pm, _ := prev.Date()
	return py, pm
}

// PreviousYear returns the calendar year immediately before t's year in loc.
func PreviousYear(loc *time.Location, t time.Time) int {
	if loc == nil {
		loc = time.UTC
	}
	y, _, _ := t.In(loc).Date()
	return y - 1
}
