package main

import (
	"crypto/rand"
	"encoding/csv"
	"fmt"
	"math/bits"
	mathrand "math/rand"
	"os"
	"sync"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
)

// bit difference

type BitDifferenceResult struct {
	changedBits int
	probability float64
}

func bitDifference(a []byte, b []byte, outputSize int) BitDifferenceResult {
	totalBits := outputSize * 8
	changedBits := 0
	for index := range a[:outputSize] {
		changedBits += bits.OnesCount8(a[index] ^ b[index])
	}
	return BitDifferenceResult{
		changedBits: changedBits,
		probability: float64(changedBits) / float64(totalBits),
	}
}

// report file generation

func generateCsvReportFile(reportName string) (*csv.Writer, *os.File, error) {
	err := os.MkdirAll("reports", 0755)
	if err != nil {
		fmt.Println("Could not create reports directory:", err)
		return nil, nil, err
	}

	file, err := os.Create("reports/" + reportName + ".csv")
	if err != nil {
		fmt.Println("Could not create report file:", err)
		return nil, nil, err
	}

	writer := csv.NewWriter(file)

	return writer, file, nil
}

// bit change test generation

type BitChangeTestConfiguration struct {
	testCount    int
	minTestBytes int
	maxTestBytes int
}

func runBitChangeTest(outputSize int, configuration BitChangeTestConfiguration) BitDifferenceResult {
	inputSize := configuration.minTestBytes + mathrand.Intn(configuration.maxTestBytes-configuration.minTestBytes+1)

	input := make([]byte, inputSize)
	rand.Read(input)

	flipped := make([]byte, inputSize)
	copy(flipped, input)
	bitIndex := mathrand.Intn(inputSize * 8)
	flipped[bitIndex/8] ^= 1 << (bitIndex % 8)

	original := hash(input, outputSize)
	modified := hash(flipped, outputSize)

	return bitDifference(original, modified, outputSize)
}

func generateBitChangeTests(outputSize int, configuration BitChangeTestConfiguration) []BitDifferenceResult {
	results := make([]BitDifferenceResult, configuration.testCount)
	var waitGroup sync.WaitGroup

	for index := range configuration.testCount {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			results[index] = runBitChangeTest(outputSize, configuration)
		}(index)
	}
	waitGroup.Wait()

	return results
}

// speed test generation

type SpeedTestConfiguration struct {
	testCount int
	testBytes int
}

func runSpeedTest(outputSize int, configuration SpeedTestConfiguration) int64 {
	input := make([]byte, configuration.testBytes)
	rand.Read(input)

	start := time.Now()
	hash(input, outputSize)
	duration := time.Since(start)

	return int64(duration.Nanoseconds())
}

func generateSpeedTests(outputSize int, configuration SpeedTestConfiguration) []int64 {
	results := make([]int64, configuration.testCount)
	for index := range configuration.testCount {
		results[index] = runSpeedTest(outputSize, configuration)
	}
	return results
}

// report file reading

func readCsvReportFile(reportName string) ([][]string, error) {
	file, err := os.Open("reports/" + reportName + ".csv")
	if err != nil {
		fmt.Println("Could not open report file:", err)
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Could not read report file:", err)
		return nil, err
	}

	return records, nil
}

// graph styling

func applyLargePlotStyle(p *plot.Plot) {
	p.Title.TextStyle.Font.Size = vg.Points(20)
	p.X.Label.TextStyle.Font.Size = vg.Points(16)
	p.Y.Label.TextStyle.Font.Size = vg.Points(16)
	p.X.Tick.Label.Font.Size = vg.Points(14)
	p.Y.Tick.Label.Font.Size = vg.Points(14)
	p.Legend.TextStyle.Font.Size = vg.Points(14)
}

// graph file generation

func saveGraphImage(graphName string, p *plot.Plot, width, height vg.Length) error {
	err := os.MkdirAll("graphs", 0755)
	if err != nil {
		fmt.Println("Could not create graphs directory:", err)
		return err
	}

	err = p.Save(width, height, "graphs/"+graphName+".png")
	if err != nil {
		fmt.Println("Could not save graph:", err)
		return err
	}

	return nil
}

// hex utilities

func hexDigitValue(c byte) int {
	if c >= '0' && c <= '9' {
		return int(c - '0')
	}
	return int(c-'a') + 10
}

func hexToBits(hexStr string) []int {
	bits := make([]int, len(hexStr)*4)
	for i, c := range hexStr {
		nibble := hexDigitValue(byte(c))
		bits[i*4+0] = (nibble >> 3) & 1
		bits[i*4+1] = (nibble >> 2) & 1
		bits[i*4+2] = (nibble >> 1) & 1
		bits[i*4+3] = nibble & 1
	}
	return bits
}
