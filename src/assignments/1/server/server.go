package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"unicode"
)

// config utilities

type ServerConfig struct {
	Port int `json:"port"`
}

func loadServerConfig(configPath string) ServerConfig {
	file, err := os.Open(configPath)
	if err != nil {
		fmt.Println("[Server] - error opening config file:", err)
		os.Exit(1)
	}
	defer file.Close()

	var serverConfig ServerConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&serverConfig); err != nil {
		fmt.Println("[Server] - error decoding config file:", err)
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

// connection handler

func handleConnection(connection net.Conn) {
	defer connection.Close()
	reader := bufio.NewReader(connection)

	messageBytes, err := reader.ReadBytes('\n')
	if err != nil {
		fmt.Printf("[Server] - error reading from connection: %v\n", err)
		return
	}

	fmt.Printf("[Server] - received message\n")

	var message Message
	if err := json.Unmarshal(messageBytes, &message); err != nil {
		fmt.Printf("[Server] - error unmarshalling message json: %v\n", err)
		return
	}

	fmt.Printf("[Server] - [%s] - processing data\n", message.Name)

	response, err := getResponseForMessage(message)
	if err != nil {
		response = Response{
			IsSuccess: false,
			Data:      err.Error(),
		}
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("[Server] - [%s] - error marshalling response: %v\n", message.Name, err)
		return
	}

	fmt.Printf("[Server] - [%s] - sending response to client\n", message.Name)

	connection.Write(append(responseBytes, '\n'))
}

func getResponseForMessage(message Message) (Response, error) {
	switch message.Action {
	case SameLengthString:
		return handleSameLengthString(message)
	case ExtractPerfectSquares:
		return handleExtractPerfectSquares(message)
	case ReversedIntegerSum:
		return handleReversedIntegerSum(message)
	case DigitSumInRange:
		return handleDigitSumInRange(message)
	case ConvertBinaryToDecimal:
		return handleConvertBinaryToDecimal(message)
	default:
		return Response{
			IsSuccess: false,
			Data:      "Error determining action type!",
		}, nil
	}
}

// utility functions

func splitByComma(str string) []string {
	parts := strings.Split(str, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func joinByComma(strs []string) string {
	return strings.Join(strs, ",")
}

func isPerfectSquare(number int) bool {
	if number < 0 {
		return false
	}
	root := int(math.Sqrt(float64(number)))
	return root*root == number
}

func convertStringsToInts(strs []string) ([]int, error) {
	nums := make([]int, len(strs))
	for i, s := range strs {
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil, errors.New("all elements of the array should be valid numbers")
		}
		nums[i] = n
	}
	return nums, nil
}

func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func sumDigits(number int) int {
	sum := 0
	absoluteNumber := int(math.Abs(float64(number))) // handle negative numbers
	for absoluteNumber > 0 {
		sum += absoluteNumber % 10
		absoluteNumber /= 10
	}
	return sum
}

// specific requests

func handleSameLengthString(message Message) (Response, error) {
	strs := splitByComma(message.Payload)

	length := len(strs[0])
	for _, str := range strs {
		if len(str) != length {
			return Response{}, errors.New("all strings should be of same length")
		}
	}

	var result []string
	for index := 0; index < length; index++ {
		var currentStr []rune
		for _, str := range strs {
			currentStr = append(currentStr, []rune(str)[index])
		}
		result = append(result, string(currentStr))
	}

	data := joinByComma(result)
	return Response{
		IsSuccess: true,
		Data:      data,
	}, nil
}

func handleExtractPerfectSquares(message Message) (Response, error) {
	strs := splitByComma(message.Payload)

	count := 0
	for _, str := range strs {
		var digitsBuilder strings.Builder
		for _, char := range str {
			if unicode.IsDigit(char) {
				digitsBuilder.WriteRune(char)
			}
		}
		if digitsBuilder.Len() == 0 {
			continue
		}
		numStr := digitsBuilder.String()
		num, err := strconv.Atoi(numStr)
		if err != nil {
			continue
		}
		if isPerfectSquare(num) {
			count++
		}
	}

	return Response{
		IsSuccess: true,
		Data:      strconv.Itoa(count),
	}, nil
}

func handleReversedIntegerSum(message Message) (Response, error) {
	reversedNumberStrings := splitByComma(reverseString(message.Payload))

	numbers, err := convertStringsToInts(reversedNumberStrings)
	if err != nil {
		return Response{}, err
	}

	sum := 0
	for _, number := range numbers {
		sum += number
	}

	return Response{
		IsSuccess: true,
		Data:      strconv.Itoa(sum),
	}, nil
}

func handleDigitSumInRange(message Message) (Response, error) {
	numberStrings := splitByComma(message.Payload)

	numbers, err := convertStringsToInts(numberStrings)
	if err != nil {
		return Response{}, err
	}

	if len(numbers) < 4 {
		return Response{}, errors.New("you need to provide at least three values (a,b,n)")
	}

	a := numbers[0]
	b := numbers[1]
	n := numbers[2]

	if a > b {
		return Response{}, errors.New("a should be smaller than b")
	}

	if len(numbers) != n+3 {
		return Response{}, errors.New("the number of values provided should equal n+3")
	}

	sum := 0
	count := 0
	for i := range n {
		digitSum := sumDigits(numbers[i+3])
		if a <= digitSum && digitSum <= b {
			sum += numbers[i+3]
			count++
		}
	}

	if count != 0 {
		sum /= count
	}

	return Response{
		IsSuccess: true,
		Data:      strconv.Itoa(sum),
	}, nil
}

func handleConvertBinaryToDecimal(message Message) (Response, error) {
	strs := splitByComma(message.Payload)

	var result []string
	for _, str := range strs {
		number, err := strconv.ParseInt(str, 2, 64)
		if err == nil {
			fmt.Println(number)
			result = append(result, strconv.FormatInt(number, 10))
		}
	}

	return Response{
		IsSuccess: true,
		Data:      joinByComma(result),
	}, nil
}

func main() {
	serverConfig := loadServerConfig("../src/assignments/1/server/server-config.json")

	address := fmt.Sprintf(":%d", serverConfig.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("[Server] - error starting server:", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Printf("[Server] - listening on port %d...\n", serverConfig.Port)

	for {
		connection, err := listener.Accept()
		if err != nil {
			fmt.Println("[Server] - error accepting connection:", err)
			continue
		}
		go handleConnection(connection)
	}
}
