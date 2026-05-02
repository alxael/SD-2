package main

import (
	"fmt"
	"runtime"
)

func main() {
	outputSize := 16
	sectionCount := 256
	maxCores := 16

	runtime.GOMAXPROCS(maxCores)

	// speed tests
	generateSpeedReport(outputSize, sectionCount)
	fmt.Println("Generated speed tests!")

	// sensitivity tests
	generateSensitivityTests(outputSize, sectionCount)
	fmt.Println("Generated text example tests!")

	// statistical confusion and diffusion means tests
	generateConfusionDiffusionMeansTests(outputSize, sectionCount)
	fmt.Println("Generated confusion and diffusion means tests!")

	// confusion and diffusion spread tests
	generateConfusionDiffusionSpreadTests(outputSize, sectionCount)
	fmt.Println("Generated confusion and diffusion spread tests!")

	// null text distribution tests
	generateNullTextDistributionTest(outputSize, sectionCount)
	fmt.Println("Generated null text distribution test!")

	// sample text distribution tests
	generateSampleTextDistributionTest(outputSize, sectionCount)
	fmt.Println("Generated sample text distribution test!")

	// character collisions test
	generateCharacterCollisionsTest(outputSize, sectionCount)
	fmt.Println("Generated character collisions test!")

	// graphs
	generateSensitivityGraph()
	fmt.Println("Generated sensitivity graph!")

	generateValueDistributionGraphs()
	fmt.Println("Generated value distribution graphs!")

	generateConfusionDiffusionSpreadGraph(outputSize)
	fmt.Println("Generated confusion diffusion spread graph!")

	generateConfusionDiffusionHistogramGraph()
	fmt.Println("Generated confusion diffusion histogram graph!")

	generateCharacterCollisionsGraph()
	fmt.Println("Generated character collisions graph!")

	generateChaosAttractorGraph()
	fmt.Println("Generated chaos attractor graph!")

	generateSpeedGraph()
	fmt.Println("Generated speed graph!")
}
