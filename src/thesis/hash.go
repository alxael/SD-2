package main

import (
	"encoding/binary"
	"math/bits"
	"sync"
)

const stateSize = 32
const stateStep = 16

type State struct {
	data [32]uint32
}

func (state *State) diffuse() {
	for half := 0; half < 2; half++ {
		base := half * stateStep
		for index := 0; index < stateStep; index++ {
			next := (index + 1) % stateStep
			state.data[base+index] ^= bits.RotateLeft32(state.data[base+next], 13)
		}
	}
	for index := 0; index < stateStep; index++ {
		state.data[index] ^= bits.RotateLeft32(state.data[stateStep+index], 7)
	}
}

func (state *State) chaos() {
	for index := 0; index < stateStep; index++ {
		bIndex := index + stateStep

		x := state.data[index]
		y := state.data[bIndex]

		for round := 0; round < 20; round++ {
			// tent map in integer arithmetic
			if x < 1<<31 {
				x = x << 1
				y = y >> 1
			} else {
				x = ^x << 1
				y = ^y>>1 | 1<<31
			}
			// cross-coupling
			x ^= bits.RotateLeft32(y, 7)
			y ^= bits.RotateLeft32(x, 13)
		}

		state.data[index] = x
		state.data[bIndex] = y

		// feed forward into the hidden half only
		if index+1 < stateStep {
			state.data[index+1+stateStep] ^= bits.RotateLeft32(x, 9)
			state.data[index+1+stateStep] ^= bits.RotateLeft32(y, 17)
		}
	}
	state.diffuse()
	state.diffuse()
}

func (state *State) squeeze() []byte {
	out := make([]byte, stateStep*4)
	for index := range stateStep {
		binary.BigEndian.PutUint32(out[index*4:], state.data[index])
	}
	return out
}

func pad(data []byte, size int) []byte {
	// 10*1 padding: append 0x01, zeros, 0x01 to fill to size boundary
	padLen := size - 1 - (len(data) % size)
	if padLen < 1 {
		padLen += size
	}
	padded := make([]byte, len(data)+1+padLen)
	copy(padded, data)
	padded[len(data)] = 0x01
	padded[len(padded)-1] = 0x01
	return padded
}

func absorbChunk(chunk []byte) State {
	bitrate := stateStep * 4
	var state State
	blockValues := make([]uint32, stateStep)
	for blockStart := 0; blockStart < len(chunk); blockStart += bitrate {
		block := chunk[blockStart : blockStart+bitrate]
		for index := range stateStep {
			blockValues[index] = binary.BigEndian.Uint32(block[index*4 : index*4+4])
		}
		for index := range stateStep {
			state.data[index] ^= blockValues[index]
		}
		state.chaos()
	}
	return state
}

func hash(input []byte, outputSize int, sectionCount int) []byte {
	bitrate := stateStep * 4

	padded := pad(input, bitrate)

	// clamp sectionCount: at least 1, at most one chunk per bitrate block
	maxSections := len(padded) / bitrate
	if sectionCount < 1 {
		sectionCount = 1
	}
	if sectionCount > maxSections {
		sectionCount = maxSections
	}

	// each chunk must be a multiple of bitrate
	blocksPerSection := maxSections / sectionCount
	chunkSize := blocksPerSection * bitrate

	chunkStates := make([]State, sectionCount)
	var waitGroup sync.WaitGroup
	for index := range sectionCount {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			start := index * chunkSize
			end := start + chunkSize
			if index == sectionCount-1 {
				end = len(padded) // last thread absorbs any remainder
			}
			chunkStates[index] = absorbChunk(padded[start:end])
		}(index)
	}
	waitGroup.Wait()

	stateBytes := make([]byte, sectionCount*stateSize*4)
	for i, s := range chunkStates {
		off := i * stateSize * 4
		for j, word := range s.data {
			binary.BigEndian.PutUint32(stateBytes[off+j*4:], word)
		}
	}

	paddedStateBytes := pad(stateBytes, bitrate)
	var finalState State
	blockValues := make([]uint32, stateStep)
	for blockStart := 0; blockStart < len(paddedStateBytes); blockStart += bitrate {
		block := paddedStateBytes[blockStart : blockStart+bitrate]
		for i := range stateStep {
			blockValues[i] = binary.BigEndian.Uint32(block[i*4 : i*4+4])
		}
		for i := range stateStep {
			finalState.data[i] ^= blockValues[i]
		}
		finalState.chaos()
	}

	intermediateRounds := 9 + (len(input)%37+outputSize%31)%11
	for round := 0; round < intermediateRounds; round++ {
		finalState.chaos()
	}

	// squeeze phase
	output := make([]byte, 0, outputSize)
	for len(output) < outputSize {
		output = append(output, finalState.squeeze()...)
		if len(output) < outputSize {
			finalState.chaos()
		}
	}
	return output[:outputSize]
}
