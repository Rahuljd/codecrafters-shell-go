package main

import (
	"fmt"
)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Print

func main() {
	// TODO: Uncomment the code below to pass the first stage
	var word string
	for {
		fmt.Print("$ ")
		_, err := fmt.Scan(&word)
		if err != nil {
			fmt.Println("Error reading input:", err)
			return
		}
		fmt.Println(word + ": command not found")
	}
}
