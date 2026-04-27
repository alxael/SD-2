package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
)

func generateSensitivityTests(outputSize int, sectionCount int) {
	initial := "Time spent with cats is never wasted."
	variants := []string{
		"Time spent with cars is never wasted.",  // 1. t in cats -> r
		"Time spenT with cats is never wasted.",  // 2. t in spent -> T
		"Time spent with cats is never wasted ",  // 3. period -> space
		"Time spent with cats!is never wasted.",  // 4. space after cats -> !
		"Times spent with cats is never wasted.", // 5. s added after Time
	}

	messages := append([]string{initial}, variants...)
	initialDigest := hash([]byte(initial), outputSize, sectionCount)

	err := os.MkdirAll("reports", 0755)
	if err != nil {
		fmt.Println("Could not create reports directory:", err)
		return
	}

	writer, file, err := generateCsvReportFile("test-sensitivity")
	if err != nil {
		return // fail silently
	}
	defer file.Close()
	defer writer.Flush()

	writer.Write([]string{"message", "hash", "changedBits", "changedBitsPercentage"})

	for index, message := range messages {
		digest := hash([]byte(message), outputSize, sectionCount)
		hexString := hex.EncodeToString(digest)

		changedBitsPercentage := ""
		changedBits := 0
		if index > 0 {
			result := bitDifference(initialDigest, digest, outputSize)
			changedBitsPercentage = fmt.Sprintf("%.2f", result.probability*100)
			changedBits = result.changedBits
		}

		writer.Write([]string{message, hexString, strconv.Itoa(changedBits), changedBitsPercentage})
	}
}
