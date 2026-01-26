package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// isBuiltin checks if a command is a shell built-in
func isBuiltin(cmd string) bool {
	builtins := []string{"exit", "echo", "type", "pwd", "cd", "history"}
	for _, b := range builtins {
		if cmd == b {
			return true
		}
	}
	return false
}

var Builtins = map[string]bool{
	"exit": true, "echo": true, "type": true, "cd": true, "pwd": true, "history": true,
}

type Shell struct{}

// ExecuteCommand executes a single command with redirection support
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

// ExecuteCommandWithIO executes a command with custom stdout/stderr writers
func (s *Shell) ExecuteCommandWithIO(cmd string, args []string, stdoutWriter, stderrWriter io.Writer) {
	s.ExecuteCommandWithIOAndStdin(cmd, args, os.Stdin, stdoutWriter, stderrWriter)
}

// ExecuteCommandWithIOAndStdin executes a command with custom stdin/stdout/stderr writers
func (s *Shell) ExecuteCommandWithIOAndStdin(cmd string, args []string, stdinReader io.Reader, stdoutWriter, stderrWriter io.Writer) {
	switch cmd {
	case "exit":
		shouldExit = true
		return
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
	case "history":
		// Handle history -w <path> to write to file
		if len(args) > 0 && args[0] == "-w" {
			if len(args) < 2 {
				fmt.Fprintf(stderrWriter, "history: -w requires a path\n")
				return
			}
			filePath := args[1]
			// Write history to file (includes the history -w command itself since it was already added to history)
			err := WriteHistoryToFile(filePath)
			if err != nil {
				fmt.Fprintf(stderrWriter, "history: %v\n", err)
			}
			return
		}

		// Handle history -a <path> to append new commands to file
		if len(args) > 0 && args[0] == "-a" {
			if len(args) < 2 {
				fmt.Fprintf(stderrWriter, "history: -a requires a path\n")
				return
			}
			filePath := args[1]
			// Append new commands to history file (includes the history -a command itself since it was already added to history)
			err := AppendHistoryToFile(filePath)
			if err != nil {
				fmt.Fprintf(stderrWriter, "history: %v\n", err)
			}
			return
		}

		// Handle history -r <path> to read from file
		if len(args) > 0 && args[0] == "-r" {
			if len(args) < 2 {
				fmt.Fprintf(stderrWriter, "history: -r requires a path\n")
				return
			}
			filePath := args[1]
			// Save the current history command before loading from file
			currentCmd := ""
			if len(commandHistory) > 0 {
				currentCmd = commandHistory[len(commandHistory)-1]
			}
			// Load from file (this clears history)
			err := LoadHistoryFromFile(filePath)
			if err != nil {
				fmt.Fprintf(stderrWriter, "history: %v\n", err)
				return
			}
			// Prepend the history -r command so it appears as the first entry
			if currentCmd != "" {
				PrependToHistory(currentCmd)
			}
			return
		}

		// Display history, optionally limited to last n entries
		var displayCount int
		if len(args) > 0 {
			// Parse the optional number argument
			n, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Fprintf(stderrWriter, "history: %s: invalid argument\n", args[0])
				return
			}
			displayCount = n
		} else {
			displayCount = len(commandHistory)
		}

		// Ensure we don't try to show more history than exists
		startIndex := len(commandHistory) - displayCount
		if startIndex < 0 {
			startIndex = 0
		}

		// Display the requested history entries
		for i := startIndex; i < len(commandHistory); i++ {
			fmt.Fprintf(stdoutWriter, "    %d  %s\n", i+1, commandHistory[i])
		}
	default:
		s.ExecuteExternalCommand(cmd, args, stdoutWriter, stderrWriter)
	}
}

// ExecuteExternalCommand executes an external command
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

// ExecutePipeline executes a pipeline of commands
func (s *Shell) ExecutePipeline(args []string, pipeIndices []int) {
	if len(pipeIndices) == 0 {
		fmt.Fprintf(os.Stderr, "error: invalid pipeline\n")
		return
	}

	// Build command segments
	var commandSegments [][]string
	prevIndex := 0

	for _, pipeIndex := range pipeIndices {
		segment := args[prevIndex:pipeIndex]
		if len(segment) == 0 {
			fmt.Fprintf(os.Stderr, "error: invalid pipeline\n")
			return
		}
		commandSegments = append(commandSegments, segment)
		prevIndex = pipeIndex + 1
	}

	// Add the last segment
	lastSegment := args[prevIndex:]
	if len(lastSegment) == 0 {
		fmt.Fprintf(os.Stderr, "error: invalid pipeline\n")
		return
	}
	commandSegments = append(commandSegments, lastSegment)

	// Create pipes between commands (n-1 pipes for n commands)
	pipes := make([]*os.File, 0)
	readers := make([]*os.File, 0)

	for i := 0; i < len(commandSegments)-1; i++ {
		reader, writer, err := os.Pipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to create pipe: %v\n", err)
			// Clean up already created pipes
			for _, p := range pipes {
				p.Close()
			}
			for _, r := range readers {
				r.Close()
			}
			return
		}
		pipes = append(pipes, writer)
		readers = append(readers, reader)
	}

	// Keep track of started external commands for waiting
	var commands []*exec.Cmd

	// Execute all commands in the pipeline
	for i, segment := range commandSegments {
		cmd := segment[0]
		cmdArgs := segment[1:]

		// Determine stdin and stdout for this command
		var cmdStdin io.Reader = os.Stdin
		var cmdStdout io.Writer = os.Stdout

		// All commands except the first get their stdin from the previous pipe
		if i > 0 {
			cmdStdin = readers[i-1]
		}

		// All commands except the last write their stdout to the next pipe
		if i < len(pipes) {
			cmdStdout = pipes[i]
		}

		// If the command is built-in, execute it
		if isBuiltin(cmd) {
			s.ExecuteCommandWithIOAndStdin(cmd, cmdArgs, cmdStdin, cmdStdout, os.Stderr)

			// Close the writer for built-in commands so the next command gets EOF
			if i < len(pipes) {
				pipes[i].Close()
			}
		} else {
			// External command: create and start it
			command := exec.Command(cmd, cmdArgs...)
			command.Stdin = cmdStdin
			command.Stdout = cmdStdout
			command.Stderr = os.Stderr

			if err := command.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "%s: command not found\n", cmd)
				// Clean up pipes
				for _, p := range pipes {
					p.Close()
				}
				for _, r := range readers {
					r.Close()
				}
				return
			}

			commands = append(commands, command)

			// Close the writer in parent process after starting the command
			if i < len(pipes) {
				pipes[i].Close()
			}
		}
	}

	// Close all pipes and readers in parent
	for _, p := range pipes {
		p.Close()
	}
	for _, r := range readers {
		r.Close()
	}

	// Wait for all started external commands to finish
	for _, command := range commands {
		_ = command.Wait()
	}
}
