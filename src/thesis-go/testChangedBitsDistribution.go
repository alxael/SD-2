package main

import (
	"crypto/rand"
	"math/bits"
	mathrand "math/rand"
	"strconv"
	"sync"
)

// generateChangedBitsDistributionTest takes a number of random messages, hashes
// each one, flips a single random input bit, hashes again, and records which
// output bits changed. The result is a histogram indexed by digest bit position:
// entry i counts how many of the messages had output bit i flip between the two
// digests.
func generateChangedBitsDistributionTest(outputSize int) {
	const inputBitLength = 10000
	const inputByteLength = inputBitLength / 8
	const messageCount = 10000

	outputBits := outputSize * 8
	histogram := make([]int, outputBits)
	var histogramMutex sync.Mutex

	var waitGroup sync.WaitGroup
	for messageIndex := 0; messageIndex < messageCount; messageIndex++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()

			input := make([]byte, inputByteLength)
			rand.Read(input)

			flipped := make([]byte, inputByteLength)
			copy(flipped, input)
			bitIndex := mathrand.Intn(inputByteLength * 8)
			flipped[bitIndex/8] ^= 1 << (bitIndex % 8)

			original := hash(input, outputSize)
			modified := hash(flipped, outputSize)

			local := make([]int, outputBits)
			for byteIndex := 0; byteIndex < outputSize; byteIndex++ {
				difference := original[byteIndex] ^ modified[byteIndex]
				for difference != 0 {
					bit := bits.TrailingZeros8(difference)
					local[byteIndex*8+bit]++
					difference &= difference - 1
				}
			}

			histogramMutex.Lock()
			for i := 0; i < outputBits; i++ {
				histogram[i] += local[i]
			}
			histogramMutex.Unlock()
		}()
	}
	waitGroup.Wait()

	writer, file, err := generateCsvReportFile("test-changed-bits-distribution")
	if err != nil {
		return
	}
	defer file.Close()
	defer writer.Flush()

	writer.Write([]string{"bitIndex", "changeCount"})
	for bitIndex := 0; bitIndex < outputBits; bitIndex++ {
		writer.Write([]string{
			strconv.Itoa(bitIndex),
			strconv.Itoa(histogram[bitIndex]),
		})
	}
}
