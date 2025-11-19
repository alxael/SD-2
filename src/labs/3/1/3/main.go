package main

import (
	"fmt"
	"os"
)

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	fmt.Println("Error checking file:", err)
	return false
}

func printFileContents(filePath string) {
	if fileExists(filePath) {
		bytes, err := os.ReadFile(filePath)
		if err == nil {
			fmt.Println(string(bytes))
		} else {
			fmt.Println("Error reading file contents!")
		}
	} else {
		fmt.Println("File does not exist!")
	}
}

func main() {
	printFileContents("../src/labs/3/1/3/test.txt")
	printFileContents("../src/main.go")
}
