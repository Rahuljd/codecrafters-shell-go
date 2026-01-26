package main

import (
	"os"
	"strings"
)

// commandHistory stores all executed commands
var commandHistory []string

// lastSyncedIndex tracks the index up to which commands have been appended to file
// Used by history -a to only append new commands since last append
var lastSyncedIndex = 0

// AddToHistory adds a command to the history
func AddToHistory(cmd string) {
	commandHistory = append(commandHistory, cmd)
}

// PrependToHistory adds a command to the beginning of history
func PrependToHistory(cmd string) {
	commandHistory = append([]string{cmd}, commandHistory...)
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

// WriteHistoryToFile writes all history entries to a file
func WriteHistoryToFile(filepath string) error {
	// Create or truncate the file
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write each history entry on a separate line
	for _, cmd := range commandHistory {
		_, err := file.WriteString(cmd + "\n")
		if err != nil {
			return err
		}
	}

	return nil
}

// AppendHistoryToFile appends only new commands (since last sync) to the history file
func AppendHistoryToFile(filepath string) error {
	// Open file in append mode
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Append only the commands that haven't been synced yet
	for i := lastSyncedIndex; i < len(commandHistory); i++ {
		_, err := file.WriteString(commandHistory[i] + "\n")
		if err != nil {
			return err
		}
	}

	// Update the lastSyncedIndex to current history length
	lastSyncedIndex = len(commandHistory)

	return nil
}
