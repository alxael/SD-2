package main

import (
	"fmt"
	"runtime"
)

func generateSpeedReport(outputSize int, sectionCount int) {
	testCount := 100
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

	writer.Write([]string{"inputMegabytes", "maxSeconds", "maxMBPerSecond"})

	for _, inputBytes := range inputSizes {
		configuration := SpeedTestConfiguration{testCount, inputBytes}
		results := generateSpeedTests(outputSize, sectionCount, configuration)

		maxNanoseconds := int64(0)
		for _, nanoseconds := range results {
			if nanoseconds > maxNanoseconds {
				maxNanoseconds = nanoseconds
			}
		}

		maxSeconds := float64(maxNanoseconds) / 1e9
		megabytes := float64(inputBytes) / (1024 * 1024)
		maxMbPerSecond := megabytes / maxSeconds

		writer.Write([]string{
			fmt.Sprintf("%.6f", megabytes),
			fmt.Sprintf("%.6f", maxSeconds),
			fmt.Sprintf("%.4f", maxMbPerSecond),
		})
	}
}

func generateSpeedReportCores(outputSize int, sectionCount int, maxCores int) {
	testCount := 100
	inputBytes := 1 << 26
	megabytes := float64(inputBytes) / (1024 * 1024)

	writer, file, err := generateCsvReportFile("test-speed-cores")
	if err != nil {
		return
	}
	defer file.Close()
	defer writer.Flush()

	writer.Write([]string{"cores", "inputMegabytes", "bestSeconds", "bestMBPerSecond"})

	originalCores := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(originalCores)

	for cores := 2; cores <= maxCores; cores += 2 {
		runtime.GOMAXPROCS(cores)

		configuration := SpeedTestConfiguration{testCount, inputBytes}
		results := generateSpeedTests(outputSize, sectionCount, configuration)

		bestNanoseconds := results[0]
		for _, nanoseconds := range results[1:] {
			if nanoseconds < bestNanoseconds {
				bestNanoseconds = nanoseconds
			}
		}

		bestSeconds := float64(bestNanoseconds) / 1e9
		bestMbPerSecond := megabytes / bestSeconds

		writer.Write([]string{
			fmt.Sprintf("%d", cores),
			fmt.Sprintf("%.6f", megabytes),
			fmt.Sprintf("%.6f", bestSeconds),
			fmt.Sprintf("%.4f", bestMbPerSecond),
		})
	}
}
