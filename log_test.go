package main

import "testing"

func TestReverseEntries(t *testing.T) {
	log1 := Log{Date: "Mon Jun", Time: "10:00 AM", Type: "Toggle1"}
	log2 := Log{Date: "Tue May", Time: "12:00 AM", Type: "Toggle2"}
	entries := []Log{log1, log2}

	stringEqual(t, entries[0].Date, "Mon Jun")
	entries = ReverseEntries(entries)
	stringEqual(t, entries[0].Date, "Tue May")
}

func TestParseLogType(t *testing.T) {
	typeInFile := "TOGGLE DOOR"
	parsedType := ParseLogType(typeInFile)
	stringEqual(t, parsedType, "Toggle")
}

func TestParseDateTime(t *testing.T) {
	givenTime := "2016-07-06 23:03:43.384659988 -0500 CDT"
	date, time := ParseDateTime(givenTime)
	stringEqual(t, date, "Wed Jul 6 2016")
	stringEqual(t, time, "11:03 PM")
}
