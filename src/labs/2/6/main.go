package main

import (
	"fmt"
	"time"
)

func sendMessage(messageChannel chan string, message string, sleepSeconds int) {
	time.Sleep(time.Duration(sleepSeconds) * time.Second)
	messageChannel <- message
}

func main() {
	messageChannel := make(chan string)

	go sendMessage(messageChannel, "Task finished successfully!", 1)
	select {
	case message := <-messageChannel:
		fmt.Println(message)
	case <-time.After(2 * time.Second):
		fmt.Println("Request timed out!")
	}

	go sendMessage(messageChannel, "Task finished successfully!", 2)
	select {
	case message := <-messageChannel:
		fmt.Println(message)
	case <-time.After(1 * time.Second):
		fmt.Println("Request timed out!")
	}
}
