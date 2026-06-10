package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
)

func generateSensitivityTests(outputSize int) {
	initial := "Timpul petrecut cu pisici nu este niciodată irosit."
	variants := []string{
		"Timpul petrecut cu pisaci nu este niciodată irosit.",  // 1. i din pisici -> a
		"Timpul petrecuT cu pisici nu este niciodată irosit.",  // 2. t din petrecut -> T
		"Timpul petrecut cu pisici nu este niciodată irosit ",  // 3. punct -> spațiu
		"Timpul petrecut cu pisici!nu este niciodată irosit.",  // 4. spațiu după pisici -> !
		"Timpul petrecut cu pisici nu este niciodată iirosit.", // 5. i adăugat la începutul cuvântului irosit
	}

	messages := append([]string{initial}, variants...)
	initialDigest := hash([]byte(initial), outputSize)

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
		digest := hash([]byte(message), outputSize)
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
