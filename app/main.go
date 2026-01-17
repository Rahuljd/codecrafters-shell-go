package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var builtins = map[string]bool{
	"exit": true,
	"echo": true,
	"type": true,
	"pwd":  true,
	"cd":   true,
}

func main() {

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("$ ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			return
		}
		words := parseInput(input)
		if len(words) == 0 {
			continue
		}

		cmd := words[0]
		args := words[1:]
		args, outFile := extractStdoutRedirection(args)

		var (
			file       *os.File
			origStdout *os.File
		)

		if outFile != "" {
			var err error
			file, err = os.OpenFile(
				outFile,
				os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
				0644,
			)
			if err != nil {
				fmt.Println("error:", err)
				continue
			}
			origStdout = os.Stdout
			os.Stdout = file
		}

		switch cmd {
		case "cd":
			if len(args) == 0 {
				continue
			}
			dir := args[0]
			if dir == "~" || strings.HasPrefix(dir, "~/") {
				home, err := os.UserHomeDir()
				if err != nil {
					fmt.Println("cd:", dir+":", "No such file or directory")
					continue
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
				continue
			}
			fmt.Println(dir)
		case "exit":
			return
		case "echo":
			fmt.Println(strings.Join(args, " "))
		case "type":
			if len(args) == 0 {
				continue
			}
			if builtins[args[0]] {
				fmt.Println(args[0], "is a shell builtin")
				continue
			}
			if path, err := exec.LookPath(args[0]); err == nil {
				fmt.Println(args[0], "is", path)
			} else {
				fmt.Println(args[0] + ": not found")
			}
		default:
			runExternal(cmd, args)

		}
		if outFile != "" {
			os.Stdout = origStdout
			file.Close()
		}
	}
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
func extractStdoutRedirection(args []string) (cleanArgs []string, outFile string) {
	for i := 0; i < len(args); i++ {
		if args[i] == ">" || args[i] == "1>" {
			if i+1 < len(args) {
				outFile = args[i+1]
			}
			return args[:i], outFile
		}
	}
	return args, ""
}

func runExternal(command string, args []string) {
	_, err := exec.LookPath(command)
	if err != nil {
		fmt.Println(command + ": command not found")
		return
	}

	cmd := exec.Command(command, args...)

	// Connect program I/O directly to shell
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Do NOT print anything unless needed
		return
	}
}
