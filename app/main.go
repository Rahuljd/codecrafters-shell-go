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
