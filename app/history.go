package main

import (
	"os"
	"strings"
)

// commandHistory stores all executed commands
var commandHistory []string

// AddToHistory adds a command to the history
func AddToHistory(cmd string) {
	commandHistory = append(commandHistory, cmd)
}

// GetHistory returns the entire command history
func GetHistory() []string {
	return commandHistory
}

// ClearHistory clears all history entries
func ClearHistory() {
	commandHistory = []string{}
}

// LoadHistoryFromFile loads history from a file
func LoadHistoryFromFile(filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	// Clear current history and load from file
	ClearHistory()
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			AddToHistory(line)
		}
	}
	return nil
}
