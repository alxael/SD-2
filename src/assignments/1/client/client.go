package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
)

// config utilities

type ClientConfig struct {
	ServerUrl string    `json:"serverUrl"`
	Messages  []Message `json:"messages"`
}

func loadClientConfig(configPath string) ClientConfig {
	file, err := os.Open(configPath)
	if err != nil {
		fmt.Println("[Client] - error opening config file:", err)
		os.Exit(1)
	}
	defer file.Close()

	var serverConfig ClientConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&serverConfig); err != nil {
		fmt.Println("[Client] - error decoding config file:", err)
		os.Exit(1)
	}

	return serverConfig
}

// types

type MessageAction string

const (
	SameLengthString       MessageAction = "SAME_LENGTH_STRING"
	ExtractPerfectSquares  MessageAction = "EXTRACT_PERFECT_SQUARES"
	ReversedIntegerSum     MessageAction = "REVERSED_INTEGER_SUM"
	DigitSumInRange        MessageAction = "DIGIT_SUM_IN_RANGE"
	ConvertBinaryToDecimal MessageAction = "CONVERT_BINARY_TO_DECIMAL"
)

type Message struct {
	Name    string        `json:"name"`
	Action  MessageAction `json:"action"`
	Payload string        `json:"payload"`
}

type Response struct {
	IsSuccess bool   `json:"isSuccess"`
	Data      string `json:"data"`
}

// connection creator

func createConnection(serverUrl string, message Message) {
	connection, err := net.Dial("tcp", serverUrl)
	if err != nil {
		fmt.Printf("[Client] - [%s] - error connecting to server: %v\n", message.Name, err)
		return
	}
	defer connection.Close()

	jsonBytes, err := json.Marshal(message)
	if err != nil {
		fmt.Printf("[Client] - [%s] - error marshalling json: %v\n", message.Name, err)
		return
	}

	fmt.Printf("[Client] - [%s] - connection successful!\n", message.Name)
	fmt.Fprintf(connection, "%s\n", jsonBytes)

	responseBytes, err := bufio.NewReader(connection).ReadBytes('\n')
	if err != nil {
		fmt.Printf("[Client] - [%s] - error reading response: %v\n", message.Name, err)
		return
	}

	var response Response
	err = json.Unmarshal(responseBytes, &response)
	if err != nil {
		fmt.Printf("[Client] - [%s] - error unmarshalling json: %v\n", message.Name, err)

		return
	}

	if response.IsSuccess {
		fmt.Printf("[Client] - [%s] - client received success response: %s\n", message.Name, response.Data)
	} else {
		fmt.Printf("[Client] - [%s] - client received failure response: %s\n", message.Name, response.Data)
	}

}

func main() {
	clientConfig := loadClientConfig("../src/assignments/1/client/client-config.json")

	for _, message := range clientConfig.Messages {
		go createConnection(clientConfig.ServerUrl, message)
	}

	var input string
	fmt.Scanln(&input)
}
