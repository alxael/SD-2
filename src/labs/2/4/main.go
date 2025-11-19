package main

import "fmt"

func sendMessageToChannel(messageChannel chan string, message string) {
	messageChannel <- message
}

func receiveMessageFromChannel(sourceMessageChannel chan string, destinationMessageChannel chan string) {
	message := <-sourceMessageChannel
	destinationMessageChannel <- message
}

func main() {
	sourceMessageChannel := make(chan string)
	destinationMessageChannel := make(chan string)

	go sendMessageToChannel(sourceMessageChannel, "Hello world!")
	go receiveMessageFromChannel(sourceMessageChannel, destinationMessageChannel)

	fmt.Println(<-destinationMessageChannel)
}
