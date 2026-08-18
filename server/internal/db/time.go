package db

import "time"

// TimeFmt is the fixed storage format for all timestamps: UTC with
// millisecond precision, matching PostgreSQL timestamptz JSON output
// ("2026-08-18T05:36:00.000Z"). Fixed-width strings keep lexicographic
// SQL comparisons correct.
const TimeFmt = "2006-01-02T15:04:05.000Z07:00"

// FormatTime formats a time for storage.
func FormatTime(t time.Time) string {
	return t.UTC().Format(TimeFmt)
}

// ParseTime parses a stored timestamp.
func ParseTime(s string) (time.Time, error) {
	return time.Parse(TimeFmt, s)
}

// Now returns the current UTC time in storage format.
func Now() string {
	return time.Now().UTC().Format(TimeFmt)
}
