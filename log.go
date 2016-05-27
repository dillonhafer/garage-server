package main

import (
	"bufio"
	"os"
	"strings"
	"time"
)

type Log struct {
	Date string `json:"date"`
	Time string `json:"time"`
	Type string `json:"type"`
}

type Logs struct {
	Entries []Log `json:"entries"`
}

func ParseDateTime(dateTime string) (formattedDate string, formattedTime string) {
	layout := "2006-01-02 15:04:05.000000000 -0700 MST"
	t, _ := time.Parse(layout, dateTime)
	formattedTime = t.Format("3:04 PM")
	formattedDate = t.Format("Mon Jan 2 2006")
	return formattedDate, formattedTime
}

func ParseLogType(logType string) string {
	return strings.Title(strings.ToLower(strings.Split(logType, " ")[0]))
}

func ParseLogs(logFile string) Logs {
	file, _ := os.Open(logFile)
	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "TOGGLE DOOR") {
			lines = append(lines, line)
		}
	}

	entries := []Log{}
	for _, line := range lines {
		logSlice := strings.Split(line, " - ")
		logType := ParseLogType(logSlice[0])
		logDate, logTime := ParseDateTime(logSlice[1])
		log := Log{Date: logDate, Time: logTime, Type: logType}
		entries = append(entries, log)
	}

	return Logs{Entries: ReverseEntries(entries)}
}

func ReverseEntries(entries []Log) []Log {
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries
}
