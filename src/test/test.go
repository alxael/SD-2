package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"
)

// duffing map with a = 2.75, b = 0.15
// state size has to be prime
var sectionCount int64 = 128
var stateSize = 91
var stateStep = 43

var roundCount = 3
var duffingMapA = float64(2.75)
var duffingMapB = float64(0.15)

var bitrate int64 = int64(stateStep) * 4
var outputBlocks int = 1
var inputBufferBlocks int = 1024

var intermediateRounds int = 16

// the input buffer blocks limits the memory consumption
// the higher the input buffer size the larger the memory consumtion

type State struct {
	data [91]uint32
}

type SectionState struct {
	state State
	index int
}

type SectionConfiguration struct {
	index  int
	size   int64
	offset int64
	file   string
}

func (state *State) chaos() {
	runRounds := func(firstIndex int, secondIndex int) {
		currentX := float64(state.data[firstIndex])
		currentY := float64(state.data[secondIndex])

		for roundIndex := 0; roundIndex < roundCount; roundIndex++ {
			nextX := currentY

			partOne := float64(uint32(-duffingMapB * currentX))
			partTwo := float64(uint32(duffingMapA * currentY))
			partThree := float64(uint32(float64(uint32(currentY*currentY)) * currentY))
			nextY := partOne + partTwo + partThree

			currentX = float64(uint32(nextX))
			currentY = float64(uint32(nextY))
		}

		state.data[firstIndex] = uint32(currentX)
		state.data[secondIndex] = uint32(currentY)
	}

	for firstIndex := 0; firstIndex < stateSize; firstIndex++ {
		secondIndex := ((firstIndex + roundCount) * stateStep) % stateSize
		if firstIndex%2 == 0 {
			runRounds(firstIndex, secondIndex)
		} else {
			runRounds(secondIndex, firstIndex)
		}

	}
}

func (state *State) squeeze() []byte {
	allBytes := make([]byte, 0, stateSize*4)
	for _, word := range state.data {
		wordBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(wordBytes, word)
		allBytes = append(allBytes, wordBytes...)
	}

	return allBytes[:bitrate]
}

func fileSizeInBytes(filename string) int64 {
	info, err := os.Stat(filename)
	if err != nil {
		panic("Could not open file!")
	}
	return info.Size()
}

func processSection(config SectionConfiguration) SectionState {
	file, err := os.Open(config.file)
	if err != nil {
		panic("Could not open file!")
	}
	defer file.Close()

	emptyBlockFound := false
	bufferSize := int(bitrate) * inputBufferBlocks
	buffer := make([]byte, bufferSize)
	bitrateValues := make([]uint32, bitrate/4)
	chunkCount := config.size / int64(inputBufferBlocks)
	blockIndex := 0
	var chunk []byte

	state := State{
		data: [91]uint32{},
	}

	for chunkIndex := 0; chunkIndex < int(chunkCount); chunkIndex++ {
		offset := config.offset + int64(chunkIndex)*int64(bufferSize)
		readBytes, err := file.ReadAt(buffer, offset)
		if err != nil && err != io.EOF {
			panic("Could not read from file!")
		}

		for index := readBytes; index < bufferSize; index++ {
			buffer[index] = 0
		}

		// introduce pading 10*1
		if readBytes != int(bufferSize) && !emptyBlockFound {
			buffer[readBytes] = 1
			emptyBlockFound = true
		}
		if readBytes != int(bufferSize) && chunkIndex == int(chunkCount)-1 && config.index == int(sectionCount)-1 {
			buffer[bufferSize-1] = 1
		}

		for blockIndex*int(bitrate) < readBytes && blockIndex < int(config.size) {
			chunk = buffer[blockIndex*int(bitrate) : (blockIndex+1)*int(bitrate)]
			for index := 0; index < len(bitrateValues); index++ {
				bitrateValues[index] = binary.BigEndian.Uint32(chunk[index*4 : index*4+4])
			}

			// xor values into state
			for index := 0; index < len(bitrateValues); index++ {
				state.data[index] = state.data[index] ^ bitrateValues[index]
			}

			// introduce chaos
			state.chaos()

			blockIndex++
		}
	}

	return SectionState{
		state: state,
		index: config.index,
	}
}

func processSectionResults(results []SectionState) State {
	state := State{
		data: [91]uint32{},
	}

	valueBytes := make([]byte, 4)
	allStates := make([]byte, 0)
	for _, result := range results {
		for _, value := range result.state.data {
			binary.BigEndian.PutUint32(valueBytes, value)
			allStates = append(allStates, valueBytes...)
		}
	}

	// introduce pading 10*1
	targetLength := ((int64(len(allStates)) + bitrate - 1) / bitrate) * bitrate
	if len(allStates) != int(targetLength) {
		remainingBytes := make([]byte, int(targetLength)-len(allStates))

		remainingBytes[0] = 1
		remainingBytes[int(targetLength)-len(allStates)-1] = 1

		allStates = append(allStates, remainingBytes...)
	}

	bitrateValues := make([]uint32, bitrate/4)
	for blockIndex := 0; blockIndex < len(allStates)/int(bitrate); blockIndex++ {
		offset := blockIndex * int(bitrate)

		// convert bytes to uint32
		for index := 0; index < len(bitrateValues); index++ {
			bitrateValues[index] = binary.BigEndian.Uint32(allStates[offset+index*4 : offset+index*4+4])
		}

		// xor values into state
		for index := 0; index < len(bitrateValues); index++ {
			state.data[index] = state.data[index] ^ bitrateValues[index]
		}

		// introduce chaos
		state.chaos()
	}

	// apply intermediate rounds
	for index := 0; index < intermediateRounds; index++ {
		state.chaos()
	}

	return state
}

func main() {
	file := "/Users/Work/Downloads/Laborator.zip"
	// file := "../src/test/test.txt"
	fileSizeBytes := fileSizeInBytes(file)

	if fileSizeBytes == -1 {
		return
	}

	availableSectionsCount := sectionCount
	availableBlocksCount := (fileSizeBytes + bitrate - 1) / bitrate
	blocksPerSection := (availableBlocksCount + sectionCount - 1) / sectionCount
	sectionSize := blocksPerSection * bitrate
	inputBufferBlocks = min(inputBufferBlocks, int(blocksPerSection))

	if availableBlocksCount < sectionCount {
		availableSectionsCount = availableBlocksCount
	}

	start := time.Now()

	// if the amount of available sections is smaller than the section count
	// then I should skip this whole section and just jump directly to sequentially applying the duffing map

	results := make([]SectionState, 0)
	var waitGroup sync.WaitGroup
	var mutex sync.Mutex

	for sectionIndex := 0; sectionIndex < int(availableSectionsCount); sectionIndex++ {
		sectionOffset := int64(sectionIndex) * sectionSize
		waitGroup.Add(1)
		go func(config SectionConfiguration) {
			defer waitGroup.Done()
			result := processSection(config)
			mutex.Lock()
			results = append(results, result)
			mutex.Unlock()
		}(SectionConfiguration{
			index:  sectionIndex,
			size:   sectionSize,
			offset: sectionOffset * bitrate,
			file:   file,
		})
	}

	fmt.Println(outputBlocks)

	waitGroup.Wait()

	sort.Slice(results, func(first, second int) bool {
		return results[first].index < results[second].index
	})

	state := processSectionResults(results)

	output := make([]byte, 0)
	for index := 0; index < outputBlocks; index++ {
		output = append(output, state.squeeze()...)
		state.chaos()
	}

	hexOutput := hex.EncodeToString(output)

	duration := time.Since(start)
	fileSizeInMegabytes := (float64(fileSizeBytes) / 1024) / 1024
	hashSpeed := fileSizeInMegabytes / duration.Seconds()

	fmt.Println(hexOutput)
	fmt.Printf("Total duration %.2f\n", duration.Seconds())
	fmt.Printf("Speed: %.2f mb/s", hashSpeed)
}
