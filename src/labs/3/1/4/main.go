package main

import (
	"bufio"
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

func readFileContents(filePath string) {
	if fileExists(filePath) {
		file, err := os.Open(filePath)
		if err != nil {
			fmt.Println("Error reading file contents!")
			return
		}

		allBytes := []byte{}
		reader := bufio.NewReader(file)
		bufferSize := 128
		buffer := make([]byte, bufferSize)

		for {
			readBytes, err := reader.Read(buffer)
			if readBytes > 0 {
				fmt.Println(buffer[:readBytes])
				allBytes = append(allBytes, buffer[:readBytes]...)
			}
			if err != nil {
				if err.Error() == "EOF" {
					fmt.Println("Total bytes read: ", len(allBytes))
					return
				}
				fmt.Println("Error reading file contents!")
			}
		}
	} else {
		fmt.Println("File does not exist!")
	}
}

func main() {
	readFileContents("../src/labs/3/1/4/test.txt")
	readFileContents("../src/main.go")
}
