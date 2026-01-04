package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	// TODO: Uncomment the code below to pass the first stage
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

/* gemini

func main() {
    scanner := bufio.NewScanner(os.Stdin) // Scanner is often preferred over Reader for line-by-line
    for {
        fmt.Print("$ ")
        if !scanner.Scan() {
            break
        }
        input := scanner.Text()
        words := strings.Fields(input)

        if len(words) == 0 {
            continue
        }

        command := words[0]
        args := words[1:] // Efficient slicing

        switch command {
        case "exit":
            return
        case "echo":
            fmt.Println(strings.Join(args, " ")) // More efficient than manual looping
        default:
            fmt.Printf("%s: command not found\n", command)
        }
    }
}

*/
