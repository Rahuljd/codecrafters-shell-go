package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"
)

// executableCache stores found executables to avoid repeated filesystem calls
var executableCache map[string]bool

// completionState tracks the last completion attempt for multi-match listing
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

	// Single match: complete it with trailing space
	if len(matches) == 1 {
		match := matches[0]
		suffix := strings.TrimPrefix(match, prefix)
		newLine = append(newLine, []rune(suffix+" "))
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

			fmt.Println()
			for i, match := range sortedMatches {
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

		// Check for pipe operator
		pipeIndex := -1
		for i, arg := range args {
			if arg == "|" {
				pipeIndex = i
				break
			}
		}

		if pipeIndex != -1 {
			// Execute pipeline
			shell.ExecutePipeline(args, pipeIndex)
		} else {
			// Execute single command
			cmd := args[0]
			cmdArgs := args[1:]
			shell.ExecuteCommand(cmd, cmdArgs)
		}
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

	s.ExecuteCommandWithIO(cmd, cleanArgs, stdoutWriter, stderrWriter)
}

// isBuiltin checks if a command is a shell built-in
func isBuiltin(cmd string) bool {
	builtins := []string{"exit", "echo", "type", "pwd", "cd"}
	for _, b := range builtins {
		if cmd == b {
			return true
		}
	}
	return false
}

// ExecuteCommandWithIO executes a command with custom stdout/stderr writers and optional stdin
func (s *Shell) ExecuteCommandWithIO(cmd string, args []string, stdoutWriter, stderrWriter io.Writer) {
	s.ExecuteCommandWithIOAndStdin(cmd, args, os.Stdin, stdoutWriter, stderrWriter)
}

// ExecuteCommandWithIOAndStdin executes a command with custom stdin/stdout/stderr writers
func (s *Shell) ExecuteCommandWithIOAndStdin(cmd string, args []string, stdinReader io.Reader, stdoutWriter, stderrWriter io.Writer) {
	switch cmd {
	case "exit":
		os.Exit(0)
	case "echo":
		fmt.Fprintln(stdoutWriter, strings.Join(args, " "))
	case "type":
		if len(args) == 0 {
			return
		}
		if Builtins[args[0]] {
			fmt.Fprintf(stdoutWriter, "%s is a shell builtin\n", args[0])
		} else if path, err := exec.LookPath(args[0]); err == nil {
			fmt.Fprintf(stdoutWriter, "%s is %s\n", args[0], path)
		} else {
			fmt.Fprintf(stderrWriter, "%s: not found\n", args[0])
		}
	case "pwd":
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderrWriter, "pwd: %v\n", err)
		} else {
			fmt.Fprintln(stdoutWriter, cwd)
		}
	case "cd":
		if len(args) == 0 {
			return
		}
		dir := args[0]
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
		s.ExecuteExternalCommand(cmd, args, stdoutWriter, stderrWriter)
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

func (s *Shell) ExecutePipeline(args []string, pipeIndex int) {
	// Split args at pipe: [cmd1 args...] | [cmd2 args...]
	cmd1Args := args[:pipeIndex]
	cmd2Args := args[pipeIndex+1:]

	if len(cmd1Args) == 0 || len(cmd2Args) == 0 {
		fmt.Fprintf(os.Stderr, "error: invalid pipeline\n")
		return
	}

	cmd1 := cmd1Args[0]
	cmd1Opts := cmd1Args[1:]

	cmd2 := cmd2Args[0]
	cmd2Opts := cmd2Args[1:]

	// Create a pipe
	reader, writer, err := os.Pipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to create pipe: %v\n", err)
		return
	}

	// If first command is built-in, execute it and close the writer
	if isBuiltin(cmd1) {
		s.ExecuteCommandWithIOAndStdin(cmd1, cmd1Opts, os.Stdin, writer, os.Stderr)
		writer.Close()

		// Now execute the second command with the pipe reader
		if isBuiltin(cmd2) {
			s.ExecuteCommandWithIOAndStdin(cmd2, cmd2Opts, reader, os.Stdout, os.Stderr)
		} else {
			command2 := exec.Command(cmd2, cmd2Opts...)
			command2.Stdin = reader
			command2.Stdout = os.Stdout
			command2.Stderr = os.Stderr
			_ = command2.Run()
		}
		reader.Close()
		return
	}

	// Both commands are external: start them concurrently
	command1 := exec.Command(cmd1, cmd1Opts...)
	command1.Stdin = os.Stdin
	command1.Stdout = writer
	command1.Stderr = os.Stderr

	if err := command1.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: command not found\n", cmd1)
		writer.Close()
		reader.Close()
		return
	}

	// Close writer in parent process so command2 gets EOF when command1 finishes
	writer.Close()

	// Start second command
	if isBuiltin(cmd2) {
		// For built-in commands, execute them with the pipe reader as stdin
		s.ExecuteCommandWithIOAndStdin(cmd2, cmd2Opts, reader, os.Stdout, os.Stderr)
		reader.Close()
	} else {
		// For external commands, create a process
		command2 := exec.Command(cmd2, cmd2Opts...)
		command2.Stdin = reader
		command2.Stdout = os.Stdout
		command2.Stderr = os.Stderr

		if err := command2.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "%s: command not found\n", cmd2)
			reader.Close()
			return
		}

		// Wait for both commands to finish
		_ = command1.Wait()
		_ = command2.Wait()
		reader.Close()
		return
	}

	// If we reach here, cmd1 is external and cmd2 is built-in
	// Wait for command1 to finish
	_ = command1.Wait()
}
