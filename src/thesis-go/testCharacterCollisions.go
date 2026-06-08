package main

import (
	"crypto/rand"
	"strconv"
	"sync"
)

func generateCharacterCollisionsTest(outputSize int) {
	inputBitLength := 10000
	inputByteLength := inputBitLength / 8

	input := make([]byte, inputByteLength)
	rand.Read(input)

	originalDigest := hash(input, outputSize)

	histogram := make([]int, outputSize+1)
	equalBytesCounts := make([]int, inputBitLength)

	var waitGroup sync.WaitGroup
	for bitIndex := 0; bitIndex < inputBitLength; bitIndex++ {
		waitGroup.Add(1)
		go func(bitIndex int) {
			defer waitGroup.Done()

			flipped := make([]byte, inputByteLength)
			copy(flipped, input)
			flipped[bitIndex/8] ^= 1 << (bitIndex % 8)

			modifiedDigest := hash(flipped, outputSize)

			equalBytes := 0
			for index := 0; index < outputSize; index++ {
				if originalDigest[index] == modifiedDigest[index] {
					equalBytes++
				}
			}
			equalBytesCounts[bitIndex] = equalBytes
		}(bitIndex)
	}
	waitGroup.Wait()

	for _, count := range equalBytesCounts {
		histogram[count]++
	}

	writer, file, err := generateCsvReportFile("test-character-collisions")
	if err != nil {
		return
	}
	defer file.Close()
	defer writer.Flush()

	writer.Write([]string{"equalBytes", "collisionCount"})
	for equalBytes := 0; equalBytes <= outputSize; equalBytes++ {
		writer.Write([]string{
			strconv.Itoa(equalBytes),
			strconv.Itoa(histogram[equalBytes]),
		})
	}
}
