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
		if c == '\\' && !inSingleQuotes && !inDoubleQuotes {
			escaped = true
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
