package main

import (
	"fmt"
	"sync"
)

type KeyValue struct {
	Key   int
	Value bool
}

func Map(word string, key int) KeyValue {
	wordLength := len(word)
	for index := range wordLength {
		if word[index] != word[wordLength-index-1] {
			return KeyValue{
				Key:   key,
				Value: false,
			}
		}
	}
	return KeyValue{
		Key:   key,
		Value: true,
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
		{"a1551a", "parc", "ana", "minim", "1pcl3"},
		{"calabalac", "tivit", "leu", "zece10", "ploaie", "9ana9"},
		{"lalalal", "tema", "papa", "get"},
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
