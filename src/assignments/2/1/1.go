package main

import (
	"fmt"
	"strings"
	"sync"
)

type KeyValue struct {
	Key   int
	Value bool
}

func Map(word string, key int) KeyValue {
	vowels := "aeiou"
	vowelCount := 0
	for _, char := range strings.ToLower(word) {
		if strings.ContainsRune(vowels, char) {
			vowelCount++
		}
	}

	consonantCount := len(word) - vowelCount
	return KeyValue{
		Key:   key,
		Value: (vowelCount%2 == 0) && (consonantCount%3 == 0),
	}
}

func Reduce(key int, values []bool) int {
	count := 0
	for _, value := range values {
		if value {
			count++
		}
	}
	return count
}

func main() {
	input := [][]string{
		{"aabbb", "ebep", "blablablaa", "hijk", "wsww"},
		{"abba", "eeppp", "cocor", "ppppppaaa", "qwerty", "acasq"},
		{"lalala", "lalal", "papapa", "papap"},
	}

	mapResults := make([]KeyValue, 0)
	var waitGroup sync.WaitGroup
	var mutex sync.Mutex

	// mapping
	for index, words := range input {
		for _, word := range words {
			waitGroup.Add(1)
			go func(word string, index int) {
				defer waitGroup.Done()
				result := Map(word, index)
				mutex.Lock()
				mapResults = append(mapResults, result)
				mutex.Unlock()
			}(word, index)
		}
	}

	waitGroup.Wait()

	// shuffling
	groupedResults := make(map[int][]bool)

	for _, keyValue := range mapResults {
		groupedResults[keyValue.Key] = append(groupedResults[keyValue.Key], keyValue.Value)
	}

	// reducing
	finalResults := make(map[int]int)
	for key, values := range groupedResults {
		finalResults[key] = Reduce(key, values)
	}

	matchingWordTotal := 0
	for _, value := range finalResults {
		matchingWordTotal += value
	}
	average := 0.0
	if len(finalResults) > 0 {
		average = float64(matchingWordTotal) / float64(len(finalResults))
	}
	fmt.Printf("Average: %g\n", average)
}
