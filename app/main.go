package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/term"
)

var builtins = map[string]bool{
	"exit": true,
	"echo": true,
	"type": true,
	"pwd":  true,
	"cd":   true,
}

func main() {
	// Check if stdin is a TTY (interactive terminal)
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		// Not a TTY - use standard input loop (for testing/piping)
		standardInputLoop()
		return
	}

	// Enable raw mode for terminal input
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		// Fallback to standard input if raw mode fails
		standardInputLoop()
		return
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	var inputBuffer strings.Builder

	for {
		fmt.Print("$ ")

		for {
			b := make([]byte, 1)
			n, err := os.Stdin.Read(b)
			if err != nil || n == 0 {
				fmt.Println("Error reading input:", err)
				return
			}

			char := b[0]

			// TAB key (ASCII 9)
			if char == '\t' {
				partial := inputBuffer.String()
				completed := autocomplete(partial)
				if completed != partial {
					// Clear current line and print completed command
					fmt.Print("\r$ " + completed)
					inputBuffer.Reset()
					inputBuffer.WriteString(completed)
				}
				continue
			}

			// ENTER key (ASCII 13)
			if char == '\r' {
				fmt.Print("\n")
				input := inputBuffer.String()
				inputBuffer.Reset()

				if input != "" {
					processCommand(input)
				}
				break
			}

			// Backspace (ASCII 127) or Ctrl+H (ASCII 8)
			if char == 127 || char == 8 {
				if inputBuffer.Len() > 0 {
					str := inputBuffer.String()
					inputBuffer.Reset()
					inputBuffer.WriteString(str[:len(str)-1])
					fmt.Print("\b \b")
				}
				continue
			}

			// Printable characters
			if char >= 32 && char < 127 {
				inputBuffer.WriteByte(char)
				fmt.Print(string(char))
			}
		}
	}
}

func processCommand(input string) {
	words := parseInput(input)
	if len(words) == 0 {
		return
	}

	cmd := words[0]
	args := words[1:]
	args, errFile, errAppend := extractStderrRedirection(args)
	args, outFile, appendMode := extractStdoutRedirection(args)
	// Default: stderr should go to stdout (for tester visibility)
	stderrWriter := os.Stdout
	stdoutWriter := os.Stdout

	if outFile != "" {
		flags := os.O_CREATE | os.O_WRONLY
		if appendMode {
			flags |= os.O_APPEND // >>
		} else {
			flags |= os.O_TRUNC // >
		}

		f, err := os.OpenFile(outFile, flags, 0644)
		if err != nil {
			fmt.Println("error:", err)
			return
		}
		defer f.Close()
		stdoutWriter = f

	}

	// Redirect stderr if needed
	if errFile != "" {
		flags := os.O_CREATE | os.O_WRONLY
		if errAppend {
			flags |= os.O_APPEND // 2>>
		} else {
			flags |= os.O_TRUNC // 2>
		}

		f, err := os.OpenFile(errFile, flags, 0644)
		if err != nil {
			fmt.Println("error:", err)
			return
		}
		defer f.Close()
		stderrWriter = f

	}

	switch cmd {
	case "cd":
		if len(args) == 0 {
			return
		}
		dir := args[0]
		if dir == "~" || strings.HasPrefix(dir, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				fmt.Println("cd:", dir+":", "No such file or directory")
				return
			}
			if dir == "~" {
				dir = home
			} else {
				dir = filepath.Join(home, dir[2:])
			}
		}
		if err := os.Chdir(dir); err != nil {
			fmt.Println("cd:", dir+":", "No such file or directory")
		}
	case "pwd":
		dir, err := os.Getwd()
		if err != nil {
			fmt.Println("pwd:", err)
			return
		}
		fmt.Println(dir)
	case "exit":
		os.Exit(0)
	case "echo":
		fmt.Println(strings.Join(args, " "))
	case "type":
		if len(args) == 0 {
			return
		}
		if builtins[args[0]] {
			fmt.Println(args[0], "is a shell builtin")
		} else if path, err := exec.LookPath(args[0]); err == nil {
			fmt.Println(args[0], "is", path)
		} else {
			fmt.Println(args[0] + ": not found")
		}
	default:
		runExternal(cmd, args, stdoutWriter, stderrWriter)
	}
}

func standardInputLoop() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("$ ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			return
		}

		input = strings.TrimRight(input, "\n")
		if input == "" {
			continue
		}

		processCommand(input)
	}
}

func autocomplete(input string) string {
	input = strings.TrimLeft(input, " ")

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return input
	}

	prefix := parts[0]

	// Find all matching commands
	var matches []string
	for _, cmd := range []string{"echo", "exit"} {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}

	// Only autocomplete if there's exactly one match
	if len(matches) == 1 {
		return matches[0] + " "
	}

	return input
}

func extractStderrRedirection(args []string) (cleanArgs []string, errFile string, appendMode bool) {
	for i := 0; i < len(args); i++ {
		switch args[i] {

		case "2>":
			if i+1 < len(args) {
				return args[:i], args[i+1], false // overwrite
			}
			return args[:i], "", false

		case "2>>":
			if i+1 < len(args) {
				return args[:i], args[i+1], true // append
			}
			return args[:i], "", true
		}
	}
	return args, "", false
}

func parseInput(input string) []string {
	var args []string
	var current strings.Builder
	inSingleQuotes := false
	inDoubleQuotes := false
	escaped := false

	for i := 0; i < len(input); i++ {
		c := input[i]
		if escaped {
			current.WriteByte(c)
			escaped = false
			continue
		}
		if c == '\\' {
			if !inSingleQuotes && !inDoubleQuotes {
				escaped = true
				continue
			}
			if inDoubleQuotes && i+1 < len(input) {
				next := input[i+1]
				if next == '"' || next == '\\' {
					current.WriteByte(next)
					i++
					continue
				}
			}
			// Otherwise: literal backslash
			current.WriteByte(c)
			continue
		}
		switch c {
		case '\'':
			if !inDoubleQuotes {
				inSingleQuotes = !inSingleQuotes
			} else {
				current.WriteByte(c)
			}
		case '"':
			if !inSingleQuotes {
				inDoubleQuotes = !inDoubleQuotes
			} else {
				current.WriteByte(c)
			}
		case ' ', '\t', '\n':
			if inSingleQuotes || inDoubleQuotes {
				current.WriteByte(c)
			} else {
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
			}
		default:
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}
func extractStdoutRedirection(args []string) (cleanArgs []string, outFile string, appendMode bool) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case ">", "1>":
			if i+1 < len(args) {
				return args[:i], args[i+1], false // overwrite
			}
			return args[:i], "", false

		case ">>", "1>>":
			if i+1 < len(args) {
				return args[:i], args[i+1], true // append
			}
			return args[:i], "", true
		}
	}
	return args, "", false
}

func runExternal(command string, args []string, stdout, stderr *os.File) {
	_, err := exec.LookPath(command)
	if err != nil {
		fmt.Println(command + ": command not found")
		return
	}

	cmd := exec.Command(command, args...)

	// Connect program I/O directly to shell
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		// Do NOT print anything unless needed
		return
	}
}
