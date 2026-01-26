package main

import (
	"strings"

	"github.com/chzyer/readline"
)

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

		// Add command to history
		AddToHistory(input)

		// Check for pipe operators
		pipeIndices := []int{}
		for i, arg := range args {
			if arg == "|" {
				pipeIndices = append(pipeIndices, i)
			}
		}

		if len(pipeIndices) > 0 {
			// Execute pipeline (may have multiple pipes)
			shell.ExecutePipeline(args, pipeIndices)
		} else {
			// Execute single command
			cmd := args[0]
			cmdArgs := args[1:]
			shell.ExecuteCommand(cmd, cmdArgs)
		}
	}
}
