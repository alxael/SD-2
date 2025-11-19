package main

import "fmt"

func sendMessage(messageChannel chan string, messageType int) {
	if messageType == 0 {
		messageChannel <- "Hello world!"
	} else {
		messageChannel <- "Hello world! I'm alive!"
	}
}

func receiveMessage(messageChannel chan string) {
	message := <-messageChannel
	fmt.Println(message)
}

func main() {
	messageChannel := make(chan string)
	go sendMessage(messageChannel, 0)
	receiveMessage(messageChannel)
}
