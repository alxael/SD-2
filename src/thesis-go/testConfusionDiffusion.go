package main

import (
	"fmt"
	"math"
	"strconv"
)

func generateConfusionDiffusionMeansTests(outputSize int, sectionCount int) {
	configurations := []BitChangeTestConfiguration{
		{1e3, 10, 1e3},
		{2.5e3, 10, 1e3},
		{5e3, 10, 1e3},
		// {7.5e3, 10, 1e3},
		// {1e4, 10, 1e3},
		// {1e6, 10, 1e2},
	}

	writer, file, err := generateCsvReportFile("test-confusion-diffusion-means")
	if err != nil {
		return // fail silently
	}
	defer file.Close()
	defer writer.Flush()

	writer.Write([]string{"testCount", "meanChangedBits", "meanChangedBitsProbability", "minimumChangedBits", "maximumChangedBits", "standardDeviationChangedBits", "standardDeviationChangedBitsProbability"})

	for _, configuration := range configurations {
		results := generateBitChangeTests(outputSize, sectionCount, configuration)

		totalChangedBits := 0
		minChangedBits := results[0].changedBits
		maxChangedBits := results[0].changedBits
		for _, result := range results {
			totalChangedBits += result.changedBits
			if result.changedBits < minChangedBits {
				minChangedBits = result.changedBits
			}
			if result.changedBits > maxChangedBits {
				maxChangedBits = result.changedBits
			}
		}

		meanChangedBits := float64(totalChangedBits) / float64(configuration.testCount)
		meanBitChangeProbability := meanChangedBits / float64(outputSize*8) * 100

		sumSquaredBits := 0.0
		sumSquaredProbability := 0.0
		for _, result := range results {
			differentBits := float64(result.changedBits) - meanChangedBits
			differenceProbability := result.probability - meanBitChangeProbability/100
			sumSquaredBits += differentBits * differentBits
			sumSquaredProbability += differenceProbability * differenceProbability
		}
		standardDeviationBits := math.Sqrt(sumSquaredBits / float64(configuration.testCount-1))
		standardDeviationBitChangeProbability := math.Sqrt(sumSquaredProbability / float64(configuration.testCount-1))

		writer.Write([]string{
			strconv.Itoa(configuration.testCount),
			fmt.Sprintf("%.4f", meanChangedBits),
			fmt.Sprintf("%.4f", meanBitChangeProbability),
			strconv.Itoa(minChangedBits),
			strconv.Itoa(maxChangedBits),
			fmt.Sprintf("%.4f", standardDeviationBits),
			fmt.Sprintf("%.4f", standardDeviationBitChangeProbability*100),
		})
	}
}

func generateConfusionDiffusionSpreadTests(outputSize int, sectionCount int) {
	configuration := BitChangeTestConfiguration{1e4, 10, 1e4}
	results := generateBitChangeTests(outputSize, sectionCount, configuration)

	spreadWriter, spreadFile, err := generateCsvReportFile("test-confusion-diffusion-spread")
	if err != nil {
		return // fail silently
	}
	defer spreadFile.Close()
	defer spreadWriter.Flush()

	spreadWriter.Write([]string{"iteration", "changedBits"})
	for index, result := range results {
		spreadWriter.Write([]string{strconv.Itoa(index), strconv.Itoa(result.changedBits)})
	}

	totalBits := outputSize * 8
	frequency := make([]int, totalBits+1)
	for _, result := range results {
		frequency[result.changedBits]++
	}

	histogramWriter, histogramFile, err := generateCsvReportFile("test-confusion-diffusion-spread-histogram")
	if err != nil {
		return
	}
	defer histogramFile.Close()
	defer histogramWriter.Flush()

	histogramWriter.Write([]string{"changedBits", "frequency"})
	for index := range totalBits {
		histogramWriter.Write([]string{strconv.Itoa(index), strconv.Itoa(frequency[index])})
	}
}
