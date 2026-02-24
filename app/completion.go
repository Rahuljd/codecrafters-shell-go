package main

import (
	"fmt"
	"os"
	"strings"
)

type BellCompleter struct {
}

type CompletionState struct {
	lastPrefix string
	matches    []string
}

var completionState CompletionState

// findLongestCommonPrefix finds the longest common prefix of all strings in the list
func findLongestCommonPrefix(matches []string) string {
	if len(matches) == 0 {
		return ""
	}
	if len(matches) == 1 {
		return matches[0]
	}

	// Find the shortest string to limit the search
	minLen := len(matches[0])
	for _, m := range matches[1:] {
		if len(m) < minLen {
			minLen = len(m)
		}
	}

	// Find common prefix character by character
	for i := 0; i < minLen; i++ {
		char := matches[0][i]
		for _, m := range matches[1:] {
			if m[i] != char {
				return matches[0][:i]
			}
		}
	}

	return matches[0][:minLen]
}

// Do implements the readline.AutoCompleter interface
func (b *BellCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	lineStr := string(line[:pos])

	// Find the start of the current word (first word, since we're completing commands)
	// by finding the last space or the beginning of the line
	wordStart := 0
	for i := len(lineStr) - 1; i >= 0; i-- {
		if lineStr[i] == ' ' {
			wordStart = i + 1
			break
		}
	}

	rawPrefix := lineStr[wordStart:]
	// Determine if we're completing the first word (command) or an argument
	isFirstWord := wordStart == 0
	prefix := strings.TrimSpace(rawPrefix)

	// If empty prefix on first word, ring the bell and stop.
	if prefix == "" && isFirstWord {
		fmt.Print("\x07") // Bell for empty input
		completionState.lastPrefix = ""
		completionState.matches = nil
		return [][]rune{}, 0
	}

	// If completing the first word, check builtins and PATH executables
	var matches []string
	if isFirstWord {
		builtins := []string{"exit", "echo", "type", "pwd", "cd", "history"}
		for _, cmd := range builtins {
			if strings.HasPrefix(cmd, prefix) {
				matches = append(matches, cmd)
			}
		}

		// If no builtins matched, check PATH executables
		if len(matches) == 0 {
			pathExecs := getPathExecutables()
			for exec := range pathExecs {
				if strings.HasPrefix(exec, prefix) {
					matches = append(matches, exec)
				}
			}
		}
	}

	// If the prefix contains a slash, attempt path completion in the specified directory
	if len(matches) == 0 && strings.Contains(prefix, "/") {
		// split at last '/'
		last := strings.LastIndex(prefix, "/")
		dirPath := prefix[:last+1] // includes trailing '/'
		namePrefix := prefix[last+1:]

		// Read the target directory (relative to current working directory)
		entries, err := os.ReadDir(dirPath)
		if err == nil {
			var dirMatches []string
			for _, e := range entries {
				name := e.Name()
				if strings.HasPrefix(name, namePrefix) {
					// full path to match
					dirMatches = append(dirMatches, dirPath+name)
				}
			}

			if len(dirMatches) == 1 {
				// Single match: complete the full path. If it's a directory,
				// append a trailing '/' (no space). Otherwise append a space.
				match := dirMatches[0]
				if fi, err := os.Stat(match); err == nil && fi.IsDir() {
					suffix := strings.TrimPrefix(match, prefix)
					newLine = append(newLine, []rune(suffix+"/"))
				} else {
					suffix := strings.TrimPrefix(match, prefix)
					newLine = append(newLine, []rune(suffix+" "))
				}
				completionState.lastPrefix = ""
				completionState.matches = nil
				length = len(prefix)
				return newLine, length
			}

			// If multiple matches, use them as 'matches' (so LCP logic can run)
			if len(dirMatches) > 0 {
				matches = append(matches, dirMatches...)
			}
		}
	}

	if len(matches) == 0 {
		// When completing arguments (not the first word) and prefix is empty,
		// we should consider all entries in the current directory.
		entries, err := os.ReadDir(".")
		if err == nil {
			for _, e := range entries {
				name := e.Name()
				if prefix == "" {
					// empty prefix: include all entries
					matches = append(matches, name)
				} else {
					if strings.HasPrefix(name, prefix) {
						matches = append(matches, name)
					}
				}
			}
		}
	}

	// If still no matches, ring the bell
	if len(matches) == 0 {
		fmt.Print("\x07")
		completionState.lastPrefix = ""
		completionState.matches = nil
		return [][]rune{}, 0
	}

	// Single match: if it's a directory append '/', otherwise append space
	if len(matches) == 1 {
		match := matches[0]
		suffix := strings.TrimPrefix(match, prefix)
		if fi, err := os.Stat(match); err == nil && fi.IsDir() {
			newLine = append(newLine, []rune(suffix+"/"))
		} else {
			newLine = append(newLine, []rune(suffix+" "))
		}
		completionState.lastPrefix = ""
		completionState.matches = nil
		length = len(prefix)
		return newLine, length
	}

	// Multiple matches: find the longest common prefix
	lcp := findLongestCommonPrefix(matches)

	// If LCP equals the current prefix (no progress can be made with LCP)
	if lcp == prefix {
		// Check if this is a repeated TAB press on the same prefix
		if completionState.lastPrefix == prefix && len(completionState.matches) > 0 {
			// Second TAB press: print all matches in alphabetical order
			sortedMatches := make([]string, len(matches))
			copy(sortedMatches, matches)

			// Sort matches alphabetically
			for i := 0; i < len(sortedMatches); i++ {
				for j := i + 1; j < len(sortedMatches); j++ {
					if sortedMatches[j] < sortedMatches[i] {
						sortedMatches[i], sortedMatches[j] = sortedMatches[j], sortedMatches[i]
					}
				}
			}

			// Prepare display names: show basenames and append '/' for directories
			displayNames := make([]string, 0, len(sortedMatches))
			for _, match := range sortedMatches {
				base := match
				if idx := strings.LastIndex(match, "/"); idx != -1 {
					base = match[idx+1:]
				}
				if fi, err := os.Stat(match); err == nil && fi.IsDir() {
					base = base + "/"
				}
				displayNames = append(displayNames, base)
			}

			fmt.Println()
			for i, name := range displayNames {
				if i > 0 {
					fmt.Print("  ") // Two spaces between matches
				}
				fmt.Print(name)
			}
			fmt.Println()

			// Show prompt again with original input (preserve already typed content)
			fmt.Print("$ " + lineStr)

			// Return empty to prevent readline from doing its own completion
			completionState.lastPrefix = ""
			completionState.matches = nil
			return [][]rune{}, 0
		}

		// First TAB press on multiple matches with no LCP progress: ring the bell
		fmt.Print("\x07")
		completionState.lastPrefix = prefix
		completionState.matches = matches
		return [][]rune{}, 0
	}

	// Multiple matches with LCP progress: complete to the LCP
	suffix := strings.TrimPrefix(lcp, prefix)
	newLine = append(newLine, []rune(suffix))
	completionState.lastPrefix = ""
	completionState.matches = nil
	length = len(prefix)
	return newLine, length
}
