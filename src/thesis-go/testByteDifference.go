package main

import (
	"crypto/rand"
	mathrand "math/rand"
	"strconv"
	"sync"
)

// runByteDifferenceTest generates a random message, hashes it, flips a single
// random input bit, hashes again, and returns the sum over all digest bytes of
// the absolute differences between the unsigned decimal byte values of the two
// digests: sum_{i=1}^{outputSize} |Dec(h_i) - Dec(h'_i)|.
func runByteDifferenceTest(outputSize, inputByteLength int) int {
	input := make([]byte, inputByteLength)
	rand.Read(input)

	flipped := make([]byte, inputByteLength)
	copy(flipped, input)
	bitIndex := mathrand.Intn(inputByteLength * 8)
	flipped[bitIndex/8] ^= 1 << (bitIndex % 8)

	original := hash(input, outputSize)
	modified := hash(flipped, outputSize)

	sum := 0
	for index := 0; index < outputSize; index++ {
		difference := int(original[index]) - int(modified[index])
		if difference < 0 {
			difference = -difference
		}
		sum += difference
	}
	return sum
}

// generateByteDifferenceTest runs the byte-difference avalanche test for several
// test counts and writes a CSV summarising the minimum, maximum, mean and
// per-byte mean of the byte-difference sum for each run count.
func generateByteDifferenceTest(outputSize int) {
	const inputBitLength = 10000
	const inputByteLength = inputBitLength / 8

	testCounts := []int{1000, 5000, 10000}

	writer, file, err := generateCsvReportFile("test-byte-difference")
	if err != nil {
		return
	}
	defer file.Close()
	defer writer.Flush()

	writer.Write([]string{"testCount", "minimum", "maximum", "mean", "meanPerByte"})

	for _, testCount := range testCounts {
		sums := make([]int, testCount)

		var waitGroup sync.WaitGroup
		for index := 0; index < testCount; index++ {
			waitGroup.Add(1)
			go func(index int) {
				defer waitGroup.Done()
				sums[index] = runByteDifferenceTest(outputSize, inputByteLength)
			}(index)
		}
		waitGroup.Wait()

		minimum := sums[0]
		maximum := sums[0]
		total := 0
		for _, sum := range sums {
			if sum < minimum {
				minimum = sum
			}
			if sum > maximum {
				maximum = sum
			}
			total += sum
		}

		mean := float64(total) / float64(testCount)
		meanPerByte := mean / float64(outputSize)

		writer.Write([]string{
			strconv.Itoa(testCount),
			strconv.Itoa(minimum),
			strconv.Itoa(maximum),
			strconv.FormatFloat(mean, 'f', 4, 64),
			strconv.FormatFloat(meanPerByte, 'f', 4, 64),
		})
	}
}
