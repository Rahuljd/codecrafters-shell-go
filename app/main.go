package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var builtins = map[string]bool{
	"exit": true,
	"echo": true,
	"type": true,
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
		words := strings.Fields(input)
		if len(words) == 0 {
			continue
		}

		cmd := words[0]
		args := words[1:]

		switch cmd {
		case "exit":
			return
		case "echo":
			fmt.Println(strings.Join(args, ""))
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
			fmt.Println(cmd, ": command not found")
		}
	}
}
