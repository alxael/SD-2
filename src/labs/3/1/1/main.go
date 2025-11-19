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

func checkIfFileExists(filePath string) {
	if fileExists(filePath) {
		fmt.Println("File exists!")
	} else {
		fmt.Println("File does not exist!")
	}
}

func main() {
	checkIfFileExists("../src/go.mod")
	checkIfFileExists("../src/main.go")
}
