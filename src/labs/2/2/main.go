package main

import "fmt"

func sendMessages(messageChannel chan string) {
	messageChannel <- "Hello world!"
	messageChannel <- "Hello world! I'm alive!"
}

func receiveMessage(messageChannel chan string) {
	message := <-messageChannel
	fmt.Println(message)
}

func main() {
	messageChannel := make(chan string)
	go sendMessages(messageChannel)

	receiveMessage(messageChannel)
	receiveMessage(messageChannel)
}
