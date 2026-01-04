package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	for {
		fmt.Print("$ ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			return
		}
		words := strings.Fields(input)
		if words[0] == "exit" {
			break
		} else if words[0] == "type" {
			if words[1] == "type" || words[1] == "exit" || words[1] == "echo" {
				fmt.Println(words[1] + " is a shell builtin")
			} else {
				fmt.Println(strings.Join(words[1:], " ") + ": not found")
			}
		} else if words[0] == "echo" {
			printStringArray(words[1:])
			fmt.Println()
		} else {
			printStringArray(words[0:])
			fmt.Println(": command not found")
		}
	}
}

func printStringArray(strArr []string) {
	for i, word := range strArr {
		fmt.Print(word)
		if i < len(strArr)-1 {
			fmt.Print(" ")
		}
	}
}

/* chatgpt optimized

package main

import (
	"bufio"
	"fmt"
	"os"
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
			continue // ignore empty input
		}

		cmd := words[0]
		args := words[1:]

		switch cmd {

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
			} else {
				fmt.Println(strings.Join(args, " "), ": not found")
			}

		default:
			fmt.Println(cmd, ": command not found")
		}
	}
}


*/
