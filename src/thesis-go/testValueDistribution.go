package main

import (
	"encoding/hex"
	"strconv"
	"strings"
)

func writeDistributionCSVs(input string, outputSize int, label string) {
	inputWriter, inputFile, err := generateCsvReportFile("test-value-distribution-" + label + "-input")
	if err != nil {
		return
	}
	defer inputFile.Close()
	defer inputWriter.Flush()

	inputWriter.Write([]string{"index", "value"})
	for index := range input {
		inputWriter.Write([]string{strconv.Itoa(index), strconv.Itoa(int(input[index]))})
	}

	digest := hash([]byte(input), outputSize)
	hexString := hex.EncodeToString(digest)

	outputWriter, outputFile, err := generateCsvReportFile("test-value-distribution-" + label + "-output")
	if err != nil {
		return
	}
	defer outputFile.Close()
	defer outputWriter.Flush()

	outputWriter.Write([]string{"index", "value"})
	for index := range hexString {
		outputWriter.Write([]string{strconv.Itoa(index), strconv.Itoa(hexDigitValue(hexString[index]))})
	}
}

func generateNullTextDistributionTest(outputSize int) {
	input := strings.Repeat("\x00", 1000)
	writeDistributionCSVs(input, outputSize, "null")
}

func generateSampleTextDistributionTest(outputSize int) {
	sampleText := "The cat (Felis catus), also called domestic cat and house cat, is a " +
		"small carnivorous mammal. It is an obligate carnivore, requiring a predominantly " +
		"meat-based diet. Its retractable claws are adapted to killing small prey species " +
		"such as mice and rats. It has a strong, flexible body, quick reflexes, and sharp " +
		"teeth, and its night vision and sense of smell are well developed. It is a social " +
		"species, but a solitary hunter and a crepuscular predator. The domestic cat is the " +
		"only domesticated species of the family Felidae. Advances in archaeology and genetics " +
		"have shown that the domestication of the cat started in the Near East around 7500 BCE. " +
		"Today, the domestic cat occurs across the globe and is valued by humans for companionship " +
		"and its ability to kill vermin. It is commonly kept as a pet, working cat, and pedigreed " +
		"cat shown at cat fancy events. Cats have been used for millennia to control rodents, " +
		"notably around grain stores and aboard ships, and both uses extend to the present day. " +
		"Cats are also used in the international fur trade and leather industries for making coats, " +
		"hats, blankets, and stuffed toys."

	if len(sampleText) > 1000 {
		sampleText = sampleText[:1000]
	}

	writeDistributionCSVs(sampleText, outputSize, "sample")
}
