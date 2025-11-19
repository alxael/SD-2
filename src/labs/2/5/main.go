package main

import (
	"fmt"
	"time"
)

func sendMessage(messageChannel chan string, message string) {
	time.Sleep(time.Second)
	messageChannel <- message
}

func main() {
	messageChannelOne := make(chan string)
	messageChannelTwo := make(chan string)

	go sendMessage(messageChannelOne, "Hello world!")
	go sendMessage(messageChannelTwo, "Hello world! I am alive!")

	for index := 0; index < 2; index++ {
		select {
		case messageOne := <-messageChannelOne:
			fmt.Println(messageOne)
		case messageTwo := <-messageChannelTwo:
			fmt.Println(messageTwo)
		}
	}
}
