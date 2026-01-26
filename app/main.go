package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chzyer/readline"
)

// executableCache stores found executables to avoid repeated filesystem calls
var executableCache map[string]bool

// completionState tracks the last completion attempt for multiple match handling
var completionState struct {
	lastPrefix string
	matches    []string
}

// isExecAny checks if a file has execute permissions
func isExecAny(mode os.FileMode) bool {
	return mode.Perm()&0111 != 0
}

// findAllExes searches PATH for all executable files
func findAllExes() {
	executableCache = make(map[string]bool)
	paths := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))

	for _, p := range paths {
		entries, err := os.ReadDir(p)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				info, err := entry.Info()
				if err != nil {
					continue
				}

				// On Windows, also check for .exe, .bat, .cmd, .com extensions
				isExecutable := false
				if os.PathSeparator == '\\' {
					name := strings.ToLower(entry.Name())
					if strings.HasSuffix(name, ".exe") ||
						strings.HasSuffix(name, ".bat") ||
						strings.HasSuffix(name, ".cmd") ||
						strings.HasSuffix(name, ".com") {
						isExecutable = true
					}
				} else {
					// Unix: check execute bit
					isExecutable = isExecAny(info.Mode())
				}

				if isExecutable {
					executableCache[entry.Name()] = true
				}
			}
		}
	}
}

// getPathExecutables returns all executables found in PATH directories
func getPathExecutables() map[string]bool {
	if executableCache == nil {
		findAllExes()
	}
	return executableCache
}

type BellCompleter struct {
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

	prefix := strings.TrimSpace(lineStr[wordStart:])

	if prefix == "" {
		fmt.Print("\x07") // Bell for empty input
		completionState.lastPrefix = ""
		completionState.matches = nil
		return [][]rune{}, 0
	}

	// First check builtins
	builtins := []string{"exit", "echo", "type", "pwd", "cd"}
	var matches []string

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

	// If still no matches, ring the bell
	if len(matches) == 0 {
		fmt.Print("\x07")
		completionState.lastPrefix = ""
		completionState.matches = nil
		return [][]rune{}, 0
	}

	// Handle multiple matches
	if len(matches) > 1 {
		// Check if this is a repeated TAB press on the same prefix
		if completionState.lastPrefix == prefix && len(completionState.matches) > 1 {
			// Second TAB press: print all matches
			sort.Strings(matches)
			fmt.Println()
			for i, match := range matches {
				if i > 0 {
					fmt.Print("  ") // Two spaces between matches
				}
				fmt.Print(match)
			}
			fmt.Println()

			// Show prompt again with original prefix
			fmt.Print("$ " + prefix)

			// Return empty to prevent readline from doing its own completion
			completionState.lastPrefix = ""
			completionState.matches = nil
			return [][]rune{}, 0
		}

		// First TAB press on multiple matches: ring the bell
		fmt.Print("\x07")
		completionState.lastPrefix = prefix
		completionState.matches = matches
		return [][]rune{}, 0
	}

	// Single match: complete it
	// Return only the SUFFIX after the prefix (readline will delete the prefix using length parameter)
	match := matches[0]
	suffix := strings.TrimPrefix(match, prefix)
	newLine = append(newLine, []rune(suffix+" "))

	// Reset state for single match completion
	completionState.lastPrefix = ""
	completionState.matches = nil

	// Return length of prefix to delete from line
	length = len(prefix)
	return newLine, length
}

type RedirectionType int

const (
	TokenRedirectOut RedirectionType = iota
	TokenRedirectAppend
	TokenRedirectIn
	TokenRedirect2
	TokenRedirect22
)

type Redirection struct {
	Type     RedirectionType
	Filename string
}

type Shell struct {
}

var Builtins = map[string]bool{
	"exit": true, "echo": true, "type": true, "cd": true, "pwd": true,
}

func InputParser(input string) (string, []string) {
	var word strings.Builder
	var newArr []string
	preserveNextLiteral := false
	backslashInQuotes := false
	inQuotes := false
	quoteChar := rune(0)

	for _, ch := range input {
		if preserveNextLiteral {
			word.WriteRune(ch)
			preserveNextLiteral = false
			continue
		}
		if backslashInQuotes {
			if ch == '$' || ch == '\\' || ch == '"' || ch == '`' {
				word.WriteRune(ch)
			} else {
				word.WriteRune('\\')
				word.WriteRune(ch)
			}
			backslashInQuotes = false
			continue
		}

		switch {
		case ch == '"' || ch == '\'':
			if !inQuotes {
				inQuotes = true
				quoteChar = ch
			} else if ch == quoteChar {
				inQuotes = false
				quoteChar = rune(0)
			} else {
				word.WriteRune(ch)
			}
		case ch == '\\':
			if !inQuotes {
				preserveNextLiteral = true
			} else if quoteChar == '"' {
				backslashInQuotes = true
			} else {
				word.WriteRune(ch)
			}
		case ch == ' ':
			if inQuotes {
				word.WriteRune(ch)
			} else if word.Len() > 0 {
				newArr = append(newArr, word.String())
				word.Reset()
			}
		default:
			word.WriteRune(ch)
		}
	}

	if word.Len() > 0 {
		newArr = append(newArr, word.String())
	}
	if len(newArr) == 0 {
		return "", nil
	}

	noSingles := strings.ReplaceAll(input, "'", "")
	noDoubles := strings.ReplaceAll(noSingles, `"`, "")
	output := noDoubles

	return output, newArr
}

func main() {
	shell := &Shell{}

	rl, _ := readline.New("$ ")

	// Use our custom completer that handles both builtins and PATH executables
	bellCompleter := &BellCompleter{}
	rl.Config.AutoComplete = bellCompleter

	for {
		input, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				continue
			}
			return
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		_, args := InputParser(input)
		if len(args) == 0 {
			continue
		}

		cmd := args[0]
		cmdArgs := args[1:]

		shell.ExecuteCommand(cmd, cmdArgs)
	}
}

func (s *Shell) ExecuteCommand(cmd string, args []string) {
	// Extract redirections
	var stdoutFile, stderrFile string
	var appendOut, appendErr bool
	var cleanArgs []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case ">":
			if i+1 < len(args) {
				stdoutFile = args[i+1]
				appendOut = false
				i++
			}
		case ">>":
			if i+1 < len(args) {
				stdoutFile = args[i+1]
				appendOut = true
				i++
			}
		case "1>":
			if i+1 < len(args) {
				stdoutFile = args[i+1]
				appendOut = false
				i++
			}
		case "1>>":
			if i+1 < len(args) {
				stdoutFile = args[i+1]
				appendOut = true
				i++
			}
		case "2>":
			if i+1 < len(args) {
				stderrFile = args[i+1]
				appendErr = false
				i++
			}
		case "2>>":
			if i+1 < len(args) {
				stderrFile = args[i+1]
				appendErr = true
				i++
			}
		default:
			cleanArgs = append(cleanArgs, args[i])
		}
	}

	// Setup output writers
	var stdoutWriter io.Writer = os.Stdout
	var stderrWriter io.Writer = os.Stderr

	if stdoutFile != "" {
		flags := os.O_CREATE | os.O_WRONLY
		if appendOut {
			flags |= os.O_APPEND
		} else {
			flags |= os.O_TRUNC
		}
		f, err := os.OpenFile(stdoutFile, flags, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		defer f.Close()
		stdoutWriter = f
	}

	if stderrFile != "" {
		flags := os.O_CREATE | os.O_WRONLY
		if appendErr {
			flags |= os.O_APPEND
		} else {
			flags |= os.O_TRUNC
		}
		f, err := os.OpenFile(stderrFile, flags, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
		defer f.Close()
		stderrWriter = f
	}

	switch cmd {
	case "exit":
		os.Exit(0)
	case "echo":
		fmt.Fprintln(stdoutWriter, strings.Join(cleanArgs, " "))
	case "type":
		if len(cleanArgs) == 0 {
			return
		}
		if Builtins[cleanArgs[0]] {
			fmt.Fprintf(stdoutWriter, "%s is a shell builtin\n", cleanArgs[0])
		} else if path, err := exec.LookPath(cleanArgs[0]); err == nil {
			fmt.Fprintf(stdoutWriter, "%s is %s\n", cleanArgs[0], path)
		} else {
			fmt.Fprintf(stderrWriter, "%s: not found\n", cleanArgs[0])
		}
	case "pwd":
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderrWriter, "pwd: %v\n", err)
		} else {
			fmt.Fprintln(stdoutWriter, cwd)
		}
	case "cd":
		if len(cleanArgs) == 0 {
			return
		}
		dir := cleanArgs[0]
		if dir == "~" || strings.HasPrefix(dir, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(stderrWriter, "cd: %s: No such file or directory\n", dir)
				return
			}
			if dir == "~" {
				dir = home
			} else {
				dir = filepath.Join(home, dir[2:])
			}
		}
		if err := os.Chdir(dir); err != nil {
			fmt.Fprintf(stderrWriter, "cd: %s: No such file or directory\n", dir)
		}
	default:
		s.ExecuteExternalCommand(cmd, cleanArgs, stdoutWriter, stderrWriter)
	}
}

func (s *Shell) ExecuteExternalCommand(cmd string, args []string, stdoutWriter, stderrWriter io.Writer) {
	// Check if command exists first
	if _, err := exec.LookPath(cmd); err != nil {
		fmt.Fprintf(stderrWriter, "%s: command not found\n", cmd)
		return
	}

	command := exec.Command(cmd, args...)
	command.Stdin = os.Stdin
	command.Stdout = stdoutWriter
	command.Stderr = stderrWriter

	_ = command.Run()
}
