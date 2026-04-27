package main

import (
	"fmt"
)

func generateSpeedReport(outputSize int, sectionCount int) {
	testCount := 10
	inputSizes := []int{
		1 << 18,
		1 << 20,
		1 << 22,
		1 << 24,
		1 << 26,
		1 << 28,
	}

	writer, file, err := generateCsvReportFile("test-speed")
	if err != nil {
		return
	}
	defer file.Close()
	defer writer.Flush()

	writer.Write([]string{"inputMegabytes", "averageSeconds", "averageMBPerSecond"})

	for _, inputBytes := range inputSizes {
		configuration := SpeedTestConfiguration{testCount, inputBytes}
		results := generateSpeedTests(outputSize, sectionCount, configuration)

		totalNs := int64(0)
		for _, ns := range results {
			totalNs += ns
		}
		averageNs := float64(totalNs) / float64(testCount)
		averageSeconds := averageNs / 1e9
		megabytes := float64(inputBytes) / (1024 * 1024)
		mbPerSecond := megabytes / averageSeconds

		writer.Write([]string{
			fmt.Sprintf("%.6f", megabytes),
			fmt.Sprintf("%.6f", averageSeconds),
			fmt.Sprintf("%.4f", mbPerSecond),
		})
	}
}
